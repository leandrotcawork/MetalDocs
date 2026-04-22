package signature

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("signature: invalid credentials")
	ErrRateLimited        = errors.New("signature: too many failed attempts, try again later")
)

// IamUserReader abstracts password-hash lookup for testability.
type IamUserReader interface {
	GetPasswordHash(ctx context.Context, userID string) ([]byte, error)
}

// EventEmitterStub abstracts audit-event emission for Phase 3 (Phase 5 wires real one).
type EventEmitterStub interface {
	EmitAuthFailed(ctx context.Context, actorUserID, reason string)
}

const (
	maxFailures  = 5
	windowDur    = 60 * time.Second
	entryTTL     = windowDur * 2
	janitorPeriod = 30 * time.Second
)

type failEntry struct {
	count   int
	oldest  time.Time // oldest failure in window
	lastAt  time.Time
}

// PasswordReauthProvider implements Provider using bcrypt against iam_users.password_hash.
type PasswordReauthProvider struct {
	reader   IamUserReader
	emitter  EventEmitterStub

	mu      sync.Mutex
	entries map[string]*failEntry
	cancel  context.CancelFunc
}

// NewPasswordReauthProvider creates the provider and starts a background janitor.
func NewPasswordReauthProvider(ctx context.Context, reader IamUserReader, emitter EventEmitterStub) *PasswordReauthProvider {
	ctx2, cancel := context.WithCancel(ctx)
	p := &PasswordReauthProvider{
		reader:  reader,
		emitter: emitter,
		entries: make(map[string]*failEntry),
		cancel:  cancel,
	}
	go p.janitor(ctx2)
	return p
}

func (p *PasswordReauthProvider) Method() string { return "password_reauth" }

func (p *PasswordReauthProvider) Sign(ctx context.Context, req SignRequest) (SignatureResult, error) {
	password, ok := req.Credentials["password"]
	if !ok || password == "" {
		return SignatureResult{}, ErrInvalidCredentials
	}

	// Rate limit check.
	if err := p.checkRateLimit(req.ActorUserID); err != nil {
		return SignatureResult{}, err
	}

	hash, err := p.reader.GetPasswordHash(ctx, req.ActorUserID)
	if err != nil {
		// User missing → same error as wrong password (disclosure-safe).
		p.recordFailure(req.ActorUserID)
		if p.emitter != nil {
			p.emitter.EmitAuthFailed(ctx, req.ActorUserID, "user_not_found")
		}
		return SignatureResult{}, ErrInvalidCredentials
	}

	cost, _ := bcrypt.Cost(hash)
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		p.recordFailure(req.ActorUserID)
		if p.emitter != nil {
			p.emitter.EmitAuthFailed(ctx, req.ActorUserID, "wrong_password")
		}
		return SignatureResult{}, ErrInvalidCredentials
	}

	// Success — clear failure state.
	p.clearFailure(req.ActorUserID)

	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"method":       "password_reauth",
		"bcrypt_cost":  cost,
		"verified_at":  now.Format(time.RFC3339),
	})
	return SignatureResult{Method: "password_reauth", Payload: payload, SignedAt: now}, nil
}

func (p *PasswordReauthProvider) checkRateLimit(actorID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.entries[actorID]
	if !ok {
		return nil
	}

	// Evict if window expired.
	if time.Since(e.oldest) >= windowDur {
		delete(p.entries, actorID)
		return nil
	}

	if e.count >= maxFailures {
		return ErrRateLimited
	}
	return nil
}

func (p *PasswordReauthProvider) recordFailure(actorID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	e, ok := p.entries[actorID]
	if !ok {
		p.entries[actorID] = &failEntry{count: 1, oldest: now, lastAt: now}
		return
	}
	// Reset window if oldest is expired.
	if time.Since(e.oldest) >= windowDur {
		e.count = 1
		e.oldest = now
	} else {
		e.count++
	}
	e.lastAt = now
}

func (p *PasswordReauthProvider) clearFailure(actorID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.entries, actorID)
}

// janitor sweeps expired entries every 30s.
func (p *PasswordReauthProvider) janitor(ctx context.Context) {
	ticker := time.NewTicker(janitorPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.sweepExpired()
		}
	}
}

func (p *PasswordReauthProvider) sweepExpired() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, e := range p.entries {
		if time.Since(e.lastAt) >= entryTTL {
			delete(p.entries, id)
		}
	}
	_ = fmt.Sprintf // keep fmt imported for future debug
}
