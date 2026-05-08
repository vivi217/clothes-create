package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv       string
	Port         string
	PostgresDSN  string
	RedisAddr    string
	MinioEndpoint string
	MinioPublicEndpoint string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket  string
	MilvusAddress string
	EngineBaseURL string
}

func Load() Config {
	return Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "8080"),
		PostgresDSN:    getEnv("POSTGRES_DSN", "postgres://garment:garment123@localhost:5432/garment_ai?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin123"),
		MinioBucket:    getEnv("MINIO_BUCKET", "garment-3d-models"),
		MilvusAddress:  getEnv("MILVUS_ADDRESS", "localhost:19530"),
		EngineBaseURL:  getEnv("ENGINE_BASE_URL", "http://localhost:8010"),
	}
}

func (c Config) Address() string {
	return fmt.Sprintf(":%s", c.Port)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}