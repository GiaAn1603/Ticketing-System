package events

import (
	"Ticketing-System/internal/repositories"
	"context"
	"fmt"
	"log"

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

func (c *KafkaConsumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close kafka consumer: %w", err)
	}

	return nil
}
