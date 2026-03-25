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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	pgRepo *repositories.PostgresRepo,
	brokers []string,
	topic, groupID string,
	partitions, replFactor, minBytes, maxBytes, backoffJitter int,
	kafkaTimeout, dbTimeout, commitTimeout, backoffBase time.Duration,
) (*KafkaConsumer, error) {
	logger := infrastructure.GetLogger("KAFKA_CONSUMER")

	if err := utils.EnsureTopicExists(logger, brokers, topic, partitions, replFactor, kafkaTimeout); err != nil {
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
				"kafka_key", string(msg.Key),
				infrastructure.KeyError, err.Error(),
			)

			c.reader.CommitMessages(ctx, msg)
			continue
		}

		ctxWithTrace := otel.GetTextMapPropagator().Extract(context.Background(), &KafkaHeaderPropagator{Headers: &msg.Headers})

		tr := otel.Tracer("ticket-consumer")
		spanCtx, span := tr.Start(ctxWithTrace, "consume_kafka_msg")
		span.SetAttributes(
			attribute.String("event_id", event.EventID),
			attribute.String("user_id", event.UserID),
			attribute.String("request_id", event.RequestID),
		)

		c.log.Info(
			"OrderEvent consumed successfully",
			"event_id", event.EventID,
			"user_id", event.UserID,
			"request_id", event.RequestID,
			"quantity", event.Quantity,
		)

		event.Status = "Success"

		dbCtx, dbCancel := context.WithTimeout(spanCtx, c.dbTimeout)
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

			span.End()

			jitter := time.Duration(rand.Intn(c.backoffJitter)) * time.Millisecond
			time.Sleep(c.backoffBase + jitter)

			continue
		}

		commitCtx, commitCancel := context.WithTimeout(spanCtx, c.commitTimeout)
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
		span.End()
	}
}

func (c *KafkaConsumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close consumer: %w", err)
	}

	return nil
}
