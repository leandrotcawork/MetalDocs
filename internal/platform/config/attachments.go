package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AttachmentsConfig struct {
	RootPath           string
	DownloadSecret     string
	DownloadTTLSeconds int
}

func LoadAttachmentsConfig() (AttachmentsConfig, error) {
	root := strings.TrimSpace(os.Getenv("METALDOCS_ATTACHMENTS_ROOT"))
	if root == "" {
		root = "non_git/attachments"
	}

	secret := strings.TrimSpace(os.Getenv("METALDOCS_ATTACHMENTS_SIGNING_SECRET"))
	if secret == "" {
		secret = "metaldocs-local-dev-secret"
	}

	ttlSeconds := 300
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_ATTACHMENTS_DOWNLOAD_TTL_SECONDS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 30 {
			return AttachmentsConfig{}, fmt.Errorf("invalid METALDOCS_ATTACHMENTS_DOWNLOAD_TTL_SECONDS")
		}
		ttlSeconds = parsed
	}

	return AttachmentsConfig{
		RootPath:           root,
		DownloadSecret:     secret,
		DownloadTTLSeconds: ttlSeconds,
	}, nil
}
