package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	AppEnv string

	HTTPPort string

	// Built from POSTGRES_USER/PASSWORD/HOST/PORT/DB components
	DatabaseURL    string
	MigrationsPath string

	RedisAddr     string
	RedisPassword string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	UploadsDir     string
	UploadsBaseURL string
}

func Load() (*Config, error) {
	dbHost := getEnv("POSTGRES_HOST", "postgres")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "richtalk")
	dbPass := getEnv("POSTGRES_PASSWORD", "")
	dbName := getEnv("POSTGRES_DB", "richtalk")

	cfg := &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		HTTPPort: getEnv("API_PORT", "8080"),

		DatabaseURL:    fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),

		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		JWTSecret:      getEnv("JWT_SECRET", ""),
		UploadsDir:     getEnv("UPLOADS_DIR", "/uploads"),
		UploadsBaseURL: getEnv("UPLOADS_BASE_URL", "/uploads"),
	}

	accessTTL, err := time.ParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid ACCESS_TOKEN_TTL: %w", err)
	}
	cfg.AccessTokenTTL = accessTTL

	refreshTTL, err := time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid REFRESH_TOKEN_TTL: %w", err)
	}
	cfg.RefreshTokenTTL = refreshTTL

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters long")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
