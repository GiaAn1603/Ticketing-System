package config

import "time"

type TracerConfig struct {
	MaxQueue      int
	MaxBatch      int
	TraceRatio    float64
	BatchTimeout  time.Duration
	ExportTimeout time.Duration
}

type RedisConfig struct {
	Addr     string
	PoolSize int
	MinIdle  int
}

type DBConfig struct {
	Addr        string
	User        string
	Password    string
	DBName      string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
}

type CircuitBreakerConfig struct {
	MaxReq    uint32
	MinReq    uint32
	FailRatio float64
	Interval  time.Duration
	Timeout   time.Duration
}

type KafkaTopicConfig struct {
	Brokers           []string
	Topic             string
	GroupID           string
	Partitions        int
	ReplicationFactor int
	Timeout           time.Duration
}

type ProducerConfig struct {
	TopicConfig  KafkaTopicConfig
	BatchSize    int
	BatchTimeout time.Duration
	KafkaTimeout time.Duration
}

type ConsumerConfig struct {
	TopicConfig   KafkaTopicConfig
	MinBytes      int
	MaxBytes      int
	BackoffJitter int
	KafkaTimeout  time.Duration
	DBTimeout     time.Duration
	CommitTimeout time.Duration
	BackoffBase   time.Duration
}

type RateLimiterConfig struct {
	CBConfig      CircuitBreakerConfig
	Capacity      int
	Rate          int
	BannedMaxSize int
	Timeout       time.Duration
	BannedTTL     time.Duration
}

type RedisRepoConfig struct {
	Addr       string
	CBConfig   CircuitBreakerConfig
	HistoryTTL int
}

type TicketServiceConfig struct {
	SoldOutMaxSize  int
	SoldOutTTL      time.Duration
	RollbackTimeout time.Duration
	RedisTimeout    time.Duration
}

type OrderServiceConfig struct {
	DBTimeout     time.Duration
	BackoffBase   time.Duration
	BackoffJitter int
}
