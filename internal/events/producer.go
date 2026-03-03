package events

import (
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

func ensureTopicExists(brokers []string, topic string) error {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	conn, err := dialer.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial initial broker at %s: %w", brokers[0], err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("failed to read partitions from broker at %s: %w", brokers[0], err)
	}

	for _, p := range partitions {
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
		NumPartitions:     3,
		ReplicationFactor: 1,
	}

	if err := controllerConn.CreateTopics(topicConfig); err != nil {
		return fmt.Errorf("failed to create topic %s via controller at %s: %w", topic, controllerAddr, err)
	}

	log.Printf("[KAFKA][INFO] Topic created successfully | topic=%s | partitions=%d", topic, topicConfig.NumPartitions)

	return nil
}

func NewKafkaProducer(brokers []string, topic string) (*KafkaProducer, error) {
	if err := ensureTopicExists(brokers, topic); err != nil {
		return nil, fmt.Errorf("kafka setup failed at %v for topic %s: %w", brokers, topic, err)
	}

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
	}, nil
}

func (p *KafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka producer: %w", err)
	}

	return nil
}
