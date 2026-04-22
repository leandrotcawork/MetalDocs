package domain

import (
	"errors"
	"testing"
)

func TestSoDSelfSign(t *testing.T) {
	err := CheckSoD("author-1", "author-1", nil)
	if !errors.Is(err, ErrAuthorCannotSign) {
		t.Errorf("want ErrAuthorCannotSign; got %v", err)
	}
}

func TestSoDReSignAcrossStages(t *testing.T) {
	prior := []Signoff{makeSignoff("actor-1", DecisionApprove)}
	err := CheckSoD("author-1", "actor-1", prior)
	if !errors.Is(err, ErrActorAlreadySigned) {
		t.Errorf("want ErrActorAlreadySigned; got %v", err)
	}
}

func TestSoDFreshActor(t *testing.T) {
	prior := []Signoff{makeSignoff("actor-1", DecisionApprove)}
	err := CheckSoD("author-1", "fresh-actor", prior)
	if err != nil {
		t.Errorf("fresh actor should pass SoD; got %v", err)
	}
}

func TestSoDEmptyPrior(t *testing.T) {
	err := CheckSoD("author-1", "actor-2", nil)
	if err != nil {
		t.Errorf("empty prior signoffs should pass SoD; got %v", err)
	}
}
