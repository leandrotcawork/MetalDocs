package objectstore_test

import (
	"testing"
	"time"

	"metaldocs/internal/platform/objectstore"
)

func TestPresignContext_Caps(t *testing.T) {
	ctx, err := objectstore.NewPresignContext(objectstore.Config{
		MaxSizeBytes: 10 * 1024 * 1024, TTL: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.TTL != 15*time.Minute {
		t.Fatalf("ttl")
	}
}
