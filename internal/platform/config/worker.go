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
}

func LoadWorkerConfig() (WorkerConfig, error) {
	cfg := WorkerConfig{
		PollIntervalSeconds: 10,
		BatchSize:           25,
		ReviewReminderDays:  14,
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

	return cfg, nil
}
