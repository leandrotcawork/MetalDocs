package domain

import (
	"strings"
	"time"
)

const (
	StatusDraft     = "DRAFT"
	StatusInReview  = "IN_REVIEW"
	StatusApproved  = "APPROVED"
	StatusPublished = "PUBLISHED"
	StatusArchived  = "ARCHIVED"
)

const (
	ClassificationPublic       = "PUBLIC"
	ClassificationInternal     = "INTERNAL"
	ClassificationConfidential = "CONFIDENTIAL"
	ClassificationRestricted   = "RESTRICTED"
)

type Document struct {
	ID                   string
	Title                string
	DocumentType         string
	DocumentProfile      string
	DocumentFamily       string
	ProfileSchemaVersion int
	ProcessArea          string
	Subject              string
	OwnerID              string
	BusinessUnit         string
	Department           string
	Classification       string
	Status               string
	Tags                 []string
	EffectiveAt          *time.Time
	ExpiryAt             *time.Time
	MetadataJSON         map[string]any
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Version struct {
	DocumentID    string
	Number        int
	Content       string
	ContentHash   string
	ChangeSummary string
	CreatedAt     time.Time
}

type Attachment struct {
	ID          string
	DocumentID  string
	FileName    string
	ContentType string
	SizeBytes   int64
	StorageKey  string
	UploadedBy  string
	CreatedAt   time.Time
}

type CreateDocumentCommand struct {
	DocumentID      string
	Title           string
	DocumentType    string
	DocumentProfile string
	ProcessArea     string
	Subject         string
	OwnerID         string
	BusinessUnit    string
	Department      string
	Classification  string
	Tags            []string
	EffectiveAt     *time.Time
	ExpiryAt        *time.Time
	MetadataJSON    map[string]any
	InitialContent  string
	TraceID         string
}

type AddVersionCommand struct {
	DocumentID    string
	Content       string
	ChangeSummary string
	TraceID       string
}

type UploadAttachmentCommand struct {
	DocumentID  string
	FileName    string
	ContentType string
	Content     []byte
	UploadedBy  string
	TraceID     string
}

type VersionDiff struct {
	DocumentID            string
	FromVersion           int
	ToVersion             int
	ContentChanged        bool
	MetadataChanged       []string
	ClassificationChanged bool
	EffectiveAtChanged    bool
	ExpiryAtChanged       bool
}

type DocumentType struct {
	Code               string
	Name               string
	Description        string
	ReviewIntervalDays int
}

type DocumentFamily struct {
	Code        string
	Name        string
	Description string
}

type DocumentProfile struct {
	Code                string
	FamilyCode          string
	Name                string
	Alias               string
	Description         string
	ReviewIntervalDays  int
	ActiveSchemaVersion int
	WorkflowProfile     string
	ApprovalRequired    bool
	RetentionDays       int
	ValidityDays        int
}

type ProcessArea struct {
	Code        string
	Name        string
	Description string
}

type Subject struct {
	Code            string
	ProcessAreaCode string
	Name            string
	Description     string
}

type DocumentProfileSchemaVersion struct {
	ProfileCode   string
	Version       int
	IsActive      bool
	MetadataRules []MetadataFieldRule
}

type DocumentProfileGovernance struct {
	ProfileCode        string
	WorkflowProfile    string
	ReviewIntervalDays int
	ApprovalRequired   bool
	RetentionDays      int
	ValidityDays       int
}

type AccessPolicy struct {
	SubjectType   string
	SubjectID     string
	ResourceScope string
	ResourceID    string
	Capability    string
	Effect        string
}

const (
	SubjectTypeUser  = "user"
	SubjectTypeRole  = "role"
	SubjectTypeGroup = "group"
)

const (
	ResourceScopeDocument     = "document"
	ResourceScopeDocumentType = "document_type"
	ResourceScopeArea         = "area"
)

const (
	CapabilityDocumentCreate            = "document.create"
	CapabilityDocumentView              = "document.view"
	CapabilityDocumentEdit              = "document.edit"
	CapabilityDocumentUploadAttachment  = "document.upload_attachment"
	CapabilityDocumentChangeWorkflow    = "document.change_workflow"
	CapabilityDocumentManagePermissions = "document.manage_permissions"
)

const (
	PolicyEffectAllow = "allow"
	PolicyEffectDeny  = "deny"
)

const DocumentProfileAliasMaxLength = 24

func DefaultDocumentTypes() []DocumentType {
	profiles := DefaultDocumentProfiles()
	out := make([]DocumentType, 0, len(profiles))
	for _, item := range profiles {
		out = append(out, DocumentType{
			Code:               item.Code,
			Name:               item.Name,
			Description:        item.Description,
			ReviewIntervalDays: item.ReviewIntervalDays,
		})
	}
	return out
}

func DefaultDocumentFamilies() []DocumentFamily {
	return []DocumentFamily{
		{Code: "procedure", Name: "Procedure", Description: "Operational procedure with controlled steps"},
		{Code: "work_instruction", Name: "Work Instruction", Description: "Detailed execution instruction"},
		{Code: "record", Name: "Record", Description: "Controlled record generated by an operational process"},
	}
}

func DefaultDocumentProfiles() []DocumentProfile {
	governanceByCode := DefaultDocumentProfileGovernanceByCode()
	return []DocumentProfile{
		{Code: "po", FamilyCode: "procedure", Name: "Procedimento Operacional", Alias: "Procedimentos", Description: "Procedimento operacional da Metal Nobre", ReviewIntervalDays: governanceByCode["po"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["po"].WorkflowProfile, ApprovalRequired: governanceByCode["po"].ApprovalRequired, RetentionDays: governanceByCode["po"].RetentionDays, ValidityDays: governanceByCode["po"].ValidityDays},
		{Code: "it", FamilyCode: "work_instruction", Name: "Instrucao de Trabalho", Alias: "Instrucoes", Description: "Instrucao de trabalho da Metal Nobre", ReviewIntervalDays: governanceByCode["it"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["it"].WorkflowProfile, ApprovalRequired: governanceByCode["it"].ApprovalRequired, RetentionDays: governanceByCode["it"].RetentionDays, ValidityDays: governanceByCode["it"].ValidityDays},
		{Code: "rg", FamilyCode: "record", Name: "Registro", Alias: "Registros", Description: "Registro operacional da Metal Nobre", ReviewIntervalDays: governanceByCode["rg"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["rg"].WorkflowProfile, ApprovalRequired: governanceByCode["rg"].ApprovalRequired, RetentionDays: governanceByCode["rg"].RetentionDays, ValidityDays: governanceByCode["rg"].ValidityDays},
	}
}

func DefaultDocumentProfilesByCode() map[string]DocumentProfile {
	items := DefaultDocumentProfiles()
	out := make(map[string]DocumentProfile, len(items))
	for _, item := range items {
		out[item.Code] = item
	}
	return out
}

func DefaultProcessAreas() []ProcessArea {
	return []ProcessArea{
		{Code: "quality", Name: "Quality", Description: "Quality management and ISO-aligned operations"},
		{Code: "marketplaces", Name: "Marketplaces", Description: "Marketplace commercial and operational routines"},
		{Code: "commercial", Name: "Commercial", Description: "Commercial and customer-facing processes"},
		{Code: "purchasing", Name: "Purchasing", Description: "Procurement and supplier acquisition processes"},
		{Code: "logistics", Name: "Logistics", Description: "Logistics, shipping and fulfillment processes"},
		{Code: "finance", Name: "Finance", Description: "Financial and fiscal control processes"},
	}
}

func DefaultSubjects() []Subject {
	return []Subject{}
}

func DefaultDocumentProfileSchemas() []DocumentProfileSchemaVersion {
	return []DocumentProfileSchemaVersion{
		DocumentProfileSchemaVersion{
			ProfileCode: "po",
			Version:     1,
			IsActive:    true,
			MetadataRules: []MetadataFieldRule{
				{Name: "procedure_code", Type: "string", Required: true},
			},
		},
		DocumentProfileSchemaVersion{
			ProfileCode: "it",
			Version:     1,
			IsActive:    true,
			MetadataRules: []MetadataFieldRule{
				{Name: "instruction_code", Type: "string", Required: true},
			},
		},
		DocumentProfileSchemaVersion{
			ProfileCode: "rg",
			Version:     1,
			IsActive:    true,
			MetadataRules: []MetadataFieldRule{
				{Name: "record_code", Type: "string", Required: true},
			},
		},
	}
}

func DefaultDocumentProfileGovernance() []DocumentProfileGovernance {
	return []DocumentProfileGovernance{
		{ProfileCode: "po", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
		{ProfileCode: "it", WorkflowProfile: "standard_approval", ReviewIntervalDays: 180, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
		{ProfileCode: "rg", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
	}
}

func DefaultDocumentProfileGovernanceByCode() map[string]DocumentProfileGovernance {
	items := DefaultDocumentProfileGovernance()
	out := make(map[string]DocumentProfileGovernance, len(items))
	for _, item := range items {
		out[item.ProfileCode] = item
	}
	return out
}

func NormalizeDocumentProfileAlias(value string) string {
	return strings.TrimSpace(value)
}

func ValidateDocumentProfileAlias(value string) error {
	alias := NormalizeDocumentProfileAlias(value)
	if alias == "" {
		return ErrInvalidDocumentProfileAlias
	}
	if len([]rune(alias)) > DocumentProfileAliasMaxLength {
		return ErrInvalidDocumentProfileAlias
	}
	return nil
}

func NormalizeDocumentProfile(profile DocumentProfile) (DocumentProfile, error) {
	profile.Code = strings.ToLower(strings.TrimSpace(profile.Code))
	profile.FamilyCode = strings.ToLower(strings.TrimSpace(profile.FamilyCode))
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.WorkflowProfile = strings.TrimSpace(profile.WorkflowProfile)
	profile.Alias = NormalizeDocumentProfileAlias(profile.Alias)
	if profile.Code == "" || profile.FamilyCode == "" || profile.Name == "" {
		return DocumentProfile{}, ErrInvalidCommand
	}
	if profile.ReviewIntervalDays <= 0 {
		return DocumentProfile{}, ErrInvalidCommand
	}
	if profile.ActiveSchemaVersion <= 0 {
		profile.ActiveSchemaVersion = 1
	}
	if err := ValidateDocumentProfileAlias(profile.Alias); err != nil {
		return DocumentProfile{}, err
	}
	return profile, nil
}

func NormalizeDocumentProfileGovernance(item DocumentProfileGovernance) (DocumentProfileGovernance, error) {
	item.ProfileCode = strings.ToLower(strings.TrimSpace(item.ProfileCode))
	item.WorkflowProfile = strings.TrimSpace(item.WorkflowProfile)
	if item.ProfileCode == "" || item.WorkflowProfile == "" {
		return DocumentProfileGovernance{}, ErrInvalidCommand
	}
	if item.ReviewIntervalDays <= 0 || item.RetentionDays < 0 || item.ValidityDays < 0 {
		return DocumentProfileGovernance{}, ErrInvalidCommand
	}
	return item, nil
}

func NormalizeProcessArea(item ProcessArea) (ProcessArea, error) {
	item.Code = strings.ToLower(strings.TrimSpace(item.Code))
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	if item.Code == "" || item.Name == "" {
		return ProcessArea{}, ErrInvalidCommand
	}
	return item, nil
}

func NormalizeSubject(item Subject) (Subject, error) {
	item.Code = strings.ToLower(strings.TrimSpace(item.Code))
	item.ProcessAreaCode = strings.ToLower(strings.TrimSpace(item.ProcessAreaCode))
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	if item.Code == "" || item.ProcessAreaCode == "" || item.Name == "" {
		return Subject{}, ErrInvalidCommand
	}
	return item, nil
}

type MetadataFieldRule struct {
	Name     string
	Type     string
	Required bool
}

func DefaultMetadataRules() map[string][]MetadataFieldRule {
	return map[string][]MetadataFieldRule{
		"po": {
			{Name: "procedure_code", Type: "string", Required: true},
		},
		"it": {
			{Name: "instruction_code", Type: "string", Required: true},
		},
		"rg": {
			{Name: "record_code", Type: "string", Required: true},
		},
	}
}
