package config

import "time"

const (
	ServerStartupTimeout  = 5 * time.Second
	ServerReadTimeout     = 5 * time.Second
	ServerWriteTimeout    = 10 * time.Second
	ServerIdleTimeout     = 15 * time.Second
	ServerShutdownTimeout = 5 * time.Second
)

const (
	HistoryTTLSeconds = 86400
	RedisTimeout      = 500 * time.Millisecond
)

const (
	DBMaxOpenConns       = 25
	DBMaxIdleConns       = 10
	DBConnMaxLifetime    = 5 * time.Minute
	DBTimeout            = 5 * time.Second
	DBRetryBackoffBase   = 1 * time.Second
	DBRetryBackoffJitter = 1000
)

const (
	KafkaTopicOrders          = "orders"
	KafkaGroupID              = "ticket_worker_group"
	KafkaNumPartitions        = 3
	KafkaReplicationFactor    = 1
	KafkaProducerBatchSize    = 100
	KafkaProducerBatchTimeout = 10 * time.Millisecond
	KafkaConsumerMinBytes     = 10e3
	KafkaConsumerMaxBytes     = 10e6
	KafkaTimeout              = 5 * time.Second
	KafkaCommitTimeout        = 2 * time.Second
)

const (
	RateLimitCapacity = 10
	RateLimitRate     = 5
	RateLimitTimeout  = 500 * time.Millisecond
)
