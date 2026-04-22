package signature

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// fakeUserReader implements IamUserReader.
type fakeUserReader struct {
	users map[string][]byte // userID → bcrypt hash
}

func newFakeReader(users map[string]string) *fakeUserReader {
	hashes := make(map[string][]byte)
	for id, pw := range users {
		h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
		hashes[id] = h
	}
	return &fakeUserReader{users: hashes}
}

func (f *fakeUserReader) GetPasswordHash(_ context.Context, userID string) ([]byte, error) {
	h, ok := f.users[userID]
	if !ok {
		return nil, errors.New("not found")
	}
	return h, nil
}

// fakeEmitter records auth failures.
type fakeEmitter struct {
	failures []string
}
func (e *fakeEmitter) EmitAuthFailed(_ context.Context, actorID, _ string) {
	e.failures = append(e.failures, actorID)
}

func newProvider(users map[string]string) (*PasswordReauthProvider, *fakeEmitter) {
	em := &fakeEmitter{}
	p := NewPasswordReauthProvider(context.Background(), newFakeReader(users), em)
	return p, em
}

func TestPasswordReauthHappy(t *testing.T) {
	p, _ := newProvider(map[string]string{"u1": "secret123"})
	res, err := p.Sign(context.Background(), SignRequest{
		ActorUserID: "u1", ActorTenantID: "t1",
		ContentHash: "abc", Credentials: map[string]string{"password": "secret123"},
	})
	if err != nil {
		t.Fatalf("happy path: %v", err)
	}
	if res.Method != "password_reauth" {
		t.Error("method mismatch")
	}
}

func TestPasswordReauthWrongPassword(t *testing.T) {
	p, em := newProvider(map[string]string{"u1": "correct"})
	_, err := p.Sign(context.Background(), SignRequest{
		ActorUserID: "u1", Credentials: map[string]string{"password": "wrong"},
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials; got %v", err)
	}
	if len(em.failures) == 0 {
		t.Error("failure event should be emitted")
	}
}

func TestPasswordReauthMissingUser(t *testing.T) {
	p, _ := newProvider(map[string]string{})
	_, err := p.Sign(context.Background(), SignRequest{
		ActorUserID: "ghost", Credentials: map[string]string{"password": "pw"},
	})
	// Must return same error as wrong password — no disclosure.
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("missing user: want ErrInvalidCredentials; got %v", err)
	}
}

func TestPasswordReauthRateLimitTrip(t *testing.T) {
	p, _ := newProvider(map[string]string{"u1": "correct"})
	req := SignRequest{ActorUserID: "u1", Credentials: map[string]string{"password": "wrong"}}

	for i := 0; i < maxFailures; i++ {
		p.Sign(context.Background(), req) //nolint
	}
	// 6th attempt should be rate-limited.
	_, err := p.Sign(context.Background(), req)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("want ErrRateLimited after %d failures; got %v", maxFailures, err)
	}
}

func TestPasswordReauthRateLimitResetAfterWindow(t *testing.T) {
	p, _ := newProvider(map[string]string{"u1": "correct"})
	req := SignRequest{ActorUserID: "u1", Credentials: map[string]string{"password": "wrong"}}

	for i := 0; i < maxFailures; i++ {
		p.Sign(context.Background(), req) //nolint
	}

	// Fake expiry by directly manipulating entry.
	p.mu.Lock()
	p.entries["u1"].oldest = time.Now().Add(-windowDur - time.Second)
	p.mu.Unlock()

	// Window expired — rate limit should reset; still fails (wrong pw) but not rate-limited.
	_, err := p.Sign(context.Background(), req)
	if errors.Is(err, ErrRateLimited) {
		t.Error("rate limit should reset after window expires")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials after reset; got %v", err)
	}
}
