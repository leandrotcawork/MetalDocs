package signature

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrUnknownSignatureMethod returned when registry has no Provider for the given method.
var ErrUnknownSignatureMethod = errors.New("signature: unknown method")

// SignRequest carries per-sign inputs.
type SignRequest struct {
	ActorUserID  string
	ActorTenantID string
	ContentHash  string
	Credentials  map[string]string // method-specific; "password" key for password_reauth
}

// SignatureResult is the opaque attestation returned on successful signing.
type SignatureResult struct {
	Method    string
	Payload   json.RawMessage // opaque bag — no secrets
	SignedAt  time.Time
}

// Provider is the signature method seam. Adding a new method (e.g. ICP-Brasil)
// means implementing Provider and registering it — zero service-code change.
type Provider interface {
	Method() string
	Sign(ctx context.Context, req SignRequest) (SignatureResult, error)
}

// Registry dispatches Sign calls by method name.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider. Overwrites if method already registered.
func (r *Registry) Register(p Provider) {
	r.providers[p.Method()] = p
}

// Get returns the provider for the given method name.
func (r *Registry) Get(method string) (Provider, error) {
	p, ok := r.providers[method]
	if !ok {
		return nil, ErrUnknownSignatureMethod
	}
	return p, nil
}
