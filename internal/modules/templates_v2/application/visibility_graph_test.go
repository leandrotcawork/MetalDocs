package application_test

import (
	"errors"
	"testing"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func TestVisibilityGraph_SimpleCycle(t *testing.T) {
	err := application.DetectVisibilityCycle([]domain.Placeholder{
		{ID: "p1", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p2"}},
		{ID: "p2", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p1"}},
	})
	if !errors.Is(err, domain.ErrPlaceholderCycle) {
		t.Fatalf("expected ErrPlaceholderCycle, got %v", err)
	}
}

func TestVisibilityGraph_LongCycle(t *testing.T) {
	err := application.DetectVisibilityCycle([]domain.Placeholder{
		{ID: "p1", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p2"}},
		{ID: "p2", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p3"}},
		{ID: "p3", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p1"}},
	})
	if !errors.Is(err, domain.ErrPlaceholderCycle) {
		t.Fatalf("expected ErrPlaceholderCycle, got %v", err)
	}
}

func TestVisibilityGraph_Acyclic_OK(t *testing.T) {
	err := application.DetectVisibilityCycle([]domain.Placeholder{
		{ID: "p1"},
		{ID: "p2", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p1"}},
		{ID: "p3", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "p2"}},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestVisibilityGraph_UnknownReference(t *testing.T) {
	err := application.DetectVisibilityCycle([]domain.Placeholder{
		{ID: "p1", VisibleIf: &domain.VisibilityCondition{PlaceholderID: "missing"}},
	})
	if !errors.Is(err, domain.ErrInvalidConstraint) {
		t.Fatalf("expected ErrInvalidConstraint, got %v", err)
	}
}
