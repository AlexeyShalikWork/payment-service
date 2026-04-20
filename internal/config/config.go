package config

import (
	"os"
	"strings"
)

type Config struct {
	DBUrl         string
	KafkaBrokers  []string
	HTTPPort      string
	LogFormat     string
	MigrationsDir string
	ElasticURL    string
}

func Load() *Config {
	return &Config{
		DBUrl:        getEnv("DB_URL", "postgres://localhost:5432/payments"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		HTTPPort:     getEnv("HTTP_PORT", "3005"),
		LogFormat:      getEnv("LOG_FORMAT", "text"),
		MigrationsDir:  getEnv("MIGRATIONS_DIR", "migrations"),
		ElasticURL:     getEnv("ELASTIC_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
