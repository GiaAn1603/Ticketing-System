package events

import (
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(
	ctx context.Context,
	brokers []string,
	topic string,
	partitions, replFactor, batchSize int,
	batchTimeout, kafkaTimeout time.Duration,
) (*KafkaProducer, error) {
	if err := utils.EnsureTopicExists(brokers, topic, partitions, replFactor, kafkaTimeout); err != nil {
		return nil, fmt.Errorf("failed to set up kafka brokers %v for topic %s: %w", brokers, topic, err)
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: false,
		BatchSize:              batchSize,
		BatchTimeout:           batchTimeout,
		RequiredAcks:           kafka.RequireAll,
		WriteTimeout:           kafkaTimeout,
		ReadTimeout:            kafkaTimeout,
	}

	log.Printf("[KAFKA][INFO] Warming up connection | topic=%s", topic)

	if conn, err := kafka.DialLeader(ctx, "tcp", brokers[0], topic, 0); err != nil {
		log.Printf("[KAFKA][WARN] Warm-up connection failed | err=%v", err)
	} else {
		conn.Close()
		log.Println("[KAFKA][INFO] Warm-up connection successful")
	}

	log.Printf("[KAFKA][INFO] Producer initialized | brokers=%v | topic=%s", brokers, topic)

	return &KafkaProducer{
		writer: w,
	}, nil
}

func (p *KafkaProducer) PublishOrderEvent(ctx context.Context, event models.OrderEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event %s: %w", event.RequestID, err)
	}

	msg := kafka.Message{
		Key:   []byte(event.RequestID),
		Value: payload,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to kafka for request %s: %w", event.RequestID, err)
	}

	log.Printf("[KAFKA][INFO] Published OrderEvent | event_id=%s | user_id=%s | req_id=%s | qty=%d", event.EventID, event.UserID, event.RequestID, event.Quantity)

	return nil
}

func (p *KafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka producer: %w", err)
	}

	return nil
}
