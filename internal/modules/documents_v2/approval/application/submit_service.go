package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

// SubmitService handles document submission for approval.
type SubmitService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}

// SubmitRequest carries all inputs for SubmitRevisionForReview.
type SubmitRequest struct {
	TenantID        string
	DocumentID      string         // UUID as string
	RouteID         string         // UUID as string
	SubmittedBy     string         // user_id
	ContentFormData map[string]any // raw form data for hashing
	RevisionVersion int            // OCC version from caller
}

// SubmitResult is returned on successful submission.
type SubmitResult struct {
	InstanceID string // UUID of created approval_instance
}

// SubmitRevisionForReview creates a new approval instance for the document revision.
// Returns repository.ErrDuplicateSubmission (unwrapped) when a concurrent submission
// with the same idempotency key already exists so callers can check via errors.Is.
func (s *SubmitService) SubmitRevisionForReview(ctx context.Context, db *sql.DB, req SubmitRequest) (SubmitResult, error) {
	// Step 1: validate payload — no float64 values.
	if err := ValidateEventPayload(req.ContentFormData); err != nil {
		return SubmitResult{}, err
	}

	// Step 2: compute content hash.
	contentHash, err := ComputeContentHash(ContentHashInput{
		TenantID:       req.TenantID,
		DocumentID:     req.DocumentID,
		RevisionNumber: req.RevisionVersion,
		FormData:       req.ContentFormData,
	})
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit: content hash: %w", err)
	}

	// Step 3: compute idempotency key.
	idempotencyKey := ComputeIdempotencyKey(IdempotencyInput{
		ActorUserID: req.SubmittedBy,
		DocumentID:  req.DocumentID,
		Timestamp:   s.clock.Now(),
	})

	// Step 4: begin transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit: begin tx: %w", err)
	}

	ctx = authz.WithCapCache(ctx)

	if err := setAuthzGUC(ctx, tx, req.TenantID, req.SubmittedBy); err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: %w", err)
	}

	areaCode, err := loadDocumentAreaCode(ctx, tx, req.TenantID, req.DocumentID)
	if err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: load document area: %w", err)
	}
	if err := authz.Require(ctx, tx, "doc.submit", areaCode); err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, err
	}

	// Step 5: load route with stages.
	route, err := s.loadRoute(ctx, tx, req.TenantID, req.RouteID)
	if err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: load route: %w", err)
	}

	// Step 6: validate route structural invariants.
	if err := route.Validate(); err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: invalid route: %w", err)
	}

	// Step 7: create the approval instance.
	instanceID := uuid.New().String()
	now := s.clock.Now()

	inst := domain.Instance{
		ID:                   instanceID,
		TenantID:             req.TenantID,
		DocumentID:           req.DocumentID,
		RouteID:              req.RouteID,
		RouteVersionSnapshot: route.Version,
		Status:               domain.InstanceInProgress,
		SubmittedBy:          req.SubmittedBy,
		SubmittedAt:          now,
		ContentHashAtSubmit:  contentHash,
		IdempotencyKey:       idempotencyKey,
		RevisionVersion:      req.RevisionVersion,
	}

	if err := s.repo.InsertInstance(ctx, tx, inst); err != nil {
		_ = tx.Rollback()
		// Pass through duplicate submission sentinel unwrapped.
		if errors.Is(err, repository.ErrDuplicateSubmission) {
			return SubmitResult{}, err
		}
		return SubmitResult{}, fmt.Errorf("submit: %w", err)
	}

	// Step 8: create stage instances.
	// First stage is active; all others start pending.
	stageInstances := make([]domain.StageInstance, len(route.Stages))
	for i, stage := range route.Stages {
		status := domain.StagePending
		var openedAt *time.Time
		if i == 0 {
			status = domain.StageActive
			openedAt = &now
		}
		stageInstances[i] = domain.StageInstance{
			ID:                         uuid.New().String(),
			ApprovalInstanceID:         instanceID,
			StageOrder:                 stage.Order,
			NameSnapshot:               stage.Name,
			RequiredRoleSnapshot:       stage.RequiredRole,
			RequiredCapabilitySnapshot: stage.RequiredCapability,
			AreaCodeSnapshot:           stage.AreaCode,
			QuorumSnapshot:             stage.Quorum,
			QuorumMSnapshot:            stage.QuorumM,
			OnEligibilityDriftSnapshot: stage.OnEligibilityDrift,
			EligibleActorIDs:           []string{}, // Phase 6 wires real IAM lookup
			Status:                     status,
			OpenedAt:                   openedAt,
		}
	}

	if err := s.repo.InsertStageInstances(ctx, tx, stageInstances); err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: %w", err)
	}

	// Step 8b: transition document draft → under_review.
	res, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status           = 'under_review',
		       revision_version = revision_version + 1
		 WHERE id               = $1
		   AND tenant_id        = $2
		   AND status           = 'draft'
		   AND revision_version = $3`,
		req.DocumentID, req.TenantID, req.RevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: transition document to under_review: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_ = tx.Rollback()
		return SubmitResult{}, repository.ErrStaleRevision
	}

	// Step 9: emit governance event.
	payloadMap := map[string]any{
		"instance_id":  instanceID,
		"route_id":     req.RouteID,
		"content_hash": contentHash,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: marshal event payload: %w", err)
	}

	event := GovernanceEvent{
		TenantID:     req.TenantID,
		EventType:    "approval_submitted",
		ActorUserID:  req.SubmittedBy,
		ResourceType: "document",
		ResourceID:   req.DocumentID,
		PayloadJSON:  json.RawMessage(payloadBytes),
		OccurredAt:   now,
	}
	if err := s.emitter.Emit(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		return SubmitResult{}, fmt.Errorf("submit: emit event: %w", err)
	}

	// Step 10: commit.
	if err := tx.Commit(); err != nil {
		return SubmitResult{}, fmt.Errorf("submit: commit: %w", err)
	}

	// Step 11: return result.
	return SubmitResult{InstanceID: instanceID}, nil
}

// loadRoute fetches an approval route and its stages from the database within the
// caller's transaction. This is intentionally not part of ApprovalRepository
// because route configuration is read-only catalogue data separate from the
// instance lifecycle that ApprovalRepository manages.
func (s *SubmitService) loadRoute(ctx context.Context, tx *sql.Tx, tenantID, routeID string) (domain.Route, error) {
	var route domain.Route
	err := tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, profile_code, version
		FROM approval_routes
		WHERE id = $1 AND tenant_id = $2`,
		routeID, tenantID,
	).Scan(&route.ID, &route.TenantID, &route.ProfileCode, &route.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Route{}, fmt.Errorf("route %s not found for tenant %s", routeID, tenantID)
		}
		return domain.Route{}, err
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT stage_order, name, required_role, required_capability,
		       area_code, quorum, quorum_m, on_eligibility_drift
		FROM approval_route_stages
		WHERE route_id = $1
		ORDER BY stage_order ASC`,
		routeID,
	)
	if err != nil {
		return domain.Route{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var stage domain.Stage
		var quorumM sql.NullInt32
		if err := rows.Scan(
			&stage.Order, &stage.Name, &stage.RequiredRole, &stage.RequiredCapability,
			&stage.AreaCode, &stage.Quorum, &quorumM, &stage.OnEligibilityDrift,
		); err != nil {
			return domain.Route{}, err
		}
		if quorumM.Valid {
			v := int(quorumM.Int32)
			stage.QuorumM = &v
		}
		route.Stages = append(route.Stages, stage)
	}
	if err := rows.Err(); err != nil {
		return domain.Route{}, err
	}

	return route, nil
}

func loadDocumentAreaCode(ctx context.Context, tx *sql.Tx, tenantID, documentID string) (string, error) {
	var areaCode string
	err := tx.QueryRowContext(ctx, `
		SELECT area_code
		FROM documents
		WHERE id = $1 AND tenant_id = $2`,
		documentID, tenantID,
	).Scan(&areaCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "tenant", nil
		}
		return "", err
	}
	if areaCode == "" {
		return "tenant", nil
	}
	return areaCode, nil
}
