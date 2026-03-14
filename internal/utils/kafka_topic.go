package utils

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

func EnsureTopicExists(brokers []string, topic string, partitions, replicationFactor int, timeout time.Duration) error {
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
