package config

import "time"

const (
	ServerStartupTimeout  = 5 * time.Second
	ServerReadTimeout     = 5 * time.Second
	ServerWriteTimeout    = 15 * time.Second
	ServerIdleTimeout     = 15 * time.Second
	ServerShutdownTimeout = 5 * time.Second
)

const (
	RedisPoolSize     = 1000
	RedisMinIdleConns = 200
	HistoryTTLSeconds = 86400
	RedisTimeout      = 500 * time.Millisecond
)

const (
	DBMaxOpenConns       = 200
	DBMaxIdleConns       = 50
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
	KafkaProducerBatchSize    = 10000
	KafkaProducerBatchTimeout = 100 * time.Millisecond
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

const (
	CBMaxRequests  = 5
	CBMinRequests  = 10
	CBFailureRatio = 0.5
	CBInterval     = 10 * time.Second
	CBTimeout      = 15 * time.Second
)

const (
	CacheSoldOutMaxSize  = 1000
	CacheSoldOutTTL      = 1 * time.Minute
	CacheBannedIPMaxSize = 10000
	CacheBannedIPTTL     = 2 * time.Second
)

const (
	OtelBatchMaxQueueSize  = 32768
	OtelBatchMaxExportSize = 4096
	OtelBatchTimeout       = 5 * time.Second
	OtelExportTimeout      = 30 * time.Second
	OtelTraceRatio         = 0.05
)
