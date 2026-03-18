package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type WorkerConfig struct {
	PollIntervalSeconds int
	BatchSize           int
	ReviewReminderDays  int
	RunOnce             bool
	MaxAttempts         int
	RetryBaseSeconds    int
	RetryMaxSeconds     int
}

func LoadWorkerConfig() (WorkerConfig, error) {
	cfg := WorkerConfig{
		PollIntervalSeconds: 10,
		BatchSize:           25,
		ReviewReminderDays:  14,
		MaxAttempts:         5,
		RetryBaseSeconds:    10,
		RetryMaxSeconds:     300,
	}

	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_POLL_INTERVAL_SECONDS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return WorkerConfig{}, fmt.Errorf("invalid METALDOCS_WORKER_POLL_INTERVAL_SECONDS")
		}
		cfg.PollIntervalSeconds = value
	}
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_BATCH_SIZE")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return WorkerConfig{}, fmt.Errorf("invalid METALDOCS_WORKER_BATCH_SIZE")
		}
		cfg.BatchSize = value
	}
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_REVIEW_REMINDER_DAYS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return WorkerConfig{}, fmt.Errorf("invalid METALDOCS_WORKER_REVIEW_REMINDER_DAYS")
		}
		cfg.ReviewReminderDays = value
	}
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_RUN_ONCE")); raw != "" {
		cfg.RunOnce = strings.EqualFold(raw, "true") || raw == "1"
	}
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_MAX_ATTEMPTS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return WorkerConfig{}, fmt.Errorf("invalid METALDOCS_WORKER_MAX_ATTEMPTS")
		}
		cfg.MaxAttempts = value
	}
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_RETRY_BASE_SECONDS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return WorkerConfig{}, fmt.Errorf("invalid METALDOCS_WORKER_RETRY_BASE_SECONDS")
		}
		cfg.RetryBaseSeconds = value
	}
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_WORKER_RETRY_MAX_SECONDS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < cfg.RetryBaseSeconds {
			return WorkerConfig{}, fmt.Errorf("invalid METALDOCS_WORKER_RETRY_MAX_SECONDS")
		}
		cfg.RetryMaxSeconds = value
	}

	return cfg, nil
}
