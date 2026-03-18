package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	StorageProviderMemory = "memory"
	StorageProviderLocal  = "local"
	StorageProviderMinIO  = "minio"
)

type AttachmentsConfig struct {
	Provider              string
	AppEnv                string
	RootPath              string
	DownloadSecret        string
	DownloadTTLSeconds    int
	MinIOEndpoint         string
	MinIOAccessKey        string
	MinIOSecretKey        string
	MinIOBucket           string
	MinIOUseSSL           bool
	MinIOAutoCreateBucket bool
}

func LoadAttachmentsConfig() (AttachmentsConfig, error) {
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "local"
	}

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("METALDOCS_STORAGE_PROVIDER")))
	if provider == "" {
		provider = StorageProviderLocal
	}
	switch provider {
	case StorageProviderMemory, StorageProviderLocal, StorageProviderMinIO:
	default:
		return AttachmentsConfig{}, fmt.Errorf("invalid METALDOCS_STORAGE_PROVIDER: %s", provider)
	}

	root := strings.TrimSpace(os.Getenv("METALDOCS_ATTACHMENTS_ROOT"))
	if root == "" {
		root = "non_git/attachments"
	}

	secret := strings.TrimSpace(os.Getenv("METALDOCS_ATTACHMENTS_SIGNING_SECRET"))
	if secret == "" {
		return AttachmentsConfig{}, fmt.Errorf("METALDOCS_ATTACHMENTS_SIGNING_SECRET is required for provider %s", provider)
	}

	ttlSeconds := 300
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_ATTACHMENTS_DOWNLOAD_TTL_SECONDS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 30 {
			return AttachmentsConfig{}, fmt.Errorf("invalid METALDOCS_ATTACHMENTS_DOWNLOAD_TTL_SECONDS")
		}
		ttlSeconds = parsed
	}

	cfg := AttachmentsConfig{
		Provider:           provider,
		AppEnv:             appEnv,
		RootPath:           root,
		DownloadSecret:     secret,
		DownloadTTLSeconds: ttlSeconds,
	}

	if provider == StorageProviderMinIO {
		cfg.MinIOEndpoint = strings.TrimSpace(os.Getenv("METALDOCS_MINIO_ENDPOINT"))
		cfg.MinIOAccessKey = strings.TrimSpace(os.Getenv("METALDOCS_MINIO_ACCESS_KEY"))
		cfg.MinIOSecretKey = os.Getenv("METALDOCS_MINIO_SECRET_KEY")
		cfg.MinIOBucket = strings.TrimSpace(os.Getenv("METALDOCS_MINIO_BUCKET"))
		cfg.MinIOUseSSL = parseBoolEnv("METALDOCS_MINIO_USE_SSL", false)
		cfg.MinIOAutoCreateBucket = parseBoolEnv("METALDOCS_MINIO_AUTO_CREATE_BUCKET", false)

		if cfg.MinIOEndpoint == "" || cfg.MinIOAccessKey == "" || cfg.MinIOSecretKey == "" || cfg.MinIOBucket == "" {
			return AttachmentsConfig{}, fmt.Errorf("minio config missing: set METALDOCS_MINIO_ENDPOINT/METALDOCS_MINIO_ACCESS_KEY/METALDOCS_MINIO_SECRET_KEY/METALDOCS_MINIO_BUCKET")
		}
	}

	return cfg, nil
}

func parseBoolEnv(name string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}
	return strings.EqualFold(raw, "true") || raw == "1"
}
