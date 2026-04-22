package test

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	e2eAreaCode    = "qa"
	e2eProfileCode = "seed_profile"
	e2ePassword    = "test1234"
)

type seedHandler struct {
	db               *sql.DB
	runSchedulerTick func(context.Context) error
}

type seedRequest struct {
	TenantID string   `json:"tenantId"`
	DocID    string   `json:"docId"`
	Roles    []string `json:"roles"`
}

type resetRequest struct {
	TenantID string `json:"tenantId"`
}

type governanceEventRow struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	EventType     string          `json:"event_type"`
	ActorUserID   string          `json:"actor_user_id"`
	ResourceType  string          `json:"resource_type"`
	ResourceID    string          `json:"resource_id"`
	Reason        string          `json:"reason,omitempty"`
	PayloadJSON   json.RawMessage `json:"payload_json"`
	CreatedAt     string          `json:"created_at"`
	DedupeKey     string          `json:"dedupe_key,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	InstanceID    string          `json:"instance_id,omitempty"`
	DocumentID    string          `json:"doc_id,omitempty"`
}

type seededUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type seedResponse struct {
	TenantID string `json:"tenantId"`
	DocID    string `json:"docId"`
	Users    struct {
		Author   seededUser `json:"author"`
		Reviewer seededUser `json:"reviewer"`
		Approver seededUser `json:"approver"`
		Admin    seededUser `json:"admin"`
	} `json:"users"`
	Cookies map[string]string `json:"cookies"`
}

func RegisterE2EHandlers(mux *http.ServeMux, db *sql.DB, runSchedulerTick func(context.Context) error) {
	if mux == nil || db == nil {
		return
	}
	if os.Getenv("METALDOCS_E2E") != "1" {
		return
	}

	h := &seedHandler{db: db, runSchedulerTick: runSchedulerTick}
	mux.HandleFunc("POST /internal/test/seed", h.seed)
	mux.HandleFunc("POST /internal/test/reset", h.reset)
	mux.HandleFunc("GET /internal/test/governance-events", h.governanceEvents)
	mux.HandleFunc("POST /internal/test/advance-clock", h.advanceClock)
	mux.HandleFunc("POST /internal/test/trigger-scheduler-tick", h.triggerSchedulerTick)
}

func (h *seedHandler) seed(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("METALDOCS_E2E") != "1" {
		http.NotFound(w, r)
		return
	}

	var req seedRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	tenantID := strings.TrimSpace(req.TenantID)
	docID := strings.TrimSpace(req.DocID)
	if tenantID == "" || docID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenantId and docId are required"})
		return
	}

	roles := normalizeRoles(req.Roles)

	tx, err := h.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensureTenant(r.Context(), tx, tenantID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := ensureAreaAndProfile(r.Context(), tx, tenantID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	usersByRole := map[string]seededUser{}
	cookiesByRole := map[string]string{}
	for _, role := range roles {
		user, cookieValue, createErr := upsertSeedUser(r.Context(), tx, tenantID, role)
		if createErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": createErr.Error()})
			return
		}
		usersByRole[role] = user
		cookiesByRole[role] = cookieValue
	}

	author, ok := usersByRole["author"]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "roles must include author"})
		return
	}
	admin := usersByRole["admin"]
	if admin.ID == "" {
		admin = author
	}

	tplVersionID, err := ensureTemplateVersion(r.Context(), tx, tenantID, admin.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := ensureApprovalRoute(r.Context(), tx, tenantID, admin.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := upsertDraftDocument(r.Context(), tx, tenantID, docID, tplVersionID, author.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	res := seedResponse{TenantID: tenantID, DocID: docID, Cookies: cookiesByRole}
	res.Users.Author = usersByRole["author"]
	res.Users.Reviewer = usersByRole["reviewer"]
	res.Users.Approver = usersByRole["approver"]
	res.Users.Admin = usersByRole["admin"]
	writeJSON(w, http.StatusOK, res)
}

func (h *seedHandler) reset(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("METALDOCS_E2E") != "1" {
		http.NotFound(w, r)
		return
	}

	var req resetRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenantId is required"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		`DELETE FROM approval_signoffs s USING approval_instances i WHERE s.approval_instance_id = i.id AND i.tenant_id = $1`,
		`DELETE FROM signoffs WHERE tenant_id = $1`,
		`DELETE FROM approval_stage_instances s USING approval_instances i WHERE s.approval_instance_id = i.id AND i.tenant_id = $1`,
		`DELETE FROM approval_instances WHERE tenant_id = $1`,
		`DELETE FROM approval_route_stages rs USING approval_routes r WHERE rs.route_id = r.id AND r.tenant_id = $1`,
		`DELETE FROM approval_routes WHERE tenant_id = $1`,
		`DELETE FROM governance_events WHERE tenant_id = $1`,
		`DELETE FROM documents_v2 WHERE tenant_id = $1`,
		`DELETE FROM documents WHERE tenant_id = $1`,
		`DELETE FROM metaldocs.auth_sessions s USING metaldocs.iam_users u WHERE s.user_id = u.user_id AND u.tenant_id = $1`,
		`DELETE FROM metaldocs.auth_identities i USING metaldocs.iam_users u WHERE i.user_id = u.user_id AND u.tenant_id = $1`,
		`DELETE FROM metaldocs.iam_user_roles ur USING metaldocs.iam_users u WHERE ur.user_id = u.user_id AND u.tenant_id = $1`,
		`DELETE FROM metaldocs.iam_users WHERE tenant_id = $1`,
		`DELETE FROM users WHERE tenant_id = $1`,
		`DELETE FROM tenants WHERE id = $1`,
	}

	for _, q := range statements {
		if _, execErr := tx.ExecContext(r.Context(), q, tenantID); execErr != nil {
			if isUndefinedTable(execErr) || isUndefinedColumn(execErr) {
				continue
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": execErr.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *seedHandler) governanceEvents(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("METALDOCS_E2E") != "1" {
		http.NotFound(w, r)
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenantId"))
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenantId is required"})
		return
	}

	docID := strings.TrimSpace(r.URL.Query().Get("docId"))
	instanceID := strings.TrimSpace(r.URL.Query().Get("instanceId"))

	rows, err := h.db.QueryContext(r.Context(), `
