package events

import (
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: false,
		BatchSize:              100,
		BatchTimeout:           10 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
		WriteTimeout:           5 * time.Second,
		ReadTimeout:            5 * time.Second,
	}

	log.Printf("[KAFKA][INFO] Producer initialized | brokers=%v | topic=%s", brokers, topic)

	return &KafkaProducer{
		writer: w,
	}
}

func (p *KafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka producer: %w", err)
	}

	return nil
}
