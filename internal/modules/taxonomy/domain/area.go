package domain

import (
	"errors"
	"time"
)

type ProcessArea struct {
	Code                string
	TenantID            string
	Name                string
	Description         string
	ParentCode          *string
	OwnerUserID         *string
	DefaultApproverRole *string
	ArchivedAt          *time.Time
	CreatedAt           time.Time
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