SELECT
  ge.id::text,
  ge.tenant_id::text,
  ge.event_type,
  ge.actor_user_id,
  ge.resource_type,
  ge.resource_id,
  COALESCE(ge.reason, ''),
  ge.payload_json,
  ge.created_at,
  COALESCE(ge.dedupe_key, ''),
  COALESCE(ge.correlation_id, ''),
  COALESCE(NULLIF(ge.payload_json->>'instance_id', ''), CASE WHEN ge.resource_type = 'approval_instance' THEN ge.resource_id ELSE '' END) AS instance_id,
  COALESCE(
    NULLIF(ge.payload_json->>'doc_id', ''),
    NULLIF(ge.payload_json->>'document_id', ''),
    CASE WHEN ge.resource_type = 'document' THEN ge.resource_id ELSE '' END,
    ai.document_v2_id::text
  ) AS doc_id
FROM governance_events ge
LEFT JOIN approval_instances ai
  ON ge.resource_type = 'approval_instance'
 AND ge.resource_id = ai.id::text
WHERE ge.tenant_id = $1
  AND ($2 = '' OR
       ge.resource_id = $2 OR
       ge.payload_json->>'doc_id' = $2 OR
       ge.payload_json->>'document_id' = $2 OR
       ai.document_v2_id::text = $2)
  AND ($3 = '' OR
       ge.resource_id = $3 OR
       ge.payload_json->>'instance_id' = $3)
