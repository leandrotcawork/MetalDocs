package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	docapp "metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

type FreezeInvoker interface {
	Freeze(ctx context.Context, tx *sql.Tx, tenantID, revisionID string, approver docapp.ApproverContext) error
}

type PDFDispatchInvoker interface {
	Dispatch(ctx context.Context, tenantID, revisionID string) error
}

// DecisionService handles approver approve/reject decisions.
type DecisionService struct {
	repo          repository.ApprovalRepository
	emitter       EventEmitter
	clock         Clock
	freezeInvoker FreezeInvoker
	pdfDispatcher PDFDispatchInvoker
}

func NewDecisionService(
	repo repository.ApprovalRepository,
	emitter EventEmitter,
	clock Clock,
	freezeInvoker FreezeInvoker,
	pdfDispatcher PDFDispatchInvoker,
) *DecisionService {
	return &DecisionService{
		repo:          repo,
		emitter:       emitter,
		clock:         clock,
		freezeInvoker: freezeInvoker,
		pdfDispatcher: pdfDispatcher,
	}
}

// SignoffRequest carries all inputs for RecordSignoff.
type SignoffRequest struct {
	TenantID         string
	InstanceID       string
	StageInstanceID  string
	ActorUserID      string
	Decision         string // "approve" or "reject"
	Comment          string
	SignatureMethod  string
	SignaturePayload map[string]any
	ContentFormData  map[string]any // current document content for hash
	Capabilities     []string
}

// SignoffResult is returned by RecordSignoff.
type SignoffResult struct {
	StageCompleted   bool
	InstanceApproved bool // true when all stages complete
	InstanceRejected bool // true when a reject decision collapses instance
}

