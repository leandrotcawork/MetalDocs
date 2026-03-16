package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	RepositoryMemory   = "memory"
	RepositoryPostgres = "postgres"
)

func RepositoryMode() (string, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("METALDOCS_REPOSITORY")))
	if mode == "" {
		mode = RepositoryMemory
	}
	switch mode {
	case RepositoryMemory, RepositoryPostgres:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid METALDOCS_REPOSITORY: %s", mode)
	}
}
