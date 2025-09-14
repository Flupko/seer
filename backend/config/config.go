package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type DBConfig struct {
	DSN             string
	MaxConns        int
	MinConns        int
	MaxConnIdleTime time.Duration
}

type Config struct {
	Port int
	DB   DBConfig
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	portStr := os.Getenv("PORT")
	if portStr == "" {
		return nil, fmt.Errorf("PORT is required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}
	cfg.Port = port

	mocStr := os.Getenv("POSTGRES_MAX_CONNS")
	if mocStr == "" {
		cfg.DB.MaxConns = 5
	} else if cfg.DB.MaxConns, err = strconv.Atoi(mocStr); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_MAX_CONNS: %w", err)
	}

	micStr := os.Getenv("POSTGRES_MIN_CONNS")
	if micStr == "" {
		cfg.DB.MinConns = 5
	} else if cfg.DB.MinConns, err = strconv.Atoi(micStr); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_MIN_CONNS: %w", err)
	}

	mitStr := os.Getenv("POSTGRES_MAX_IDLE_TIME")
	if mitStr == "" {
		cfg.DB.MaxConnIdleTime = 15 * time.Minute
	} else if cfg.DB.MaxConnIdleTime, err = time.ParseDuration(mitStr); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_MAX_IDLE_TIME: %w", err)
	}

	cfg.DB.DSN = os.Getenv("PG_DSN")
	if cfg.DB.DSN == "" {
		return nil, errors.New("PG_DSN is required")
	}

	return cfg, nil

}