// RecordSignoff records an approve or reject decision for the given stage instance.
// Approve path only; reject path shares this method and is gated by req.Decision.
func (s *DecisionService) RecordSignoff(ctx context.Context, db *sql.DB, req SignoffRequest) (SignoffResult, error) {
	// Step 1: validate signature payload — no float64 values.
	if err := ValidateEventPayload(req.SignaturePayload); err != nil {
		return SignoffResult{}, err
	}

	// Step 2: compute content hash.
	contentHash, err := ComputeContentHash(ContentHashInput{
		TenantID:       req.TenantID,
		DocumentID:     req.InstanceID, // keyed on instance for signoff hashing
		RevisionNumber: 0,              // signoff hash does not embed revision
		FormData:       req.ContentFormData,
	})
	if err != nil {
		return SignoffResult{}, fmt.Errorf("recordSignoff: content hash: %w", err)
	}

	// Step 3: begin transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SignoffResult{}, fmt.Errorf("recordSignoff: begin tx: %w", err)
	}

	if err := setAuthzGUC(ctx, tx, req.TenantID, req.ActorUserID); err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: %w", err)
	}

	// Step 4: load the approval instance (FOR UPDATE via LoadInstance).
	instance, err := s.repo.LoadInstance(ctx, tx, req.TenantID, req.InstanceID)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return SignoffResult{}, fmt.Errorf("recordSignoff: %w", repository.ErrNoActiveInstance)
		}
		return SignoffResult{}, fmt.Errorf("recordSignoff: load instance: %w", err)
	}
	if instance == nil {
		_ = tx.Rollback()
		return SignoffResult{}, repository.ErrNoActiveInstance
	}

	// Reject if instance is already terminal.
	if instance.Status != domain.InstanceInProgress {
		_ = tx.Rollback()
		return SignoffResult{}, repository.ErrInstanceCompleted
	}

	areaCode, err := loadDocumentAreaCode(ctx, tx, req.TenantID, instance.DocumentID)
	if err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: load document area: %w", err)
	}
	if err := authz.Require(ctx, tx, "doc.signoff", areaCode); err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, err
	}

	// Step 5: identify active stage.
	activeStage := instance.Active()
	if activeStage == nil {
		_ = tx.Rollback()
		return SignoffResult{}, domain.ErrNoActiveStage
	}
	// Ensure the requested StageInstanceID matches the active stage.
	if req.StageInstanceID != "" && activeStage.ID != req.StageInstanceID {
		_ = tx.Rollback()
		return SignoffResult{}, repository.ErrStageNotActive
	}

	// Step 6: SoD check — author cannot sign, actor cannot sign twice in same instance.
	priorSignoffs, err := s.loadPriorSignoffs(ctx, tx, req.TenantID, req.InstanceID, activeStage.ID)
	if err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: load prior signoffs: %w", err)
	}
	if err := domain.CheckSoD(instance.SubmittedBy, req.ActorUserID, priorSignoffs); err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, err
	}

	// Step 7: build the domain Signoff value object.
	sigPayload, err := marshalSignaturePayload(req.SignaturePayload)
	if err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: marshal signature payload: %w", err)
	}
	now := s.clock.Now()
	signoff, err := domain.NewSignoff(domain.SignoffParams{
		ID:                 uuid.New().String(),
		ApprovalInstanceID: req.InstanceID,
		StageInstanceID:    activeStage.ID,
		ActorUserID:        req.ActorUserID,
		ActorTenantID:      req.TenantID,
		Decision:           domain.Decision(req.Decision),
		Comment:            req.Comment,
		SignedAt:           now,
		SignatureMethod:    req.SignatureMethod,
		SignaturePayload:   sigPayload,
		ContentHash:        contentHash,
	})
	if err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: build signoff: %w", err)
	}

	// Step 8: persist the signoff, handling idempotent replay.
	insertResult, err := s.repo.InsertSignoff(ctx, tx, *signoff)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, repository.ErrActorAlreadySigned) {
			return SignoffResult{}, err
		}
		return SignoffResult{}, fmt.Errorf("recordSignoff: insert signoff: %w", err)
	}
	if insertResult.WasReplay {
		// Idempotent replay: commit and return neutral result (stage not advanced again).
		if err := tx.Commit(); err != nil {
			return SignoffResult{}, fmt.Errorf("recordSignoff: commit replay: %w", err)
		}
		return SignoffResult{}, nil
	}

	// Step 9: collect all signoffs for the active stage to evaluate quorum.
	allStageSignoffs, err := s.loadStageSignoffs(ctx, tx, activeStage.ID)
	if err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: load stage signoffs: %w", err)
	}

	// Step 10: evaluate quorum.
	approvals, rejections := splitSignoffs(allStageSignoffs)
	effectiveDenominator := domain.ComputeEffectiveDenominator(*activeStage, activeStage.EligibleActorIDs)
	if effectiveDenominator == 0 {
		// Fallback: treat every eligible actor as in scope.
		effectiveDenominator = len(activeStage.EligibleActorIDs)
	}
	if effectiveDenominator == 0 {
		// No eligible actors configured — default denominator of 1 to allow any_1_of.
		effectiveDenominator = 1
	}
	outcome := domain.EvaluateQuorum(*activeStage, approvals, rejections, effectiveDenominator)

	var result SignoffResult
	var shouldDispatchPDF bool
	var pdfTenantID string
	var pdfRevisionID string

	switch outcome {
	case domain.QuorumApprovedStage:
		// Step 11a: mark stage completed.
		if err := s.repo.UpdateStageStatus(ctx, tx, req.TenantID, activeStage.ID, domain.StageCompleted, domain.StageActive); err != nil {
			_ = tx.Rollback()
			return SignoffResult{}, fmt.Errorf("recordSignoff: complete stage: %w", err)
		}
		result.StageCompleted = true

		// Advance the in-memory instance to determine next step.
		if err := instance.AdvanceStage(); err != nil {
			_ = tx.Rollback()
			return SignoffResult{}, fmt.Errorf("recordSignoff: advance stage: %w", err)
		}

		if instance.Status == domain.InstanceApproved {
			// All stages done — complete instance.
			if err := s.repo.UpdateInstanceStatus(ctx, tx, req.TenantID, req.InstanceID,
				domain.InstanceApproved, domain.InstanceInProgress, &now); err != nil {
				_ = tx.Rollback()
				return SignoffResult{}, fmt.Errorf("recordSignoff: complete instance: %w", err)
			}
			if s.freezeInvoker != nil {
				if err := s.freezeInvoker.Freeze(ctx, tx, req.TenantID, instance.DocumentID, docapp.ApproverContext{
					UserID:       req.ActorUserID,
					Capabilities: req.Capabilities,
				}); err != nil {
					_ = tx.Rollback()
					return SignoffResult{}, fmt.Errorf("recordSignoff: freeze: %w", err)
				}
			}
			// Transition document under_review → approved.
			if _, err := tx.ExecContext(ctx, `
				UPDATE documents
				   SET status           = 'approved',
				       revision_version = revision_version + 1
				 WHERE id        = $1
				   AND tenant_id = $2
				   AND status    = 'under_review'`,
				instance.DocumentID, req.TenantID,
			); err != nil {
				_ = tx.Rollback()
				return SignoffResult{}, fmt.Errorf("recordSignoff: approve document: %w", err)
			}
			result.InstanceApproved = true
			shouldDispatchPDF = true
			pdfTenantID = req.TenantID
			pdfRevisionID = instance.DocumentID
		} else {
			// Activate the next stage that AdvanceStage marked active.
			nextStage := instance.Active()
			if nextStage != nil {
				if err := s.repo.UpdateStageStatus(ctx, tx, req.TenantID, nextStage.ID, domain.StageActive, domain.StagePending); err != nil {
					_ = tx.Rollback()
					return SignoffResult{}, fmt.Errorf("recordSignoff: activate next stage: %w", err)
				}
			}
		}

	case domain.QuorumRejectedStage:
		// Reject path — mark stage and instance rejected.
		if err := s.repo.UpdateStageStatus(ctx, tx, req.TenantID, activeStage.ID, domain.StageRejectedHere, domain.StageActive); err != nil {
			_ = tx.Rollback()
			return SignoffResult{}, fmt.Errorf("recordSignoff: reject stage: %w", err)
		}
		if err := s.repo.UpdateInstanceStatus(ctx, tx, req.TenantID, req.InstanceID,
			domain.InstanceRejected, domain.InstanceInProgress, &now); err != nil {
			_ = tx.Rollback()
			return SignoffResult{}, fmt.Errorf("recordSignoff: reject instance: %w", err)
		}
		result.InstanceRejected = true

	default:
		// QuorumPending — no stage transition needed.
	}

	// Step 12: emit governance event.
	payloadMap := map[string]any{
		"instance_id":       req.InstanceID,
		"stage_instance_id": activeStage.ID,
		"decision":          req.Decision,
		"content_hash":      contentHash,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: marshal event payload: %w", err)
	}
	event := GovernanceEvent{
		TenantID:     req.TenantID,
		EventType:    "signoff_recorded",
		ActorUserID:  req.ActorUserID,
		ResourceType: "approval_instance",
		ResourceID:   req.InstanceID,
		PayloadJSON:  json.RawMessage(payloadBytes),
		OccurredAt:   now,
	}
	if err := s.emitter.Emit(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		return SignoffResult{}, fmt.Errorf("recordSignoff: emit event: %w", err)
	}

	// Step 13: commit.
	if err := tx.Commit(); err != nil {
		return SignoffResult{}, fmt.Errorf("recordSignoff: commit: %w", err)
	}
	if shouldDispatchPDF && s.pdfDispatcher != nil {
		_ = s.pdfDispatcher.Dispatch(ctx, pdfTenantID, pdfRevisionID)
	}
	return result, nil
}