ORDER BY ge.created_at ASC, ge.id ASC
`, tenantID, docID, instanceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	events := make([]governanceEventRow, 0)
	for rows.Next() {
		var row governanceEventRow
		var createdAt time.Time
		if scanErr := rows.Scan(
			&row.ID,
			&row.TenantID,
			&row.EventType,
			&row.ActorUserID,
			&row.ResourceType,
			&row.ResourceID,
			&row.Reason,
			&row.PayloadJSON,
			&createdAt,
			&row.DedupeKey,
			&row.CorrelationID,
			&row.InstanceID,
			&row.DocumentID,
		); scanErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": scanErr.Error()})
			return
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		events = append(events, row)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (h *seedHandler) advanceClock(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("METALDOCS_E2E") != "1" {
		http.NotFound(w, r)
		return
	}

	secondsRaw := strings.TrimSpace(r.URL.Query().Get("seconds"))
	if secondsRaw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seconds is required"})
		return
	}

	seconds, err := strconv.ParseInt(secondsRaw, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seconds must be an integer"})
		return
	}

	SetE2EClockOffset(seconds)
	writeJSON(w, http.StatusOK, map[string]any{
		"offset_seconds": int64(E2EClockOffset() / time.Second),
	})
}

func (h *seedHandler) triggerSchedulerTick(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("METALDOCS_E2E") != "1" {
		http.NotFound(w, r)
		return
	}

	if h.runSchedulerTick != nil {
		if err := h.runSchedulerTick(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	select {
	case <-time.After(6 * time.Second):
	case <-r.Context().Done():
		writeJSON(w, http.StatusRequestTimeout, map[string]string{"error": "request cancelled"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ensureTenant(ctx context.Context, tx *sql.Tx, tenantID string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO tenants (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, tenantID); err != nil {
		if isUndefinedTable(err) || isUndefinedColumn(err) {
			return nil
		}
		return fmt.Errorf("upsert tenant: %w", err)
	}
	return nil
}

func normalizeRoles(roles []string) []string {
	if len(roles) == 0 {
		return []string{"author", "reviewer", "approver", "admin"}
	}

	allowed := map[string]bool{"author": true, "reviewer": true, "approver": true, "admin": true}
	seen := map[string]bool{}
	out := make([]string, 0, 4)

	for _, role := range roles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if !allowed[normalized] || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}

	if len(out) == 0 {
		return []string{"author", "reviewer", "approver", "admin"}
	}
	return out
}

func ensureAreaAndProfile(ctx context.Context, tx *sql.Tx, tenantID string) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.document_process_areas (tenant_id, code, name, description, is_active)
VALUES ($1, $2, 'QA', 'E2E seed area', TRUE)
ON CONFLICT (tenant_id, code) DO NOTHING`, tenantID, e2eAreaCode); err != nil {
		return fmt.Errorf("seed area: %w", err)
	}

	var familyCode string
	if err := tx.QueryRowContext(ctx, `SELECT code FROM metaldocs.document_families ORDER BY code LIMIT 1`).Scan(&familyCode); err != nil {
		return fmt.Errorf("select family: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.document_profiles (tenant_id, code, family_code, name, description, review_interval_days, is_active)
VALUES ($1, $2, $3, 'Seed Profile', 'E2E seed profile', 365, TRUE)
ON CONFLICT (tenant_id, code) DO NOTHING`, tenantID, e2eProfileCode, familyCode); err != nil {
		return fmt.Errorf("seed profile: %w", err)
	}

	return nil
}

func upsertSeedUser(ctx context.Context, tx *sql.Tx, tenantID, role string) (seededUser, string, error) {
	slug := sanitizeSlug(tenantID)
	userID := fmt.Sprintf("e2e-%s-%s", role, slug)
	email := fmt.Sprintf("%s@%s.e2e", role, tenantID)
	displayName := fmt.Sprintf("E2E %s", titleCase(role))

	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.iam_users (user_id, display_name, is_active, tenant_id, deactivated_at, created_at, updated_at)
VALUES ($1, $2, TRUE, $3, NULL, now(), now())
ON CONFLICT (user_id)
DO UPDATE SET display_name = EXCLUDED.display_name,
              is_active = TRUE,
              tenant_id = EXCLUDED.tenant_id,
              deactivated_at = NULL,
              updated_at = now()`, userID, displayName, tenantID); err != nil {
		return seededUser{}, "", fmt.Errorf("upsert iam user (%s): %w", role, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(e2ePassword), bcrypt.DefaultCost)
	if err != nil {
		return seededUser{}, "", fmt.Errorf("hash password: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.auth_identities (user_id, username, email, display_name, is_active, password_hash, password_algo, must_change_password, last_login_at, failed_login_attempts, locked_until, created_at, updated_at)
VALUES ($1, $2, $3, $4, TRUE, $5, 'bcrypt', FALSE, NULL, 0, NULL, now(), now())
ON CONFLICT (user_id)
DO UPDATE SET username = EXCLUDED.username,
              email = EXCLUDED.email,
              display_name = EXCLUDED.display_name,
              is_active = TRUE,
              password_hash = EXCLUDED.password_hash,
              password_algo = 'bcrypt',
              must_change_password = FALSE,
              updated_at = now()`, userID, userID, email, displayName, string(hash)); err != nil {
		return seededUser{}, "", fmt.Errorf("upsert auth identity (%s): %w", role, err)
	}

	iamRole := mapRoleToIAM(role)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.iam_user_roles (user_id, role_code, assigned_at, assigned_by)
VALUES ($1, $2, now(), 'e2e-seed')
ON CONFLICT (user_id, role_code)
DO UPDATE SET assigned_at = now(), assigned_by = EXCLUDED.assigned_by`, userID, iamRole); err != nil {
		return seededUser{}, "", fmt.Errorf("upsert iam role (%s): %w", role, err)
	}

	membershipRole := mapRoleToMembership(role)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO user_process_areas (user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by, revoked_by)
VALUES ($1, $2, $3, $4, now(), NULL, 'e2e-seed', NULL)
ON CONFLICT (tenant_id, user_id, area_code, role) WHERE effective_to IS NULL DO NOTHING`, userID, tenantID, e2eAreaCode, membershipRole); err != nil {
		if _, fnErr := tx.ExecContext(ctx,
			`SELECT metaldocs.grant_area_membership($1::uuid, $2, $3, $4, $5)`,
			tenantID, userID, e2eAreaCode, membershipRole, "e2e-seed",
		); fnErr != nil {
			return seededUser{}, "", fmt.Errorf("grant area membership (%s): %w", role, err)
		}
	}

	cookieValue, err := createSessionValue(ctx, tx, userID)
	if err != nil {
		return seededUser{}, "", err
	}

	return seededUser{ID: userID, Email: email}, cookieValue, nil
}

func mapRoleToIAM(role string) string {
	switch role {
	case "admin":
		return "admin"
	case "reviewer", "approver":
		return "reviewer"
	default:
		return "editor"
	}
}

func mapRoleToMembership(role string) string {
	switch role {
	case "author":
		return "editor"
	case "reviewer":
		return "reviewer"
	default:
		return "approver"
	}
}

func ensureTemplateVersion(ctx context.Context, tx *sql.Tx, tenantID, actorID string) (string, error) {
	templateID := uuid.NewString()
	templateVersionID := uuid.NewString()

	if _, err := tx.ExecContext(ctx, `
INSERT INTO templates (id, tenant_id, key, name, description, current_published_version_id, created_at, updated_at, created_by)
VALUES ($1, $2, $3, 'E2E Template', 'E2E seed template', NULL, now(), now(), $4)
ON CONFLICT (tenant_id, key)
DO UPDATE SET updated_at = now(), created_by = EXCLUDED.created_by`, templateID, tenantID, "e2e-seed-template", actorID); err != nil {
		return "", fmt.Errorf("upsert template: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO template_versions (id, template_id, version_num, status, grammar_version, docx_storage_key, schema_storage_key, docx_content_hash, schema_content_hash, published_at, published_by, deprecated_at, lock_version, created_at, updated_at, created_by)
VALUES ($1, $2, 1, 'published', 1, 'seed/template.docx', 'seed/template.schema.json', 'seed-docx-hash', 'seed-schema-hash', now(), $3, NULL, 0, now(), now(), $3)
ON CONFLICT (template_id, version_num)
DO UPDATE SET status = EXCLUDED.status,
              published_at = EXCLUDED.published_at,
              published_by = EXCLUDED.published_by,
              updated_at = now()
RETURNING id`, templateVersionID, templateID, actorID); err != nil {
		if queryErr := tx.QueryRowContext(ctx,
			`SELECT id::text FROM template_versions WHERE template_id = $1 AND version_num = 1`,
			templateID,
		).Scan(&templateVersionID); queryErr != nil {
			return "", fmt.Errorf("upsert template version: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE templates SET current_published_version_id = $2, updated_at = now() WHERE id = $1`,
		templateID, templateVersionID,
	); err != nil {
		return "", fmt.Errorf("update template current version: %w", err)
	}

	return templateVersionID, nil
}

func ensureApprovalRoute(ctx context.Context, tx *sql.Tx, tenantID, actorID string) error {
	var routeID string
	err := tx.QueryRowContext(ctx, `
INSERT INTO approval_routes (tenant_id, profile_code, name, version, created_by, active)
VALUES ($1, $2, 'E2E Route', 1, $3, TRUE)
ON CONFLICT (tenant_id, profile_code)
DO UPDATE SET name = EXCLUDED.name, active = TRUE
RETURNING id::text`, tenantID, e2eProfileCode, actorID).Scan(&routeID)
	if err != nil {
		if scanErr := tx.QueryRowContext(ctx,
			`SELECT id::text FROM approval_routes WHERE tenant_id = $1 AND profile_code = $2`,
			tenantID, e2eProfileCode,
		).Scan(&routeID); scanErr != nil {
			return fmt.Errorf("upsert approval route: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM approval_route_stages WHERE route_id = $1`, routeID); err != nil {
		return fmt.Errorf("clear route stages: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO approval_route_stages (route_id, stage_order, name, required_role, required_capability, area_code, quorum, quorum_m, on_eligibility_drift)
VALUES
  ($1, 1, 'Review', 'reviewer', 'doc.signoff', $2, 'any_1_of', NULL, 'fail_stage'),
  ($1, 2, 'Approval', 'approver', 'doc.signoff', $2, 'any_1_of', NULL, 'fail_stage')`, routeID, e2eAreaCode); err != nil {
		return fmt.Errorf("insert route stages: %w", err)
	}

	return nil
}

func upsertDraftDocument(ctx context.Context, tx *sql.Tx, tenantID, docID, templateVersionID, authorID string) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO documents (id, tenant_id, template_version_id, name, status, form_data_json, created_by)
VALUES ($1, $2, $3, 'E2E Draft', 'draft', '{}'::jsonb, $4)
ON CONFLICT (id)
DO UPDATE SET tenant_id = EXCLUDED.tenant_id,
              template_version_id = EXCLUDED.template_version_id,
              name = EXCLUDED.name,
              status = 'draft',
              form_data_json = '{}'::jsonb,
              created_by = EXCLUDED.created_by,
              updated_at = now()`, docID, tenantID, templateVersionID, authorID); err != nil {
		return fmt.Errorf("upsert document: %w", err)
	}
	return nil
}

func createSessionValue(ctx context.Context, tx *sql.Tx, userID string) (string, error) {
	secret := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_SESSION_SECRET"))
	if secret == "" {
		return "", fmt.Errorf("METALDOCS_AUTH_SESSION_SECRET is required for e2e seed sessions")
	}

	ttlHours := 12
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_SESSION_TTL_HOURS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			ttlHours = parsed
		}
	}

	token, err := randomToken(32)
	if err != nil {
		return "", err
	}

	sessionID := hashToken(token)
	signature := signToken(token, secret)
	now := time.Now().UTC()

	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.auth_sessions (session_id, user_id, created_at, expires_at, revoked_at, ip_address, user_agent, last_seen_at)
VALUES ($1, $2, $3, $4, NULL, '127.0.0.1', 'e2e-seed', $3)
ON CONFLICT (session_id)
DO UPDATE SET expires_at = EXCLUDED.expires_at,
              revoked_at = NULL,
              last_seen_at = EXCLUDED.last_seen_at`,
		sessionID,
		userID,
		now,
		now.Add(time.Duration(ttlHours)*time.Hour),
	); err != nil {
		return "", fmt.Errorf("insert auth session: %w", err)
	}

	return token + "." + signature, nil
}

func sanitizeSlug(value string) string {
	out := strings.ToLower(value)
	out = strings.ReplaceAll(out, "_", "")
	out = strings.ReplaceAll(out, "-", "")
	if len(out) > 12 {
		out = out[:12]
	}
	if out == "" {
		return "seed"
	}
	return out
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func signToken(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func readJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("invalid json body: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isUndefinedTable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "sqlstate 42p01") || strings.Contains(message, "does not exist")
}

func isUndefinedColumn(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "sqlstate 42703")
}
