package utils

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

func EnsureTopicExists(
	logger *slog.Logger,
	brokers []string,
	topic string,
	partitions, replicationFactor int,
	timeout time.Duration,
) error {
	dialer := &kafka.Dialer{
		Timeout:   timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial initial broker: %w", err)
	}
	defer conn.Close()

	topicPartitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("read partitions: %w", err)
	}

	for _, p := range topicPartitions {
		if p.Topic == topic {
			logger.Info(
				"Topic already exists",
				"topic", topic,
			)

			return nil
		}
	}

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get controller: %w", err)
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := dialer.Dial("tcp", controllerAddr)
	if err != nil {
		return fmt.Errorf("dial controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
	}

	if err := controllerConn.CreateTopics(topicConfig); err != nil {
		return fmt.Errorf("create topic: %w", err)
	}

	logger.Info(
		"Topic created successfully",
		"topic", topic,
		"num_partitions", topicConfig.NumPartitions,
	)

	return nil
}
