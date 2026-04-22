package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var validHash = "a" + strings.Repeat("b", 63) // 64 lowercase hex chars

func validSignoffParams() SignoffParams {
	return SignoffParams{
		ID:                 "sig-1",
		ApprovalInstanceID: "inst-1",
		StageInstanceID:    "stage-1",
		ActorUserID:        "user-1",
		ActorTenantID:      "tenant-1",
		Decision:           DecisionApprove,
		SignedAt:           time.Now(),
		SignatureMethod:    "password",
		SignaturePayload:   json.RawMessage(`{}`),
		ContentHash:        validHash,
	}
}

func TestNewSignoffHappy(t *testing.T) {
	s, err := NewSignoff(validSignoffParams())
	if err != nil {
		t.Fatalf("NewSignoff: %v", err)
	}
	if s.ActorUserID() != "user-1" {
		t.Error("ActorUserID mismatch")
	}
	if s.ContentHash() != validHash {
		t.Error("ContentHash mismatch")
	}
}

func TestNewSignoffEmptyID(t *testing.T) {
	p := validSignoffParams()
	p.ID = ""
	if _, err := NewSignoff(p); err == nil {
		t.Error("empty ID should fail")
	}
}

func TestNewSignoffEmptyInstanceID(t *testing.T) {
	p := validSignoffParams()
	p.ApprovalInstanceID = ""
	if _, err := NewSignoff(p); err == nil {
		t.Error("empty ApprovalInstanceID should fail")
	}
}

func TestNewSignoffEmptyActorID(t *testing.T) {
	p := validSignoffParams()
	p.ActorUserID = ""
	if _, err := NewSignoff(p); err == nil {
		t.Error("empty ActorUserID should fail")
	}
}

func TestNewSignoffBadHash63Chars(t *testing.T) {
	p := validSignoffParams()
	p.ContentHash = strings.Repeat("a", 63)
	if _, err := NewSignoff(p); err == nil {
		t.Error("63-char hash should fail")
	}
}

func TestNewSignoffBadHashNonHex(t *testing.T) {
	p := validSignoffParams()
	p.ContentHash = strings.Repeat("g", 64) // 'g' not hex
	if _, err := NewSignoff(p); err == nil {
		t.Error("non-hex hash should fail")
	}
}

func TestNewSignoffZeroTime(t *testing.T) {
	p := validSignoffParams()
	p.SignedAt = time.Time{}
	if _, err := NewSignoff(p); err == nil {
		t.Error("zero SignedAt should fail")
	}
}

func TestNewSignoffUnknownDecision(t *testing.T) {
	p := validSignoffParams()
	p.Decision = "abstain"
	if _, err := NewSignoff(p); err == nil {
		t.Error("unknown decision should fail")
	}
}

// Hash canonicalization: uppercase input → lowercase stored.
func TestNewSignoffHashCanonicalizedUppercase(t *testing.T) {
	p := validSignoffParams()
	upper := strings.ToUpper(validHash)
	p.ContentHash = upper

	s, err := NewSignoff(p)
	if err != nil {
		t.Fatalf("uppercase hash should be accepted: %v", err)
	}
	if s.ContentHash() != validHash {
		t.Errorf("ContentHash() = %q; want lowercase %q", s.ContentHash(), validHash)
	}
}

func TestSignoffGetters(t *testing.T) {
	p := validSignoffParams()
	s, err := NewSignoff(p)
	if err != nil {
		t.Fatalf("NewSignoff: %v", err)
	}
	if s.ID() != "sig-1" { t.Error("ID") }
	if s.ApprovalInstanceID() != "inst-1" { t.Error("ApprovalInstanceID") }
	if s.StageInstanceID() != "stage-1" { t.Error("StageInstanceID") }
	if s.ActorTenantID() != "tenant-1" { t.Error("ActorTenantID") }
	if s.Decision() != DecisionApprove { t.Error("Decision") }
	if s.SignatureMethod() != "password" { t.Error("SignatureMethod") }
	if s.SignaturePayload() == nil { t.Error("SignaturePayload") }
	if s.Comment() != "" { t.Error("Comment should be empty") }
	if s.SignedAt().IsZero() { t.Error("SignedAt") }
}

func TestSignoffMarshalJSON(t *testing.T) {
	s, _ := NewSignoff(validSignoffParams())
	b, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["actor_user_id"] != "user-1" {
		t.Errorf("actor_user_id = %v", m["actor_user_id"])
	}
}
