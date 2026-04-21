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
	ID                        string
	TenantID                  string
	ProfileCode               string
	ProcessAreaCode           string
	DepartmentCode            *string
	Code                      string
	SequenceNum               *int
	Title                     string
	OwnerUserID               string
	OverrideTemplateVersionID *string
	Status                    CDStatus
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
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
