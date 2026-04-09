package utils

import (
	"Ticketing-System/internal/config"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func EnsureTopicExists(logger *slog.Logger, cfg config.KafkaTopicConfig) error {
	dialer := &kafka.Dialer{
		Timeout:   cfg.Timeout,
		DualStack: true,
	}

	conn, err := dialer.Dial("tcp", cfg.Brokers[0])
	if err != nil {
		return fmt.Errorf("dial initial broker: %w", err)
	}
	defer conn.Close()

	topicPartitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("read partitions: %w", err)
	}

	for _, p := range topicPartitions {
		if p.Topic == cfg.Topic {
			logger.Info(
				"Topic already exists",
				"topic", cfg.Topic,
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
		Topic:             cfg.Topic,
		NumPartitions:     cfg.Partitions,
		ReplicationFactor: cfg.ReplicationFactor,
	}

	if err := controllerConn.CreateTopics(topicConfig); err != nil {
		return fmt.Errorf("create topic: %w", err)
	}

	logger.Info(
		"Topic created successfully",
		"topic", cfg.Topic,
		"num_partitions", topicConfig.NumPartitions,
	)

	return nil
}
