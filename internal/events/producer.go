package events

import (
	"Ticketing-System/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func ensureTopicExists(brokers []string, topic string, partitions, replicationFactor int, timeout time.Duration) error {
	dialer := &kafka.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial initial broker at %s: %w", brokers[0], err)
	}
	defer conn.Close()

	topicPartitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("failed to read partitions from broker at %s: %w", brokers[0], err)
	}

	for _, p := range topicPartitions {
		if p.Topic == topic {
			log.Printf("[KAFKA][INFO] Topic already exists | topic=%s", topic)
			return nil
		}
	}

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller from broker at %s: %w", brokers[0], err)
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := dialer.Dial("tcp", controllerAddr)
	if err != nil {
		return fmt.Errorf("failed to dial controller at %s: %w", controllerAddr, err)
	}
	defer controllerConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
	}

	if err := controllerConn.CreateTopics(topicConfig); err != nil {
		return fmt.Errorf("failed to create topic %s via controller at %s: %w", topic, controllerAddr, err)
	}

	log.Printf("[KAFKA][INFO] Topic created successfully | topic=%s | partitions=%d", topic, topicConfig.NumPartitions)

	return nil
}

func NewKafkaProducer(ctx context.Context, brokers []string, topic string, partitions, replFactor, batchSize int, batchTimeout, kafkaTimeout time.Duration) (*KafkaProducer, error) {
	if err := ensureTopicExists(brokers, topic, partitions, replFactor, kafkaTimeout); err != nil {
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
