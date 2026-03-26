package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv         string
	HTTPAddr       string
	CORSOrigin     string
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	SessionTTL     time.Duration
	RoomTTL        time.Duration
	RoomCodeLength int
}

func Load() Config {
	cfg := Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		CORSOrigin:     getEnv("CORS_ORIGIN", "http://localhost"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/huzhoumahjong?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getEnvInt("REDIS_DB", 0),
		SessionTTL:     getEnvDuration("SESSION_TTL", 24*time.Hour),
		RoomTTL:        getEnvDuration("ROOM_TTL", 24*time.Hour),
		RoomCodeLength: getEnvInt("ROOM_CODE_LENGTH", 6),
	}

	log.Printf("config loaded env=%s http=%s redis=%s", cfg.AppEnv, cfg.HTTPAddr, cfg.RedisAddr)
	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
