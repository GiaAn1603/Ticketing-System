package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort           string
	RedisAddr            string
	KafkaAddr            string
	PostgresAddr         string
	PostgresUser         string
	PostgresPassword     string
	PostgresDB           string
	OtelExporterEndpoint string

	ServerStartupTimeout  time.Duration
	ServerReadTimeout     time.Duration
	ServerWriteTimeout    time.Duration
	ServerIdleTimeout     time.Duration
	ServerShutdownTimeout time.Duration

	RedisPoolSize     int
	RedisMinIdleConns int
	HistoryTTLSeconds int
	RedisTimeout      time.Duration

	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBRetryBackoffJitter int
	DBConnMaxLifetime    time.Duration
	DBTimeout            time.Duration
	DBRetryBackoffBase   time.Duration

	KafkaTopicOrders          string
	KafkaGroupID              string
	KafkaNumPartitions        int
	KafkaReplicationFactor    int
	KafkaProducerBatchSize    int
	KafkaConsumerMinBytes     int
	KafkaConsumerMaxBytes     int
	KafkaProducerBatchTimeout time.Duration
	KafkaTimeout              time.Duration
	KafkaCommitTimeout        time.Duration

	RateLimitCapacity int
	RateLimitRate     int
	RateLimitTimeout  time.Duration

	CBMaxRequests  uint32
	CBMinRequests  uint32
	CBFailureRatio float64
	CBInterval     time.Duration
	CBTimeout      time.Duration

	CacheSoldOutMaxSize  int
	CacheBannedIPMaxSize int
	CacheSoldOutTTL      time.Duration
	CacheBannedIPTTL     time.Duration

	OtelBatchMaxQueueSize  int
	OtelBatchMaxExportSize int
	OtelTraceRatio         float64
	OtelBatchTimeout       time.Duration
	OtelExportTimeout      time.Duration
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func validateConfig(cfg *Config) error {
	if cfg.PostgresUser == "" || cfg.PostgresPassword == "" {
		return errors.New("missing db credentials")
	}

	return nil
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(
			"Env file load failed",
			"layer", "CONFIG",
			"error", err.Error(),
		)
	} else {
		slog.Info(
			"Env file loaded successfully",
			"layer", "CONFIG",
			"path", ".env",
		)
	}

	cfg := &Config{
		ServerPort:           getEnv("SERVER_PORT", ":8080"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaAddr:            getEnv("KAFKA_ADDR", "localhost:9092"),
		PostgresAddr:         getEnv("POSTGRES_ADDR", "localhost:5432"),
		PostgresUser:         os.Getenv("POSTGRES_USER"),
		PostgresPassword:     os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:           getEnv("POSTGRES_DB", "ticket_db"),
		OtelExporterEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),

		ServerStartupTimeout:  ServerStartupTimeout,
		ServerReadTimeout:     ServerReadTimeout,
		ServerWriteTimeout:    ServerWriteTimeout,
		ServerIdleTimeout:     ServerIdleTimeout,
		ServerShutdownTimeout: ServerShutdownTimeout,

		RedisPoolSize:     RedisPoolSize,
		RedisMinIdleConns: RedisMinIdleConns,
		HistoryTTLSeconds: HistoryTTLSeconds,
		RedisTimeout:      RedisTimeout,

		DBMaxOpenConns:       DBMaxOpenConns,
		DBMaxIdleConns:       DBMaxIdleConns,
		DBRetryBackoffJitter: DBRetryBackoffJitter,
		DBConnMaxLifetime:    DBConnMaxLifetime,
		DBTimeout:            DBTimeout,
		DBRetryBackoffBase:   DBRetryBackoffBase,

		KafkaTopicOrders:          KafkaTopicOrders,
		KafkaGroupID:              KafkaGroupID,
		KafkaNumPartitions:        KafkaNumPartitions,
		KafkaReplicationFactor:    KafkaReplicationFactor,
		KafkaProducerBatchSize:    KafkaProducerBatchSize,
		KafkaConsumerMinBytes:     KafkaConsumerMinBytes,
		KafkaConsumerMaxBytes:     KafkaConsumerMaxBytes,
		KafkaProducerBatchTimeout: KafkaProducerBatchTimeout,
		KafkaTimeout:              KafkaTimeout,
		KafkaCommitTimeout:        KafkaCommitTimeout,

		RateLimitCapacity: RateLimitCapacity,
		RateLimitRate:     RateLimitRate,
		RateLimitTimeout:  RateLimitTimeout,

		CBMaxRequests:  CBMaxRequests,
		CBMinRequests:  CBMinRequests,
		CBFailureRatio: CBFailureRatio,
		CBInterval:     CBInterval,
		CBTimeout:      CBTimeout,

		CacheSoldOutMaxSize:  CacheSoldOutMaxSize,
		CacheBannedIPMaxSize: CacheBannedIPMaxSize,
		CacheSoldOutTTL:      CacheSoldOutTTL,
		CacheBannedIPTTL:     CacheBannedIPTTL,

		OtelBatchMaxQueueSize:  OtelBatchMaxQueueSize,
		OtelBatchMaxExportSize: OtelBatchMaxExportSize,
		OtelTraceRatio:         OtelTraceRatio,
		OtelBatchTimeout:       OtelBatchTimeout,
		OtelExportTimeout:      OtelExportTimeout,
	}

	if err := validateConfig(cfg); err != nil {
		slog.Error(
			"Config validation failed",
			"layer", "CONFIG",
			"error", err.Error(),
		)

		return nil, fmt.Errorf("validate config: %w", err)
	}

	slog.Info(
		"Config loaded successfully",
		"layer", "CONFIG",
		"server_port", cfg.ServerPort,
		"redis_addr", cfg.RedisAddr,
		"kafka_addr", cfg.KafkaAddr,
		"pg_addr", cfg.PostgresAddr,
		"otel_endpoint", cfg.OtelExporterEndpoint,
	)

	return cfg, nil
}

