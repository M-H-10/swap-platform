// Package configs reads the application settings from environment variables.
package configs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every setting the server needs to start.
type Config struct {
	AppEnv      string   // development or production
	Host        string   // address to listen on, e.g. 127.0.0.1
	Port        string   // port to listen on, e.g. 5002
	CORSOrigins []string // frontend addresses allowed to call the API
	DatabaseURL string   // PostgreSQL connection string
	RabbitMQURL string   // RabbitMQ connection string; empty = ML predictions off

	JWTSecret       string        // key that signs access tokens
	AccessTokenTTL  time.Duration // how long an access token lives
	RefreshTokenTTL time.Duration // how long a login session lives
	CookieSecure    bool          // send the refresh cookie over HTTPS only
}

// Load reads the .env file (if it exists) and then the environment variables.
func Load() (*Config, error) {
	// In development the values come from .env. In production there is no
	// .env file and the values come from the real environment, so a missing
	// file is not an error.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("reading .env: %w", err)
	}

	accessTTL, err := time.ParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("REFRESH_TOKEN_TTL: %w", err)
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Host:        getEnv("APP_HOST", "127.0.0.1"),
		Port:        getEnv("APP_PORT", "5002"),
		CORSOrigins: splitList(getEnv("CORS_ORIGINS", "http://localhost:5173")),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),

		JWTSecret:       getEnv("JWT_SECRET", ""),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		CookieSecure:    getEnv("COOKIE_SECURE", "false") == "true",
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Addr returns "host:port", the format Fiber's Listen expects.
func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
}

func (c *Config) validate() error {
	if c.AppEnv != "development" && c.AppEnv != "production" {
		return fmt.Errorf("APP_ENV must be development or production, got %q", c.AppEnv)
	}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("APP_PORT must be a number between 1 and 65535, got %q", c.Port)
	}
	if len(c.CORSOrigins) == 0 {
		return errors.New("CORS_ORIGINS must contain at least one address")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required (see .env.example)")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters (see .env.example)")
	}
	if c.AccessTokenTTL < time.Minute || c.AccessTokenTTL > time.Hour {
		return errors.New("ACCESS_TOKEN_TTL must be between 1m and 1h")
	}
	if c.RefreshTokenTTL < time.Hour {
		return errors.New("REFRESH_TOKEN_TTL must be at least 1h")
	}
	if c.AppEnv == "production" && !c.CookieSecure {
		return errors.New("COOKIE_SECURE must be true in production")
	}
	return nil
}

// getEnv returns the variable's value, or fallback when it is empty.
func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// splitList turns "a, b,c" into ["a", "b", "c"].
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
