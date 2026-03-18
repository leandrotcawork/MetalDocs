package unit

import (
	"os"
	"testing"

	"metaldocs/internal/platform/config"
)

func TestLoadAttachmentsConfigRequiresSecretForMinIO(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("METALDOCS_STORAGE_PROVIDER", "minio")
	t.Setenv("METALDOCS_MINIO_ENDPOINT", "minio:9000")
	t.Setenv("METALDOCS_MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("METALDOCS_MINIO_SECRET_KEY", "secret")
	t.Setenv("METALDOCS_MINIO_BUCKET", "metaldocs-attachments")
	_ = os.Unsetenv("METALDOCS_ATTACHMENTS_SIGNING_SECRET")

	_, err := config.LoadAttachmentsConfig()
	if err == nil {
		t.Fatalf("expected error when minio storage has no signing secret")
	}
}

func TestLoadAttachmentsConfigRequiresSecretForLocalProvider(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("METALDOCS_STORAGE_PROVIDER", "local")
	_ = os.Unsetenv("METALDOCS_ATTACHMENTS_SIGNING_SECRET")

	_, err := config.LoadAttachmentsConfig()
	if err == nil {
		t.Fatalf("expected error when local storage has no signing secret")
	}
}
