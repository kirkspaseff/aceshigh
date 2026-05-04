// Package config loads runtime configuration from environment variables.
// We follow the 12-factor convention: connection strings and credentials
// come from env, never from CLI flags or config files.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
}

// Load reads required env vars and returns a Config. Returns an error
// listing any missing vars (rather than failing on the first one) so
// users see all problems at once.
func Load() (*Config, error) {
	var missing []string

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}

	return &Config{DatabaseURL: dbURL}, nil
}
