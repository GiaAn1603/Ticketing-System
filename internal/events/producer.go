package events

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type KafkaHeaderPropagator struct {
	Headers *[]kafka.Header
}

type KafkaProducer struct {
	writer *kafka.Writer
	log    *slog.Logger
}

func (k *KafkaHeaderPropagator) Get(key string) string {
	for _, h := range *k.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}

	return ""
}

func (k *KafkaHeaderPropagator) Set(key, val string) {
	*k.Headers = append(*k.Headers, kafka.Header{Key: key, Value: []byte(val)})
}

func (k *KafkaHeaderPropagator) Keys() []string {
	keys := make([]string, len(*k.Headers))

	for i, h := range *k.Headers {
		keys[i] = h.Key
	}

	return keys
}

func NewKafkaProducer(
	ctx context.Context,
	brokers []string,
	topic string,
	partitions, replFactor, batchSize int,
	kafkaTimeout, batchTimeout time.Duration,
) (*KafkaProducer, error) {
	logger := infrastructure.GetLogger("KAFKA_PRODUCER")

	if err := utils.EnsureTopicExists(logger, brokers, topic, partitions, replFactor, kafkaTimeout); err != nil {
		return nil, fmt.Errorf("setup kafka brokers: %w", err)
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

	logger.Info(
		"Connection warming up",
		"topic", topic,
	)

	if conn, err := kafka.DialLeader(ctx, "tcp", brokers[0], topic, 0); err != nil {
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
		"Producer initialized successfully",
		"brokers", brokers,
		"topic", topic,
	)

	return &KafkaProducer{
		writer: w,
		log:    logger,
	}, nil
}

func (p *KafkaProducer) PublishOrderEvent(ctx context.Context, event models.OrderEvent) error {
	tr := otel.Tracer("ticket-producer")
	ctx, span := tr.Start(ctx, "publish_kafka_event")
	span.SetAttributes(
		attribute.String("event_id", event.EventID),
		attribute.String("user_id", event.UserID),
		attribute.String("request_id", event.RequestID),
	)
	defer span.End()

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:     []byte(event.RequestID),
		Value:   payload,
		Time:    time.Now(),
		Headers: []kafka.Header{},
	}

	otel.GetTextMapPropagator().Inject(ctx, &KafkaHeaderPropagator{Headers: &msg.Headers})

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	p.log.Debug(
		"OrderEvent published successfully",
		"event_id", event.EventID,
		"user_id", event.UserID,
		"request_id", event.RequestID,
		"quantity", event.Quantity,
		"order_status", event.Status,
	)

	return nil
}

func (p *KafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close producer: %w", err)
	}

	return nil
}
