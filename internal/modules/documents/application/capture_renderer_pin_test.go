package application

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

type fakePinRepo struct {
	capturedPin     *domain.RendererPin
	capturedDoc     string
	capturedVersion int
}

func (f *fakePinRepo) SetVersionRendererPin(ctx context.Context, documentID string, versionNumber int, pin *domain.RendererPin) error {
	f.capturedPin = pin
	f.capturedDoc = documentID
	f.capturedVersion = versionNumber
	return nil
}

func TestCaptureRendererPin_WritesExpectedFields(t *testing.T) {
	repo := &fakePinRepo{}
	clock := func() time.Time { return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC) }

	capture := NewRendererPinCapturer(RendererPinCapturerConfig{
		CurrentRendererVersion: "1.0.0",
		CurrentLayoutIRHash:    "hash-deadbeef",
		Repo:                   repo,
		Clock:                  clock,
	})

	version := domain.Version{
		DocumentID:      "doc-1",
		Number:          3,
		ContentSource:   domain.ContentSourceBrowserEditor,
		TemplateKey:     "po-mddm-canvas",
		TemplateVersion: 2,
	}

	if err := capture.OnRelease(context.Background(), version); err != nil {
		t.Fatalf("OnRelease: %v", err)
	}

	if repo.capturedDoc != "doc-1" || repo.capturedVersion != 3 {
		t.Fatalf("wrong target: doc=%q version=%d", repo.capturedDoc, repo.capturedVersion)
	}
	if repo.capturedPin == nil {
		t.Fatalf("expected pin to be written")
	}
	want := domain.RendererPin{
		RendererVersion: "1.0.0",
		LayoutIRHash:    "hash-deadbeef",
		TemplateKey:     "po-mddm-canvas",
		TemplateVersion: 2,
		PinnedAt:        clock(),
	}
	if *repo.capturedPin != want {
		t.Fatalf("pin mismatch:\n got %+v\nwant %+v", *repo.capturedPin, want)
	}
}

func TestCaptureRendererPin_SkipsNonBrowserEditorSources(t *testing.T) {
	repo := &fakePinRepo{}
	capture := NewRendererPinCapturer(RendererPinCapturerConfig{
		CurrentRendererVersion: "1.0.0",
		CurrentLayoutIRHash:    "h",
		Repo:                   repo,
		Clock:                  time.Now,
	})

	// Native content does not use the MDDM engine, so no pin is needed.
	version := domain.Version{
		DocumentID:    "doc-2",
		Number:        1,
		ContentSource: domain.ContentSourceNative,
	}

	if err := capture.OnRelease(context.Background(), version); err != nil {
		t.Fatalf("OnRelease: %v", err)
	}
	if repo.capturedPin != nil {
		t.Fatalf("expected no pin for native content source, got %+v", repo.capturedPin)
	}
}

func TestCaptureRendererPin_ErrorsWhenTemplateMissing(t *testing.T) {
	repo := &fakePinRepo{}
	capture := NewRendererPinCapturer(RendererPinCapturerConfig{
		CurrentRendererVersion: "1.0.0",
		CurrentLayoutIRHash:    "h",
		Repo:                   repo,
		Clock:                  time.Now,
	})

	version := domain.Version{
		DocumentID:    "doc-3",
		Number:        1,
		ContentSource: domain.ContentSourceBrowserEditor,
		// Missing TemplateKey / TemplateVersion
	}

	if err := capture.OnRelease(context.Background(), version); err == nil {
		t.Fatalf("expected error when browser editor version has no template")
	}
}
