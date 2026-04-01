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

const (
	ContentSourceNative     = "native"
	ContentSourceDocxUpload = "docx_upload"
)

const (
	AudienceModeInternal   = "INTERNAL"
	AudienceModeDepartment = "DEPARTMENT"
	AudienceModeAreas      = "AREAS"
	AudienceModeExplicit   = "EXPLICIT"
)

type Document struct {
	ID                   string
	Title                string
	DocumentType         string
	DocumentProfile      string
	DocumentFamily       string
	DocumentSequence     int
	DocumentCode         string
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
	DocumentID       string
	Number           int
	Content          string
	ContentHash      string
	ChangeSummary    string
	ContentSource    string
	NativeContent    DocumentValues
	BodyBlocks       []EtapaBody
	DocxStorageKey   string
	PdfStorageKey    string
	TextContent      string
	FileSizeBytes    int64
	OriginalFilename string
	PageCount        int
	CreatedAt        time.Time
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
	Audience        *DocumentAudience
	Tags            []string
	EffectiveAt     *time.Time
	ExpiryAt        *time.Time
	MetadataJSON    map[string]any
	InitialContent  string
	TraceID         string
}

type DocumentAudience struct {
	Mode             string
	DepartmentCodes  []string
	ProcessAreaCodes []string
	RoleCodes        []string
	UserIDs          []string
}

type AddVersionCommand struct {
	DocumentID    string
	Content       string
	ChangeSummary string
	TraceID       string
}

type SaveEtapaBodyCommand struct {
	DocumentID    string
	VersionNumber int
	StepIndex     int
	Blocks        []RichBlock
	TraceID       string
}

type SaveNativeContentCommand struct {
	DocumentID string
	Content    map[string]any
	TraceID    string
}