// loadPriorSignoffs fetches all signoffs for the instance EXCEPT the active stage,
// used for SoD checking (actor must not have signed in any prior stage).
func (s *DecisionService) loadPriorSignoffs(ctx context.Context, tx *sql.Tx, tenantID, instanceID, activeStageID string) ([]domain.Signoff, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, approval_instance_id, stage_instance_id,
		       actor_user_id, actor_tenant_id, decision,
		       comment, signed_at, signature_method, signature_payload, content_hash
		FROM approval_signoffs
		WHERE approval_instance_id = $1
		  AND stage_instance_id != $2
		  AND actor_tenant_id = $3
		ORDER BY signed_at ASC`,
		instanceID, activeStageID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSignoffs(rows)
}

// loadStageSignoffs fetches all signoffs for a single stage instance.
func (s *DecisionService) loadStageSignoffs(ctx context.Context, tx *sql.Tx, stageInstanceID string) ([]domain.Signoff, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, approval_instance_id, stage_instance_id,
		       actor_user_id, actor_tenant_id, decision,
		       comment, signed_at, signature_method, signature_payload, content_hash
		FROM approval_signoffs
		WHERE stage_instance_id = $1
		ORDER BY signed_at ASC`,
		stageInstanceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSignoffs(rows)
}

// scanSignoffs reads rows into domain.Signoff slice.
func scanSignoffs(rows *sql.Rows) ([]domain.Signoff, error) {
	var signoffs []domain.Signoff
	for rows.Next() {
		var (
			id                 string
			approvalInstanceID string
			stageInstanceID    string
			actorUserID        string
			actorTenantID      string
			decision           string
			comment            string
			signedAt           time.Time
			signatureMethod    string
			signaturePayload   []byte
			contentHash        string
		)
		if err := rows.Scan(
			&id, &approvalInstanceID, &stageInstanceID,
			&actorUserID, &actorTenantID, &decision,
			&comment, &signedAt, &signatureMethod, &signaturePayload, &contentHash,
		); err != nil {
			return nil, err
		}
		s, err := domain.NewSignoff(domain.SignoffParams{
			ID:                 id,
			ApprovalInstanceID: approvalInstanceID,
			StageInstanceID:    stageInstanceID,
			ActorUserID:        actorUserID,
			ActorTenantID:      actorTenantID,
			Decision:           domain.Decision(decision),
			Comment:            comment,
			SignedAt:           signedAt,
			SignatureMethod:    signatureMethod,
			SignaturePayload:   json.RawMessage(signaturePayload),
			ContentHash:        contentHash,
		})
		if err != nil {
			return nil, fmt.Errorf("scan signoff %s: %w", id, err)
		}
		signoffs = append(signoffs, *s)
	}
	return signoffs, rows.Err()
}

// splitSignoffs partitions a slice of Signoff into approvals and rejections.
func splitSignoffs(all []domain.Signoff) (approvals, rejections []domain.Signoff) {
	for _, s := range all {
		switch s.Decision() {
		case domain.DecisionApprove:
			approvals = append(approvals, s)
		case domain.DecisionReject:
			rejections = append(rejections, s)
		}
	}
	return
}

// marshalSignaturePayload converts the map to json.RawMessage.
// Returns an empty JSON object for a nil/empty map.
func marshalSignaturePayload(payload map[string]any) (json.RawMessage, error) {
	if len(payload) == 0 {
		return json.RawMessage("{}"), nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
