package domain

import (
	"context"
	"time"
)

type ControlledDocumentRepository interface {
	GetByID(ctx context.Context, tenantID, id string) (*ControlledDocument, error)
	GetByCode(ctx context.Context, tenantID, profileCode, code string) (*ControlledDocument, error)
	CodeExists(ctx context.Context, tenantID, profileCode, code string) (bool, error)
	List(ctx context.Context, tenantID string, filter CDFilter) ([]ControlledDocument, error)
	Create(ctx context.Context, doc *ControlledDocument) error
	UpdateStatus(ctx context.Context, tenantID, id string, status CDStatus, updatedAt time.Time) error
}

type CDFilter struct {
	ProfileCode     *string
	ProcessAreaCode *string
	DepartmentCode  *string
	OwnerUserID     *string
	Status          *CDStatus
	Query           *string
	Limit           int
	Offset          int
}