type UploadDocxContentCommand struct {
	DocumentID string
	FileName   string
	Content    []byte
	TraceID    string
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

type DocumentTypeDefinition struct {
	Key           string
	Name          string
	ActiveVersion int
	Schema        DocumentTypeSchema
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

type DocumentDepartment struct {
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
	ContentSchema map[string]any
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

func DefaultDocumentDepartments() []DocumentDepartment {
	return []DocumentDepartment{
		{Code: "sgq", Name: "SGQ", Description: "Sistema de Gestao da Qualidade"},
		{Code: "operacoes", Name: "Operacoes", Description: "Operacao e execucao do processo"},
		{Code: "manutencao", Name: "Manutencao", Description: "Manutencao de equipamentos e infraestrutura"},
		{Code: "compras", Name: "Compras", Description: "Compras e suprimentos"},
		{Code: "logistica", Name: "Logistica", Description: "Logistica e expedicao"},
		{Code: "financeiro", Name: "Financeiro", Description: "Financeiro e controladoria"},
		{Code: "comercial", Name: "Comercial", Description: "Relacionamento com clientes e vendas"},
		{Code: "rh", Name: "RH", Description: "Recursos humanos"},
		{Code: "ti", Name: "TI", Description: "Tecnologia da informacao"},
	}
}

func DefaultSubjects() []Subject {
	return []Subject{}
}

func DefaultDocumentProfileSchemas() []DocumentProfileSchemaVersion {
	return []DocumentProfileSchemaVersion{
		DocumentProfileSchemaVersion{
			ProfileCode:   "po",
			Version:       1,
			IsActive:      true,
			MetadataRules: []MetadataFieldRule{},
			ContentSchema: defaultContentSchemaPO(),
		},
		DocumentProfileSchemaVersion{
			ProfileCode:   "it",
			Version:       1,
			IsActive:      true,
			MetadataRules: []MetadataFieldRule{},
			ContentSchema: defaultContentSchemaIT(),
		},
		DocumentProfileSchemaVersion{
			ProfileCode:   "rg",
			Version:       1,
			IsActive:      true,
			MetadataRules: []MetadataFieldRule{},
			ContentSchema: defaultContentSchemaRG(),
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

func NormalizeDocumentProfileSchemaVersion(item DocumentProfileSchemaVersion) (DocumentProfileSchemaVersion, error) {
	item.ProfileCode = strings.ToLower(strings.TrimSpace(item.ProfileCode))
	if item.ProfileCode == "" || item.Version <= 0 {
		return DocumentProfileSchemaVersion{}, ErrInvalidCommand
	}
	normalizedRules := make([]MetadataFieldRule, 0, len(item.MetadataRules))
	for _, rule := range item.MetadataRules {
		name := strings.TrimSpace(rule.Name)
		ruleType := strings.TrimSpace(rule.Type)
		if name == "" || ruleType == "" {
			return DocumentProfileSchemaVersion{}, ErrInvalidCommand
		}
		normalizedRules = append(normalizedRules, MetadataFieldRule{
			Name:     name,
			Type:     ruleType,
			Required: rule.Required,
		})
	}
	item.MetadataRules = normalizedRules
	if item.ContentSchema == nil {
		item.ContentSchema = map[string]any{}
	}
	return item, nil
}

func defaultContentSchemaPO() map[string]any {
	return map[string]any{
		"profile": "po",
		"sections": []map[string]any{
			{
				"key":         "identification",
				"title":       "Identificacao",
				"description": "Objetivo, escopo e responsavel.",
				"fields": []map[string]any{
					{"key": "objetivo", "label": "Objetivo", "type": "textarea", "required": true},
					{"key": "escopo", "label": "Escopo", "type": "textarea"},
					{"key": "responsavel", "label": "Responsavel", "type": "text"},
					{"key": "participantes", "label": "Participantes", "type": "array", "itemType": "text"},
					{"key": "canal", "label": "Canal", "type": "select", "options": []string{"Balcao", "WhatsApp", "Externo", "E-commerce"}},
				},
			},
			{
				"key":         "io",
				"title":       "Entradas e saidas",
				"description": "Entradas, saidas e sistemas.",
				"fields": []map[string]any{
					{"key": "entradas", "label": "Entradas", "type": "textarea"},
					{"key": "saidas", "label": "Saidas", "type": "textarea"},
					{"key": "documentos", "label": "Documentos", "type": "array", "itemType": "text"},
					{"key": "sistemas", "label": "Sistemas", "type": "array", "itemType": "text"},
				},
			},
			{
				"key":         "process",
				"title":       "Processo",
				"description": "Etapas e pontos de controle.",
				"fields": []map[string]any{
					{
						"key":   "etapas",
						"label": "Etapas",
						"type":  "table",
						"columns": []map[string]any{
							{"key": "num", "label": "#", "type": "number"},
							{"key": "etapa", "label": "Etapa", "type": "text"},
							{"key": "blocks", "label": "Blocos", "type": "rich_blocks"},
							{"key": "responsavel", "label": "Responsavel", "type": "text"},
							{"key": "prazo", "label": "Prazo", "type": "text"},
							{"key": "observacao", "label": "Observacao", "type": "text"},
						},
					},
					{"key": "pontos_controle", "label": "Pontos de controle", "type": "textarea"},
					{"key": "excecoes", "label": "Excecoes", "type": "textarea"},
				},
			},
			{
				"key":         "kpis",
				"title":       "Indicadores",
				"description": "Indicadores e metas.",
				"fields": []map[string]any{
					{
						"key":   "kpis",
						"label": "Indicadores",
						"type":  "table",
						"columns": []map[string]any{
							{"key": "indicador", "label": "Indicador", "type": "text"},
							{"key": "meta", "label": "Meta", "type": "text"},
							{"key": "frequencia", "label": "Frequencia", "type": "select", "options": []string{"Diario", "Semanal", "Mensal"}},
						},
					},
				},
			},
		},
	}
}

func defaultContentSchemaIT() map[string]any {
	return map[string]any{
		"profile": "it",
		"sections": []map[string]any{
			{
				"key":         "context",
				"title":       "Contexto",
				"description": "Quando e como executar.",
				"fields": []map[string]any{
					{"key": "cargo_executor", "label": "Cargo executor", "type": "text"},
					{"key": "quando_executar", "label": "Quando executar", "type": "textarea"},
					{"key": "tempo_estimado", "label": "Tempo estimado (min)", "type": "number"},
					{"key": "materiais", "label": "Materiais", "type": "array", "itemType": "text"},
					{"key": "resultado_esperado", "label": "Resultado esperado", "type": "textarea"},
				},
			},
			{
				"key":         "steps",
				"title":       "Passos",
				"description": "Sequencia operacional.",
				"fields": []map[string]any{
					{
						"key":   "passos",
						"label": "Passos",
						"type":  "table",
						"columns": []map[string]any{
							{"key": "num", "label": "#", "type": "number"},
							{"key": "acao", "label": "Acao", "type": "text"},
							{"key": "alerta", "label": "Alerta", "type": "text"},
						},
					},
					{"key": "pontos_atencao", "label": "Pontos de atencao", "type": "textarea"},
					{"key": "se_der_errado", "label": "Se der errado", "type": "textarea"},
				},
			},
			{
				"key":         "verification",
				"title":       "Verificacao",
				"description": "Checklist e registro.",
				"fields": []map[string]any{
					{"key": "checklist", "label": "Checklist", "type": "array", "itemType": "text"},
					{"key": "registro_gerado", "label": "Registro gerado", "type": "text"},
				},
			},
			{
				"key":         "media",
				"title":       "Midia",
				"description": "Imagens, video e anexos.",
				"fields": []map[string]any{
					{"key": "imagens", "label": "Imagens", "type": "array", "itemType": "text"},
					{"key": "video", "label": "Video", "type": "text"},
					{"key": "anexos", "label": "Anexos", "type": "array", "itemType": "text"},
				},
			},
		},
	}
}

func defaultContentSchemaRG() map[string]any {
	return map[string]any{
		"profile": "rg",
		"sections": []map[string]any{
			{
				"key":         "event",
				"title":       "Evento",
				"description": "Dados basicos do registro.",
				"fields": []map[string]any{
					{"key": "canal", "label": "Canal", "type": "select", "options": []string{"Balcao", "WhatsApp", "E-commerce"}},
				},
			},
			{
				"key":         "content",
				"title":       "Conteudo",
				"description": "Informacoes do registro.",
				"fields": []map[string]any{
					{"key": "observacoes", "label": "Observacoes", "type": "textarea"},
				},
			},
			{
				"key":         "closure",
				"title":       "Encerramento",
				"description": "Status e encerramento.",
				"fields": []map[string]any{
					{"key": "status", "label": "Status", "type": "select", "options": []string{"aberto", "concluido", "gerou_pa"}},
					{"key": "pa_vinculado", "label": "PA vinculado", "type": "text"},
					{"key": "data_encerramento", "label": "Data de encerramento", "type": "text"},
				},
			},
		},
	}
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

func NormalizeDocumentDepartment(item DocumentDepartment) (DocumentDepartment, error) {
	item.Code = strings.ToLower(strings.TrimSpace(item.Code))
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	if item.Code == "" || item.Name == "" {
		return DocumentDepartment{}, ErrInvalidCommand
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
		"po": {},
		"it": {},
		"rg": {},
	}
}
