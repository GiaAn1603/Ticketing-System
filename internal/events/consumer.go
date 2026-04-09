package events

import (
	"Ticketing-System/internal/config"
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
	reader *kafka.Reader
	pgRepo *repositories.PostgresRepo
	cfg    config.ConsumerConfig
	log    *slog.Logger
}

func NewKafkaConsumer(ctx context.Context, pgRepo *repositories.PostgresRepo, cfg config.ConsumerConfig) (*KafkaConsumer, error) {
	logger := infrastructure.GetLogger("KAFKA_CONSUMER")

	if err := utils.EnsureTopicExists(logger, cfg.TopicConfig); err != nil {
		return nil, fmt.Errorf("setup kafka brokers: %w", err)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.TopicConfig.Brokers,
		Topic:       cfg.TopicConfig.Topic,
		GroupID:     cfg.TopicConfig.GroupID,
		MinBytes:    cfg.MinBytes,
		MaxBytes:    cfg.MaxBytes,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info(
		"Connection warming up",
		"topic", cfg.TopicConfig.Topic,
	)

	if conn, err := kafka.DialLeader(ctx, "tcp", cfg.TopicConfig.Brokers[0], cfg.TopicConfig.Topic, 0); err != nil {
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
		"brokers", cfg.TopicConfig.Brokers,
		"topic", cfg.TopicConfig.Topic,
		"group_id", cfg.TopicConfig.GroupID,
	)

	return &KafkaConsumer{
		reader: r,
		pgRepo: pgRepo,
		cfg:    cfg,
		log:    logger,
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

		dbCtx, dbCancel := context.WithTimeout(spanCtx, c.cfg.DBTimeout)
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

			jitter := time.Duration(rand.Intn(c.cfg.BackoffJitter)) * time.Millisecond
			time.Sleep(c.cfg.BackoffBase + jitter)

			continue
		}

		commitCtx, commitCancel := context.WithTimeout(spanCtx, c.cfg.CommitTimeout)
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
