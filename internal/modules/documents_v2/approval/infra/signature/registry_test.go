package signature

import (
	"context"
	"errors"
	"testing"
)

type mockProvider struct{ method string }
func (m *mockProvider) Method() string { return m.method }
func (m *mockProvider) Sign(_ context.Context, _ SignRequest) (SignatureResult, error) {
	return SignatureResult{Method: m.method}, nil
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{"password_reauth"})
	r.Register(&mockProvider{"icp_brasil"})

	p, err := r.Get("password_reauth")
	if err != nil || p.Method() != "password_reauth" {
		t.Errorf("Get(password_reauth): err=%v, method=%v", err, p)
	}

	p2, err := r.Get("icp_brasil")
	if err != nil || p2.Method() != "icp_brasil" {
		t.Errorf("Get(icp_brasil): err=%v, method=%v", err, p2)
	}
}

func TestRegistryMissingMethod(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("unknown")
	if !errors.Is(err, ErrUnknownSignatureMethod) {
		t.Errorf("want ErrUnknownSignatureMethod; got %v", err)
	}
}
