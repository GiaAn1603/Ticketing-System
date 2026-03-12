package events

import (
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/repositories"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
	pgRepo *repositories.PostgresRepo
}

func NewKafkaConsumer(ctx context.Context, brokers []string, topic, groupID string, pgRepo *repositories.PostgresRepo) (*KafkaConsumer, error) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})

	log.Printf("[KAFKA][INFO] Warming up connection | topic=%s", topic)

	if conn, err := kafka.DialLeader(ctx, "tcp", brokers[0], topic, 0); err != nil {
		log.Printf("[KAFKA][WARN] Warm-up connection failed | err=%v", err)
	} else {
		conn.Close()
		log.Println("[KAFKA][INFO] Warm-up connection successful")
	}

	log.Printf("[KAFKA][INFO] Consumer initialized | brokers=%v | topic=%s | group_id=%s", brokers, topic, groupID)

	return &KafkaConsumer{
		reader: r,
		pgRepo: pgRepo,
	}, nil
}

func (c *KafkaConsumer) ConsumeOrderEvent(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				log.Println("[KAFKA][INFO] Context cancelled | action=stop_consumer")
				return nil
			}
			return fmt.Errorf("failed to read message from kafka: %w", err)
		}

		var event models.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[KAFKA][ERROR] Failed to unmarshal event | key=%s | err=%v", string(msg.Key), err)
			c.reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[KAFKA][INFO] Consumed OrderEvent | event_id=%s | req_id=%s", event.EventID, event.RequestID)

		event.Status = "Success"

		dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = c.pgRepo.InsertOrderIfNotExists(dbCtx, event)
		dbCancel()

		if err != nil {
			log.Printf("[KAFKA][ERROR] Failed to insert order to database | req_id=%s | err=%v", event.RequestID, err)
			continue
		}

		commitCtx, commitCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.reader.CommitMessages(commitCtx, msg); err != nil {
			log.Printf("[KAFKA][ERROR] Failed to commit message | req_id=%s | err=%v", event.RequestID, err)
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
