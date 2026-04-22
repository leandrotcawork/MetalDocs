package domain

import (
	"errors"
	"time"
)

type ProcessArea struct {
	Code                string     `json:"code"`
	TenantID            string     `json:"tenantId"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	ParentCode          *string    `json:"parentCode"`
	OwnerUserID         *string    `json:"ownerUserId"`
	DefaultApproverRole *string    `json:"defaultApproverRole"`
	ArchivedAt          *time.Time `json:"archivedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
}

var (
	ErrAreaNotFound      = errors.New("process area not found")
	ErrAreaArchived      = errors.New("process area is archived")
	ErrAreaParentCycle   = errors.New("area parent assignment creates cycle")
	ErrAreaCodeImmutable = errors.New("area code is immutable")
)

func (a *ProcessArea) IsActive() bool {
	return a.ArchivedAt == nil
}

func (a *ProcessArea) Archive(now time.Time) error {
	if !a.IsActive() {
		return ErrAreaArchived
	}
	a.ArchivedAt = &now
	return nil
}
