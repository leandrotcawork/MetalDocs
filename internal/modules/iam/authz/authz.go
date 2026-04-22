package authz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
)

type ErrCapabilityDenied struct {
	Capability string
	AreaCode   string
	ActorID    string
}

func (e ErrCapabilityDenied) Error() string {
	return fmt.Sprintf("authz: capability %q denied for actor %q in area %q", e.Capability, e.ActorID, e.AreaCode)
}

type capCacheKey struct{}

type capCache struct {
	mu      sync.Mutex
	granted map[string]bool
}

func WithCapCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, capCacheKey{}, &capCache{
		granted: make(map[string]bool),
	})
}

func Require(ctx context.Context, tx *sql.Tx, capability, areaCode string) error {
	if cacheGranted(ctx, capability, areaCode) {
		return appendAssertedCap(ctx, tx, capability, areaCode)
	}

	var granted bool
	err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM metaldocs.role_capabilities rc
  JOIN metaldocs.user_process_areas upa
    ON upa.role = rc.role
   AND upa.tenant_id = current_setting('metaldocs.tenant_id', false)::uuid
   AND upa.user_id   = current_setting('metaldocs.actor_id', false)
   AND upa.effective_to IS NULL
  WHERE rc.capability = $1
    AND ($2 = 'tenant' OR upa.area_code = $2)
)`,
		capability, areaCode,
	).Scan(&granted)
	if err != nil {
		return err
	}

	if !granted {
		actorID, err := actorIDFromTx(ctx, tx)
		if err != nil {
			return err
		}
		return ErrCapabilityDenied{
			Capability: capability,
			AreaCode:   areaCode,
			ActorID:    actorID,
		}
	}

	storeGranted(ctx, capability, areaCode)
	return appendAssertedCap(ctx, tx, capability, areaCode)
}

func BypassSystem(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "SELECT set_config('metaldocs.bypass_authz', 'scheduler', true)")
	return err
}

func cacheGranted(ctx context.Context, capability, areaCode string) bool {
	cache, ok := ctx.Value(capCacheKey{}).(*capCache)
	if !ok || cache == nil {
		return false
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.granted[cacheKey(capability, areaCode)]
}

func storeGranted(ctx context.Context, capability, areaCode string) {
	cache, ok := ctx.Value(capCacheKey{}).(*capCache)
	if !ok || cache == nil {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.granted[cacheKey(capability, areaCode)] = true
}

func cacheKey(capability, areaCode string) string {
	return capability + "\x00" + areaCode
}

func actorIDFromTx(ctx context.Context, tx *sql.Tx) (string, error) {
	var actorID string
	err := tx.QueryRowContext(ctx, "SELECT current_setting('metaldocs.actor_id', false)").Scan(&actorID)
	if err != nil {
		return "", err
	}
	return actorID, nil
}

func appendAssertedCap(ctx context.Context, tx *sql.Tx, capability, areaCode string) error {
	var raw sql.NullString
	if err := tx.QueryRowContext(ctx, "SELECT current_setting('metaldocs.asserted_caps', true)").Scan(&raw); err != nil {
		return err
	}

	var asserted []map[string]string
	if raw.Valid && raw.String != "" {
		if err := json.Unmarshal([]byte(raw.String), &asserted); err != nil {
			return err
		}
	}

	asserted = append(asserted, map[string]string{
		"cap":  capability,
		"area": areaCode,
	})

	encoded, err := json.Marshal(asserted)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "SELECT set_config('metaldocs.asserted_caps', $1, true)", string(encoded))
	return err
}
