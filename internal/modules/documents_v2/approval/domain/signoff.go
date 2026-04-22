package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

var hashRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Decision represents an approve or reject vote.
type Decision string

const (
	DecisionApprove Decision = "approve"
	DecisionReject  Decision = "reject"
)

// Signoff is an immutable value object. All fields are unexported.
type Signoff struct {
	id                 string
	approvalInstanceID string
	stageInstanceID    string
	actorUserID        string
	actorTenantID      string
	decision           Decision
	comment            string
	signedAt           time.Time
	signatureMethod    string
	signaturePayload   json.RawMessage
	contentHash        string // always lowercase hex sha-256
}

// Getters — no setters exist; immutable after construction.
func (s *Signoff) ID() string                         { return s.id }
func (s *Signoff) ApprovalInstanceID() string         { return s.approvalInstanceID }
func (s *Signoff) StageInstanceID() string            { return s.stageInstanceID }
func (s *Signoff) ActorUserID() string                { return s.actorUserID }
func (s *Signoff) ActorTenantID() string              { return s.actorTenantID }
func (s *Signoff) Decision() Decision                 { return s.decision }
func (s *Signoff) Comment() string                    { return s.comment }
func (s *Signoff) SignedAt() time.Time                { return s.signedAt }
func (s *Signoff) SignatureMethod() string            { return s.signatureMethod }
func (s *Signoff) SignaturePayload() json.RawMessage  { return s.signaturePayload }
func (s *Signoff) ContentHash() string               { return s.contentHash }

// SignoffParams holds constructor inputs.
type SignoffParams struct {
	ID                 string
	ApprovalInstanceID string
	StageInstanceID    string
	ActorUserID        string
	ActorTenantID      string
	Decision           Decision
	Comment            string
	SignedAt           time.Time
	SignatureMethod    string
	SignaturePayload    json.RawMessage
	ContentHash        string
}

// NewSignoff constructs an immutable Signoff value object.
// ContentHash is normalized to lowercase; uppercase hex is accepted.
func NewSignoff(p SignoffParams) (*Signoff, error) {
	if p.ID == "" {
		return nil, errors.New("signoff: ID is required")
	}
	if p.ApprovalInstanceID == "" {
		return nil, errors.New("signoff: ApprovalInstanceID is required")
	}
	if p.StageInstanceID == "" {
		return nil, errors.New("signoff: StageInstanceID is required")
	}
	if p.ActorUserID == "" {
		return nil, errors.New("signoff: ActorUserID is required")
	}
	if p.ActorTenantID == "" {
		return nil, errors.New("signoff: ActorTenantID is required")
	}
	if p.Decision != DecisionApprove && p.Decision != DecisionReject {
		return nil, errors.New("signoff: Decision must be 'approve' or 'reject'")
	}
	if p.SignedAt.IsZero() {
		return nil, errors.New("signoff: SignedAt must not be zero")
	}

	// Normalize hash to lowercase, then validate.
	hash := strings.ToLower(p.ContentHash)
	if !hashRegex.MatchString(hash) {
		return nil, errors.New("signoff: ContentHash must be 64 lowercase hex chars (sha-256)")
	}

	return &Signoff{
		id:                 p.ID,
		approvalInstanceID: p.ApprovalInstanceID,
		stageInstanceID:    p.StageInstanceID,
		actorUserID:        p.ActorUserID,
		actorTenantID:      p.ActorTenantID,
		decision:           p.Decision,
		comment:            p.Comment,
		signedAt:           p.SignedAt,
		signatureMethod:    p.SignatureMethod,
		signaturePayload:   p.SignaturePayload,
		contentHash:        hash,
	}, nil
}

// MarshalJSON exposes Signoff for API responses.
func (s *Signoff) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":                   s.id,
		"approval_instance_id": s.approvalInstanceID,
		"stage_instance_id":    s.stageInstanceID,
		"actor_user_id":        s.actorUserID,
		"actor_tenant_id":      s.actorTenantID,
		"decision":             s.decision,
		"comment":              s.comment,
		"signed_at":            s.signedAt,
		"signature_method":     s.signatureMethod,
		"content_hash":         s.contentHash,
	})
}
