package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/utils/logger"
)

type Config struct {
	Log       LogConfig
	DB        DBConfig
	Desco     DescoConfig
	Nesco     NescoConfig
	Telemetry TelemetryConfig
	Telegram  TelegramConfig
}

type TelegramConfig struct {
	Token     string
	RateLimit float64
}

type TelemetryConfig struct {
	Enabled      bool
	OTLPEndpoint string
	ServiceName  string
	Environment  string
}

type LogConfig struct {
	Level  slog.Level
	Format logger.Format
}

type DBConfig struct {
	DSN string
}

type DescoConfig struct {
	BasePath   string
	Timeout    time.Duration
	Retry      int
	RetryDelay time.Duration
	RateLimit  float64
}

type NescoConfig struct {
	BasePath   string
	Timeout    time.Duration
	Retry      int
	RetryDelay time.Duration
	RateLimit  float64
}

var instance *Config

func Load() *Config {
	instance = &Config{
		Log: LogConfig{
			Level:  parseLogLevel(getEnv("MA_LOG_LEVEL", "info")),
			Format: logger.Format(getEnv("MA_LOG_FORMAT", "json")),
		},
		DB: DBConfig{
			DSN: getEnv("MA_DATABASE_URL", "postgres://myuser:mysecretpassword@localhost:5433/meterbot?sslmode=disable"),
		},
		Desco: DescoConfig{
			BasePath:   getEnv("MA_DESCO_BASE_PATH", "https://prepaid.desco.org.bd"),
			Timeout:    parseDuration(getEnv("MA_DESCO_TIMEOUT", "10s")),
			Retry:      parseInt(getEnv("MA_DESCO_RETRY", "3")),
			RetryDelay: parseDuration(getEnv("MA_DESCO_RETRY_DELAY", "1s")),
			RateLimit:  parseFloat(getEnv("MA_DESCO_RATE_LIMIT", "5")),
		},
		Nesco: NescoConfig{
			BasePath:   getEnv("MA_NESCO_BASE_PATH", "https://customer.nesco.gov.bd"),
			Timeout:    parseDuration(getEnv("MA_NESCO_TIMEOUT", "10s")),
			Retry:      parseInt(getEnv("MA_NESCO_RETRY", "3")),
			RetryDelay: parseDuration(getEnv("MA_NESCO_RETRY_DELAY", "1s")),
			RateLimit:  parseFloat(getEnv("MA_NESCO_RATE_LIMIT", "2")),
		},
		Telemetry: TelemetryConfig{
			Enabled:      parseBool(getEnv("MA_OTEL_ENABLED", "false")),
			OTLPEndpoint: getEnv("MA_OTLP_ENDPOINT", "localhost:4317"),
			ServiceName:  getEnv("MA_SERVICE_NAME", "meterbot"),
			Environment:  getEnv("MA_ENVIRONMENT", "development"),
		},
		Telegram: TelegramConfig{
			Token:     getEnv("MA_TELEGRAM_TOKEN", ""),
			RateLimit: parseFloat(getEnv("MA_TELEGRAM_RATE_LIMIT", "30")),
		},
	}
	return instance
}

// Get returns the loaded config, calling Load if it hasn't been called yet.
func Get() *Config {
	if instance == nil {
		return Load()
	}
	return instance
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return l
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
