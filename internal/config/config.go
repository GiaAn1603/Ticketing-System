package config

import (
	"Ticketing-System/internal/infrastructure"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort       string
	RedisAddr        string
	KafkaAddr        string
	PostgresAddr     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	ServerStartupTimeout  time.Duration
	ServerReadTimeout     time.Duration
	ServerWriteTimeout    time.Duration
	ServerIdleTimeout     time.Duration
	ServerShutdownTimeout time.Duration

	HistoryTTLSeconds int
	RedisTimeout      time.Duration

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBTimeout         time.Duration

	KafkaTopicOrders          string
	KafkaGroupID              string
	KafkaNumPartitions        int
	KafkaReplicationFactor    int
	KafkaProducerBatchSize    int
	KafkaProducerBatchTimeout time.Duration
	KafkaConsumerMinBytes     int
	KafkaConsumerMaxBytes     int
	KafkaTimeout              time.Duration
	KafkaCommitTimeout        time.Duration

	RateLimitCapacity int
	RateLimitRate     int
	RateLimitTimeout  time.Duration
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func LoadConfig() *Config {
	logger := infrastructure.GetLogger("CONFIG")

	if err := godotenv.Load(); err != nil {
		logger.Warn(
			"Env file load failed",
			infrastructure.KeyAction, "load_env",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			infrastructure.KeyError, err.Error(),
		)
	} else {
		logger.Info(
			"Env file loaded successfully",
			"path", ".env",
		)
	}

	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", ":8080"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaAddr:        getEnv("KAFKA_ADDR", "localhost:9092"),
		PostgresAddr:     getEnv("POSTGRES_ADDR", "localhost:5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "admin"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "SecretPassword123!"),
		PostgresDB:       getEnv("POSTGRES_DB", "ticket_db"),

		ServerStartupTimeout:  ServerStartupTimeout,
		ServerReadTimeout:     ServerReadTimeout,
		ServerWriteTimeout:    ServerWriteTimeout,
		ServerIdleTimeout:     ServerIdleTimeout,
		ServerShutdownTimeout: ServerShutdownTimeout,

		HistoryTTLSeconds: HistoryTTLSeconds,
		RedisTimeout:      RedisTimeout,

		DBMaxOpenConns:    DBMaxOpenConns,
		DBMaxIdleConns:    DBMaxIdleConns,
		DBConnMaxLifetime: DBConnMaxLifetime,
		DBTimeout:         DBTimeout,

		KafkaTopicOrders:          KafkaTopicOrders,
		KafkaGroupID:              KafkaGroupID,
		KafkaNumPartitions:        KafkaNumPartitions,
		KafkaReplicationFactor:    KafkaReplicationFactor,
		KafkaProducerBatchSize:    KafkaProducerBatchSize,
		KafkaProducerBatchTimeout: KafkaProducerBatchTimeout,
		KafkaConsumerMinBytes:     KafkaConsumerMinBytes,
		KafkaConsumerMaxBytes:     KafkaConsumerMaxBytes,
		KafkaTimeout:              KafkaTimeout,
		KafkaCommitTimeout:        KafkaCommitTimeout,

		RateLimitCapacity: RateLimitCapacity,
		RateLimitRate:     RateLimitRate,
		RateLimitTimeout:  RateLimitTimeout,
	}

	logger.Info(
		"Config loaded",
		"server_port", cfg.ServerPort,
		"redis_addr", cfg.RedisAddr,
		"kafka_addr", cfg.KafkaAddr,
		"pg_addr", cfg.PostgresAddr,
	)

	return cfg
}
