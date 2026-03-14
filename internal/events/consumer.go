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
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader        *kafka.Reader
	pgRepo        *repositories.PostgresRepo
	dbTimeout     time.Duration
	commitTimeout time.Duration
	log           *slog.Logger
}

func NewKafkaConsumer(
	ctx context.Context,
	brokers []string,
	topic, groupID string,
	partitions, replFactor int,
	pgRepo *repositories.PostgresRepo,
	minBytes, maxBytes int,
	kafkaTimeout, dbTimeout, commitTimeout time.Duration,
) (*KafkaConsumer, error) {
	logger := infrastructure.GetLogger("KAFKA_CONSUMER")

	if err := utils.EnsureTopicExists(brokers, topic, partitions, replFactor, kafkaTimeout, logger); err != nil {
		return nil, fmt.Errorf("failed to set up kafka brokers %v for topic %s: %w", brokers, topic, err)
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
			infrastructure.KeyError, err.Error(),
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
			return fmt.Errorf("failed to read message from kafka: %w", err)
		}

		var event models.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.log.Error(
				"Event unmarshal failed",
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
				"event_id", event.EventID,
				"user_id", event.UserID,
				"request_id", event.RequestID,
				"quantity", event.Quantity,
				"order_status", event.Status,
				infrastructure.KeyError, err.Error(),
			)

			continue
		}

		commitCtx, commitCancel := context.WithTimeout(context.Background(), c.commitTimeout)
		if err := c.reader.CommitMessages(commitCtx, msg); err != nil {
			c.log.Error(
				"Message commit failed",
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
		return fmt.Errorf("failed to close kafka consumer: %w", err)
	}

	return nil
}
