package events

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/repositories"
	"Ticketing-System/internal/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader        *kafka.Reader
	pgRepo        *repositories.PostgresRepo
	dbTimeout     time.Duration
	commitTimeout time.Duration
	backoffBase   time.Duration
	backoffJitter int
	log           *slog.Logger
}

func NewKafkaConsumer(
	ctx context.Context,
	brokers []string,
	topic, groupID string,
	partitions, replFactor int,
	pgRepo *repositories.PostgresRepo,
	minBytes, maxBytes int,
	kafkaTimeout, dbTimeout, commitTimeout, backoffBase time.Duration,
	backoffJitter int,
) (*KafkaConsumer, error) {
	logger := infrastructure.GetLogger("KAFKA_CONSUMER")

	if err := utils.EnsureTopicExists(brokers, topic, partitions, replFactor, kafkaTimeout, logger); err != nil {
		return nil, fmt.Errorf("setup kafka brokers: %w", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info(
		"Connection warming up",
		"topic", topic,
	)

	if conn, err := kafka.DialLeader(ctx, "tcp", brokers[0], topic, 0); err != nil {
		logger.Warn(
			"Warm-up connection failed",
			"error", err.Error(),
		)
	} else {
		conn.Close()

		logger.Info(
			"Warm-up connection completed successfully",
		)
	}

	logger.Info(
		"Consumer initialized successfully",
		"brokers", brokers,
		"topic", topic,
		"group_id", groupID,
	)

	return &KafkaConsumer{
		reader:        r,
		pgRepo:        pgRepo,
		dbTimeout:     dbTimeout,
		commitTimeout: commitTimeout,
		backoffBase:   backoffBase,
		backoffJitter: backoffJitter,
		log:           logger,
	}, nil
}

func (c *KafkaConsumer) ConsumeOrderEvent(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				c.log.Info("Context cancelled")
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		var event models.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.log.Error(
				"Event unmarshal failed",
				infrastructure.KeyAction, "unmarshal_kafka_msg",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				"kafka_key", string(msg.Key),
				infrastructure.KeyError, err.Error(),
			)

			c.reader.CommitMessages(ctx, msg)
			continue
		}

		c.log.Info(
			"OrderEvent consumed successfully",
			"event_id", event.EventID,
			"user_id", event.UserID,
			"request_id", event.RequestID,
			"quantity", event.Quantity,
		)

		event.Status = "Success"

		dbCtx, dbCancel := context.WithTimeout(context.Background(), c.dbTimeout)
		err = c.pgRepo.InsertOrderIfNotExists(dbCtx, event)
		dbCancel()

		if err != nil {
			c.log.Error(
				"Order insertion failed",
				infrastructure.KeyAction, "insert_order_db",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				"event_id", event.EventID,
				"user_id", event.UserID,
				"request_id", event.RequestID,
				"quantity", event.Quantity,
				"order_status", event.Status,
				infrastructure.KeyError, err.Error(),
			)

			jitter := time.Duration(rand.Intn(c.backoffJitter)) * time.Millisecond
			time.Sleep(c.backoffBase + jitter)

			continue
		}

		commitCtx, commitCancel := context.WithTimeout(context.Background(), c.commitTimeout)
		if err := c.reader.CommitMessages(commitCtx, msg); err != nil {
			c.log.Error(
				"Message commit failed",
				infrastructure.KeyAction, "commit_kafka_msg",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				"event_id", event.EventID,
				"user_id", event.UserID,
				"request_id", event.RequestID,
				"quantity", event.Quantity,
				"order_status", event.Status,
				infrastructure.KeyError, err.Error(),
			)
		}
		commitCancel()
	}
}

func (c *KafkaConsumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close consumer: %w", err)
	}

	return nil
}
