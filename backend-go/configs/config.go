// Package configs reads the application settings from environment variables.
package configs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every setting the server needs to start.
type Config struct {
	AppEnv      string   // development or production
	Host        string   // address to listen on, e.g. 127.0.0.1
	Port        string   // port to listen on, e.g. 5002
	CORSOrigins []string // frontend addresses allowed to call the API
}

// Load reads the .env file (if it exists) and then the environment variables.
func Load() (*Config, error) {
	// In development the values come from .env. In production there is no
	// .env file and the values come from the real environment, so a missing
	// file is not an error.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("reading .env: %w", err)
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Host:        getEnv("APP_HOST", "127.0.0.1"),
		Port:        getEnv("APP_PORT", "5002"),
		CORSOrigins: splitList(getEnv("CORS_ORIGINS", "http://localhost:5173")),
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
