package approvalhttp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type submitService interface {
	SubmitRevisionForReview(ctx context.Context, db *sql.DB, req application.SubmitRequest) (application.SubmitResult, error)
}

type decisionService interface {
	RecordSignoff(ctx context.Context, db *sql.DB, req application.SignoffRequest) (application.SignoffResult, error)
}

type readService interface {
	LoadInstance(ctx context.Context, db *sql.DB, tenantID, actorID, instanceID string) (*domain.Instance, error)
	ListPendingForActor(ctx context.Context, db *sql.DB, tenantID, actorID string, areaCode string, limit, offset int) ([]domain.Instance, error)
}

type routeAdminService interface {
	Create(ctx context.Context, db *sql.DB, in application.CreateRouteInput) (application.CreateRouteResult, error)
	Update(ctx context.Context, db *sql.DB, in application.UpdateRouteInput) (application.UpdateRouteResult, error)
	Deactivate(ctx context.Context, db *sql.DB, in application.DeactivateRouteInput) (application.DeactivateRouteResult, error)
}

var (
	ErrIfMatchRequired  = errors.New("precondition: If-Match header required")
	ErrIfMatchMalformed = errors.New("precondition: If-Match header malformed; expected \"v<N>\" or \"*\"")

	errIfMatchRequired  = ErrIfMatchRequired
	errIfMatchMalformed = ErrIfMatchMalformed
)

type Handler struct {
	services    *application.Services
	db          *sql.DB
	submitSvc   submitService
	decisionSvc decisionService
	readSvc     readService
	routeAdmin  routeAdminService
}

func NewHandler(services *application.Services, db *sql.DB) *Handler {
	h := &Handler{
		services: services,
		db:       db,
	}
	if services != nil {
		h.submitSvc = services.Submit
		h.decisionSvc = services.Decision
		h.readSvc = services.Read
		h.routeAdmin = services.RouteAdmin
	}
	return h
}

func requestID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Request-ID")); id != "" {
		return id
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func actorIDFromRequest(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-User-ID")); id != "" {
		return id
	}
	return iamdomain.UserIDFromContext(r.Context())
}

func tenantIDFromReq(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
}

func parseIfMatch(header string) (int, error) {
	value := strings.TrimSpace(header)
	if value == "" {
		return -1, errIfMatchRequired
	}
	if value == "*" {
		return 0, nil
	}

	value = strings.Trim(value, "\"")
	if !strings.HasPrefix(value, "v") {
		return -1, errIfMatchMalformed
	}

	version, err := strconv.Atoi(strings.TrimPrefix(value, "v"))
	if err != nil || version < 0 {
		return -1, errIfMatchMalformed
	}
	return version, nil
}
