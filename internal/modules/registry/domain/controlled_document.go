package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type CDStatus string

const (
	CDStatusActive     CDStatus = "active"
	CDStatusObsolete   CDStatus = "obsolete"
	CDStatusSuperseded CDStatus = "superseded"
)

type ControlledDocument struct {
	ID                        string     `json:"id"`
	TenantID                  string     `json:"tenantId"`
	ProfileCode               string     `json:"profileCode"`
	ProcessAreaCode           string     `json:"processAreaCode"`
	DepartmentCode            *string    `json:"departmentCode"`
	Code                      string     `json:"code"`
	SequenceNum               *int       `json:"sequenceNum"`
	Title                     string     `json:"title"`
	OwnerUserID               string     `json:"ownerUserId"`
	OverrideTemplateVersionID *string    `json:"overrideTemplateVersionId"`
	Status                    CDStatus   `json:"status"`
	CreatedAt                 time.Time  `json:"createdAt"`
	UpdatedAt                 time.Time  `json:"updatedAt"`
}

var (
	ErrCDNotFound               = errors.New("controlled document not found")
	ErrCDCodeTaken              = errors.New("controlled document code already taken")
	ErrCDArchivedCodeReuse      = errors.New("cannot reuse code from archived controlled document")
	ErrSequenceCounterNotFound  = errors.New("sequence counter not initialized for profile")
	ErrCDNotActive              = errors.New("controlled document is not active")
	ErrManualCodeReasonRequired = errors.New("manual code reason is required")
	ErrOverrideReasonRequired   = errors.New("override reason is required")
)

func (d ControlledDocument) IsActive() bool {
	return d.Status == CDStatusActive
}

func AutoCode(profileCode string, seq int) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(profileCode), seq)
}
