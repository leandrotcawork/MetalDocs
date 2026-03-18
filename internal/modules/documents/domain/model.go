package domain

import "time"

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
		{Code: "policy", Name: "Policy", Description: "High-level governance and policy document"},
		{Code: "procedure", Name: "Procedure", Description: "Operational procedure with controlled steps"},
		{Code: "work_instruction", Name: "Work Instruction", Description: "Detailed execution instruction"},
		{Code: "record", Name: "Record", Description: "Controlled record generated by an operational process"},
		{Code: "contract", Name: "Contract", Description: "Commercial or legal agreement"},
		{Code: "supplier_document", Name: "Supplier Document", Description: "Document received from supplier"},
		{Code: "technical_drawing", Name: "Technical Drawing", Description: "Engineering drawing or technical artifact"},
		{Code: "certificate", Name: "Certificate", Description: "Certificate with issuer and validity context"},
		{Code: "report", Name: "Report", Description: "Periodic or ad-hoc report"},
		{Code: "form", Name: "Form", Description: "Structured business form"},
		{Code: "manual", Name: "Manual", Description: "Reference or guidance manual"},
	}
}

func DefaultDocumentProfiles() []DocumentProfile {
	governanceByCode := DefaultDocumentProfileGovernanceByCode()
	return []DocumentProfile{
		{Code: "policy", FamilyCode: "policy", Name: "Policy", Description: "High-level governance and policy document", ReviewIntervalDays: governanceByCode["policy"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["policy"].WorkflowProfile, ApprovalRequired: governanceByCode["policy"].ApprovalRequired, RetentionDays: governanceByCode["policy"].RetentionDays, ValidityDays: governanceByCode["policy"].ValidityDays},
		{Code: "procedure", FamilyCode: "procedure", Name: "Procedure", Description: "Operational procedure with controlled steps", ReviewIntervalDays: governanceByCode["procedure"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["procedure"].WorkflowProfile, ApprovalRequired: governanceByCode["procedure"].ApprovalRequired, RetentionDays: governanceByCode["procedure"].RetentionDays, ValidityDays: governanceByCode["procedure"].ValidityDays},
		{Code: "work_instruction", FamilyCode: "work_instruction", Name: "Work Instruction", Description: "Detailed execution instruction", ReviewIntervalDays: governanceByCode["work_instruction"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["work_instruction"].WorkflowProfile, ApprovalRequired: governanceByCode["work_instruction"].ApprovalRequired, RetentionDays: governanceByCode["work_instruction"].RetentionDays, ValidityDays: governanceByCode["work_instruction"].ValidityDays},
		{Code: "contract", FamilyCode: "contract", Name: "Contract", Description: "Commercial or legal agreement", ReviewIntervalDays: governanceByCode["contract"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["contract"].WorkflowProfile, ApprovalRequired: governanceByCode["contract"].ApprovalRequired, RetentionDays: governanceByCode["contract"].RetentionDays, ValidityDays: governanceByCode["contract"].ValidityDays},
		{Code: "supplier_document", FamilyCode: "supplier_document", Name: "Supplier Document", Description: "Document received from supplier", ReviewIntervalDays: governanceByCode["supplier_document"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["supplier_document"].WorkflowProfile, ApprovalRequired: governanceByCode["supplier_document"].ApprovalRequired, RetentionDays: governanceByCode["supplier_document"].RetentionDays, ValidityDays: governanceByCode["supplier_document"].ValidityDays},
		{Code: "technical_drawing", FamilyCode: "technical_drawing", Name: "Technical Drawing", Description: "Engineering drawing or technical artifact", ReviewIntervalDays: governanceByCode["technical_drawing"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["technical_drawing"].WorkflowProfile, ApprovalRequired: governanceByCode["technical_drawing"].ApprovalRequired, RetentionDays: governanceByCode["technical_drawing"].RetentionDays, ValidityDays: governanceByCode["technical_drawing"].ValidityDays},
		{Code: "certificate", FamilyCode: "certificate", Name: "Certificate", Description: "Certificate with issuer and validity context", ReviewIntervalDays: governanceByCode["certificate"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["certificate"].WorkflowProfile, ApprovalRequired: governanceByCode["certificate"].ApprovalRequired, RetentionDays: governanceByCode["certificate"].RetentionDays, ValidityDays: governanceByCode["certificate"].ValidityDays},
		{Code: "report", FamilyCode: "report", Name: "Report", Description: "Periodic or ad-hoc report", ReviewIntervalDays: governanceByCode["report"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["report"].WorkflowProfile, ApprovalRequired: governanceByCode["report"].ApprovalRequired, RetentionDays: governanceByCode["report"].RetentionDays, ValidityDays: governanceByCode["report"].ValidityDays},
		{Code: "form", FamilyCode: "form", Name: "Form", Description: "Structured business form", ReviewIntervalDays: governanceByCode["form"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["form"].WorkflowProfile, ApprovalRequired: governanceByCode["form"].ApprovalRequired, RetentionDays: governanceByCode["form"].RetentionDays, ValidityDays: governanceByCode["form"].ValidityDays},
		{Code: "manual", FamilyCode: "manual", Name: "Manual", Description: "Reference or guidance manual", ReviewIntervalDays: governanceByCode["manual"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["manual"].WorkflowProfile, ApprovalRequired: governanceByCode["manual"].ApprovalRequired, RetentionDays: governanceByCode["manual"].RetentionDays, ValidityDays: governanceByCode["manual"].ValidityDays},
		{Code: "po", FamilyCode: "procedure", Name: "PO", Description: "Procedimento operacional da Metal Nobre", ReviewIntervalDays: governanceByCode["po"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["po"].WorkflowProfile, ApprovalRequired: governanceByCode["po"].ApprovalRequired, RetentionDays: governanceByCode["po"].RetentionDays, ValidityDays: governanceByCode["po"].ValidityDays},
		{Code: "it", FamilyCode: "work_instruction", Name: "IT", Description: "Instrucao de trabalho da Metal Nobre", ReviewIntervalDays: governanceByCode["it"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["it"].WorkflowProfile, ApprovalRequired: governanceByCode["it"].ApprovalRequired, RetentionDays: governanceByCode["it"].RetentionDays, ValidityDays: governanceByCode["it"].ValidityDays},
		{Code: "rg", FamilyCode: "record", Name: "RG", Description: "Registro operacional da Metal Nobre", ReviewIntervalDays: governanceByCode["rg"].ReviewIntervalDays, ActiveSchemaVersion: 1, WorkflowProfile: governanceByCode["rg"].WorkflowProfile, ApprovalRequired: governanceByCode["rg"].ApprovalRequired, RetentionDays: governanceByCode["rg"].RetentionDays, ValidityDays: governanceByCode["rg"].ValidityDays},
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
	rulesByType := DefaultMetadataRules()
	out := make([]DocumentProfileSchemaVersion, 0, len(rulesByType))
	for profileCode, rules := range rulesByType {
		copiedRules := make([]MetadataFieldRule, len(rules))
		copy(copiedRules, rules)
		out = append(out, DocumentProfileSchemaVersion{
			ProfileCode:   profileCode,
			Version:       1,
			IsActive:      true,
			MetadataRules: copiedRules,
		})
	}
	out = append(out,
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
	)
	return out
}

func DefaultDocumentProfileGovernance() []DocumentProfileGovernance {
	return []DocumentProfileGovernance{
		{ProfileCode: "policy", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 0, ValidityDays: 0},
		{ProfileCode: "procedure", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 0, ValidityDays: 0},
		{ProfileCode: "work_instruction", WorkflowProfile: "standard_approval", ReviewIntervalDays: 180, ApprovalRequired: true, RetentionDays: 0, ValidityDays: 0},
		{ProfileCode: "contract", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
		{ProfileCode: "supplier_document", WorkflowProfile: "standard_approval", ReviewIntervalDays: 180, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
		{ProfileCode: "technical_drawing", WorkflowProfile: "standard_approval", ReviewIntervalDays: 180, ApprovalRequired: true, RetentionDays: 0, ValidityDays: 0},
		{ProfileCode: "certificate", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 365},
		{ProfileCode: "report", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
		{ProfileCode: "form", WorkflowProfile: "standard_approval", ReviewIntervalDays: 180, ApprovalRequired: true, RetentionDays: 3650, ValidityDays: 0},
		{ProfileCode: "manual", WorkflowProfile: "standard_approval", ReviewIntervalDays: 365, ApprovalRequired: true, RetentionDays: 0, ValidityDays: 0},
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

type MetadataFieldRule struct {
	Name     string
	Type     string
	Required bool
}

func DefaultMetadataRules() map[string][]MetadataFieldRule {
	return map[string][]MetadataFieldRule{
		"contract": {
			{Name: "counterparty", Type: "string", Required: true},
			{Name: "contract_number", Type: "string", Required: true},
			{Name: "start_date", Type: "date", Required: true},
			{Name: "end_date", Type: "date", Required: true},
		},
		"certificate": {
			{Name: "issuer", Type: "string", Required: true},
			{Name: "issue_date", Type: "date", Required: true},
			{Name: "expiry_date", Type: "date", Required: true},
		},
		"technical_drawing": {
			{Name: "drawing_code", Type: "string", Required: true},
			{Name: "revision_code", Type: "string", Required: true},
			{Name: "plant", Type: "string", Required: true},
		},
		"supplier_document": {
			{Name: "supplier_name", Type: "string", Required: true},
			{Name: "supplier_document_code", Type: "string", Required: true},
		},
		"policy": {
			{Name: "policy_code", Type: "string", Required: true},
		},
		"procedure": {
			{Name: "procedure_code", Type: "string", Required: true},
		},
		"work_instruction": {
			{Name: "instruction_code", Type: "string", Required: true},
		},
		"report": {
			{Name: "report_period", Type: "string", Required: true},
		},
		"form": {
			{Name: "form_code", Type: "string", Required: true},
		},
		"manual": {
			{Name: "manual_code", Type: "string", Required: true},
		},
	}
}
