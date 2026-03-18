package observability

import (
	"context"
	"database/sql"
	"time"
)

type RuntimeStatusProvider interface {
	Live(ctx context.Context) (int, map[string]any)
	Ready(ctx context.Context) (int, map[string]any)
	RuntimeMetrics(ctx context.Context) map[string]any
}

type StaticRuntimeStatusProvider struct {
	repositoryMode  string
	storageProvider string
	authEnabled     bool
}

func NewStaticRuntimeStatusProvider(repositoryMode, storageProvider string, authEnabled bool) *StaticRuntimeStatusProvider {
	return &StaticRuntimeStatusProvider{
		repositoryMode:  repositoryMode,
		storageProvider: storageProvider,
		authEnabled:     authEnabled,
	}
}

func (p *StaticRuntimeStatusProvider) Live(_ context.Context) (int, map[string]any) {
	return 200, map[string]any{
		"status": "live",
		"checks": []map[string]any{
			{"name": "process", "status": "up"},
		},
	}
}

func (p *StaticRuntimeStatusProvider) Ready(_ context.Context) (int, map[string]any) {
	return 200, map[string]any{
		"status": "ready",
		"checks": []map[string]any{
			{"name": "repository", "status": "up", "mode": p.repositoryMode},
			{"name": "storage", "status": "up", "provider": p.storageProvider},
			{"name": "auth", "status": "up", "enabled": p.authEnabled},
		},
	}
}

func (p *StaticRuntimeStatusProvider) RuntimeMetrics(_ context.Context) map[string]any {
	return map[string]any{
		"repositoryMode":  p.repositoryMode,
		"storageProvider": p.storageProvider,
		"authEnabled":     p.authEnabled,
		"auth": map[string]any{
			"users": map[string]any{
				"active":             0,
				"inactive":           0,
				"mustChangePassword": 0,
				"locked":             0,
			},
			"sessions": map[string]any{
				"active":  0,
				"expired": 0,
				"revoked": 0,
			},
		},
		"worker": map[string]any{
			"outbox": map[string]any{
				"claimable":    0,
				"pending":      0,
				"deadLettered": 0,
			},
		},
	}
}

type PostgresRuntimeStatusProvider struct {
	*StaticRuntimeStatusProvider
	db *sql.DB
}

func NewPostgresRuntimeStatusProvider(db *sql.DB, repositoryMode, storageProvider string, authEnabled bool) *PostgresRuntimeStatusProvider {
	return &PostgresRuntimeStatusProvider{
		StaticRuntimeStatusProvider: NewStaticRuntimeStatusProvider(repositoryMode, storageProvider, authEnabled),
		db:                          db,
	}
}

func (p *PostgresRuntimeStatusProvider) Ready(ctx context.Context) (int, map[string]any) {
	readyCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	checks := []map[string]any{
		{"name": "repository", "status": "up", "mode": p.repositoryMode},
		{"name": "storage", "status": "up", "provider": p.storageProvider},
		{"name": "auth", "status": "up", "enabled": p.authEnabled},
	}
	status := "ready"
	code := 200

	if p.db == nil {
		status = "degraded"
		code = 503
		checks[0]["status"] = "down"
		checks[0]["detail"] = "database handle is not configured"
	} else if err := p.db.PingContext(readyCtx); err != nil {
		status = "degraded"
		code = 503
		checks[0]["status"] = "down"
		checks[0]["detail"] = truncateReadinessError(err)
	}

	return code, map[string]any{
		"status": status,
		"checks": checks,
	}
}

func (p *PostgresRuntimeStatusProvider) RuntimeMetrics(ctx context.Context) map[string]any {
	metrics := p.StaticRuntimeStatusProvider.RuntimeMetrics(ctx)
	if p.db == nil {
		metrics["repositoryStatus"] = "down"
		return metrics
	}

	type authStats struct {
		active             int
		inactive           int
		mustChangePassword int
		locked             int
	}
	type sessionStats struct {
		active  int
		expired int
		revoked int
	}
	type outboxStats struct {
		claimable    int
		pending      int
		deadLettered int
	}

	auth := authStats{}
	sessions := sessionStats{}
	outbox := outboxStats{}

	statsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	authErr := p.db.QueryRowContext(statsCtx, `
SELECT
  COUNT(*) FILTER (WHERE is_active = TRUE),
  COUNT(*) FILTER (WHERE is_active = FALSE),
  COUNT(*) FILTER (WHERE must_change_password = TRUE),
  COUNT(*) FILTER (WHERE locked_until IS NOT NULL AND locked_until > NOW())
FROM metaldocs.auth_identities
`).Scan(&auth.active, &auth.inactive, &auth.mustChangePassword, &auth.locked)
	sessionErr := p.db.QueryRowContext(statsCtx, `
SELECT
  COUNT(*) FILTER (WHERE revoked_at IS NULL AND expires_at > NOW()),
  COUNT(*) FILTER (WHERE expires_at <= NOW()),
  COUNT(*) FILTER (WHERE revoked_at IS NOT NULL)
FROM metaldocs.auth_sessions
`).Scan(&sessions.active, &sessions.expired, &sessions.revoked)
	outboxErr := p.db.QueryRowContext(statsCtx, `
SELECT
  COUNT(*) FILTER (WHERE published_at IS NULL AND dead_lettered_at IS NULL AND (next_attempt_at IS NULL OR next_attempt_at <= NOW())),
  COUNT(*) FILTER (WHERE published_at IS NULL AND dead_lettered_at IS NULL),
  COUNT(*) FILTER (WHERE dead_lettered_at IS NOT NULL)
FROM metaldocs.outbox_events
`).Scan(&outbox.claimable, &outbox.pending, &outbox.deadLettered)

	metrics["repositoryStatus"] = "up"
	metrics["auth"] = map[string]any{
		"users": map[string]any{
			"active":             auth.active,
			"inactive":           auth.inactive,
			"mustChangePassword": auth.mustChangePassword,
			"locked":             auth.locked,
		},
		"sessions": map[string]any{
			"active":  sessions.active,
			"expired": sessions.expired,
			"revoked": sessions.revoked,
		},
	}
	metrics["worker"] = map[string]any{
		"outbox": map[string]any{
			"claimable":    outbox.claimable,
			"pending":      outbox.pending,
			"deadLettered": outbox.deadLettered,
		},
	}

	errors := map[string]string{}
	if authErr != nil {
		errors["auth"] = truncateReadinessError(authErr)
	}
	if sessionErr != nil {
		errors["sessions"] = truncateReadinessError(sessionErr)
	}
	if outboxErr != nil {
		errors["worker"] = truncateReadinessError(outboxErr)
	}
	if len(errors) > 0 {
		metrics["repositoryStatus"] = "degraded"
		metrics["errors"] = errors
	}

	return metrics
}

func truncateReadinessError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 160 {
		return msg[:160]
	}
	return msg
}