func (c *Config) ToTracerConfig() TracerConfig {
	return TracerConfig{
		MaxQueue:      c.OtelBatchMaxQueueSize,
		MaxBatch:      c.OtelBatchMaxExportSize,
		TraceRatio:    c.OtelTraceRatio,
		BatchTimeout:  c.OtelBatchTimeout,
		ExportTimeout: c.OtelExportTimeout,
	}
}

func (c *Config) ToRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:     c.RedisAddr,
		PoolSize: c.RedisPoolSize,
		MinIdle:  c.RedisMinIdleConns,
	}
}

func (c *Config) ToDBConfig() DBConfig {
	return DBConfig{
		Addr:        c.PostgresAddr,
		User:        c.PostgresUser,
		Password:    c.PostgresPassword,
		DBName:      c.PostgresDB,
		MaxOpen:     c.DBMaxOpenConns,
		MaxIdle:     c.DBMaxIdleConns,
		MaxLifetime: c.DBConnMaxLifetime,
	}
}

func (c *Config) ToCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxReq:    c.CBMaxRequests,
		MinReq:    c.CBMinRequests,
		FailRatio: c.CBFailureRatio,
		Interval:  c.CBInterval,
		Timeout:   c.CBTimeout,
	}
}

func (c *Config) ToKafkaTopicConfig() KafkaTopicConfig {
	return KafkaTopicConfig{
		Brokers:           []string{c.KafkaAddr},
		Topic:             c.KafkaTopicOrders,
		GroupID:           c.KafkaGroupID,
		Partitions:        c.KafkaNumPartitions,
		ReplicationFactor: c.KafkaReplicationFactor,
		Timeout:           c.KafkaTimeout,
	}
}

func (c *Config) ToProducerConfig() ProducerConfig {
	return ProducerConfig{
		BatchSize:    c.KafkaProducerBatchSize,
		BatchTimeout: c.KafkaProducerBatchTimeout,
		KafkaTimeout: c.KafkaTimeout,
		TopicConfig:  c.ToKafkaTopicConfig(),
	}
}

func (c *Config) ToConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		MinBytes:      c.KafkaConsumerMinBytes,
		MaxBytes:      c.KafkaConsumerMaxBytes,
		BackoffJitter: c.DBRetryBackoffJitter,
		KafkaTimeout:  c.KafkaTimeout,
		DBTimeout:     c.DBTimeout,
		CommitTimeout: c.KafkaCommitTimeout,
		BackoffBase:   c.DBRetryBackoffBase,
		TopicConfig:   c.ToKafkaTopicConfig(),
	}
}

func (c *Config) ToRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Capacity:      c.RateLimitCapacity,
		Rate:          c.RateLimitRate,
		BannedMaxSize: c.CacheBannedIPMaxSize,
		Timeout:       c.RateLimitTimeout,
		BannedTTL:     c.CacheBannedIPTTL,
		CBConfig:      c.ToCircuitBreakerConfig(),
	}
}

func (c *Config) ToRedisRepoConfig() RedisRepoConfig {
	return RedisRepoConfig{
		Addr:       c.RedisAddr,
		HistoryTTL: c.HistoryTTLSeconds,
		CBConfig:   c.ToCircuitBreakerConfig(),
	}
}

func (c *Config) ToTicketServiceConfig() TicketServiceConfig {
	return TicketServiceConfig{
		SoldOutMaxSize:  c.CacheSoldOutMaxSize,
		SoldOutTTL:      c.CacheSoldOutTTL,
		RollbackTimeout: c.RedisTimeout,
		RedisTimeout:    c.RedisTimeout,
	}
}
