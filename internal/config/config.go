package config

import (
	"log"
	"os"

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
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("[CONFIG][WARN] Load env file failed | action=use_defaults | err=%v", err)
	} else {
		log.Println("[CONFIG][INFO] Load env file success | source=.env")
	}

	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", ":8080"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaAddr:        getEnv("KAFKA_ADDR", "localhost:9092"),
		PostgresAddr:     getEnv("POSTGRES_ADDR", "localhost:5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "admin"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "SecretPassword123!"),
		PostgresDB:       getEnv("POSTGRES_DB", "ticket_db"),
	}

	log.Printf("[CONFIG][INFO] Config loaded | port=%s | redis_addr=%s | kafka_addr=%s | pg_addr=%s", cfg.ServerPort, cfg.RedisAddr, cfg.KafkaAddr, cfg.PostgresAddr)

	return cfg
}
