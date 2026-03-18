package httpdelivery

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/platform/security"
)

type Handler struct {
	service     *application.Service
	signer      *security.AttachmentSigner
	downloadTTL time.Duration
}

type CreateDocumentRequest struct {
	Title           string         `json:"title"`
	DocumentType    string         `json:"documentType"`
	DocumentProfile string         `json:"documentProfile,omitempty"`
	ProcessArea     string         `json:"processArea,omitempty"`
	Subject         string         `json:"subject,omitempty"`
	OwnerID         string         `json:"ownerId"`
	BusinessUnit    string         `json:"businessUnit"`
	Department      string         `json:"department"`
	Classification  string         `json:"classification"`
	Tags            []string       `json:"tags,omitempty"`
	EffectiveAt     string         `json:"effectiveAt,omitempty"`
	ExpiryAt        string         `json:"expiryAt,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	InitialContent  string         `json:"initialContent,omitempty"`
}

type DocumentResponse struct {
	DocumentID           string   `json:"documentId"`
	Title                string   `json:"title"`
	DocumentType         string   `json:"documentType"`
	DocumentProfile      string   `json:"documentProfile"`
	DocumentFamily       string   `json:"documentFamily"`
	ProfileSchemaVersion int      `json:"profileSchemaVersion"`
	ProcessArea          string   `json:"processArea,omitempty"`
	Subject              string   `json:"subject,omitempty"`
	OwnerID              string   `json:"ownerId"`
	BusinessUnit         string   `json:"businessUnit"`
	Department           string   `json:"department"`
	Classification       string   `json:"classification"`
	Status               string   `json:"status"`
	Tags                 []string `json:"tags"`
	EffectiveAt          string   `json:"effectiveAt,omitempty"`
	ExpiryAt             string   `json:"expiryAt,omitempty"`
}

type DocumentCreatedResponse struct {
	DocumentID           string `json:"documentId"`
	Version              int    `json:"version"`
	Status               string `json:"status"`
	DocumentType         string `json:"documentType"`
	DocumentProfile      string `json:"documentProfile"`
	DocumentFamily       string `json:"documentFamily"`
	ProfileSchemaVersion int    `json:"profileSchemaVersion"`
	ProcessArea          string `json:"processArea,omitempty"`
	Subject              string `json:"subject,omitempty"`
}

type VersionResponse struct {
	DocumentID    string `json:"documentId"`
	Version       int    `json:"version"`
	ContentHash   string `json:"contentHash"`
	ChangeSummary string `json:"changeSummary"`
	CreatedAt     string `json:"createdAt"`
}

type DocumentTypeResponse struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	ReviewIntervalDays int    `json:"reviewIntervalDays"`
}

type DocumentFamilyResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DocumentProfileResponse struct {
	Code                string `json:"code"`
	FamilyCode          string `json:"familyCode"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	ReviewIntervalDays  int    `json:"reviewIntervalDays"`
	ActiveSchemaVersion int    `json:"activeSchemaVersion"`
	WorkflowProfile     string `json:"workflowProfile"`
	ApprovalRequired    bool   `json:"approvalRequired"`
	RetentionDays       int    `json:"retentionDays"`
	ValidityDays        int    `json:"validityDays"`
}

type DocumentProfileSchemaResponse struct {
	ProfileCode   string                     `json:"profileCode"`
	Version       int                        `json:"version"`
	IsActive      bool                       `json:"isActive"`
	MetadataRules []domain.MetadataFieldRule `json:"metadataRules"`
}

type DocumentProfileGovernanceResponse struct {
	ProfileCode        string `json:"profileCode"`
	WorkflowProfile    string `json:"workflowProfile"`
	ReviewIntervalDays int    `json:"reviewIntervalDays"`
	ApprovalRequired   bool   `json:"approvalRequired"`
	RetentionDays      int    `json:"retentionDays"`
	ValidityDays       int    `json:"validityDays"`
}

type ProcessAreaResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SubjectResponse struct {
	Code            string `json:"code"`
	ProcessAreaCode string `json:"processAreaCode"`
	Name            string `json:"name"`
	Description     string `json:"description"`
}

type AccessPolicyRequest struct {
	SubjectType string `json:"subjectType"`
	SubjectID   string `json:"subjectId"`
	Capability  string `json:"capability"`
	Effect      string `json:"effect"`
}

type ReplaceAccessPoliciesRequest struct {
	ResourceScope string                `json:"resourceScope"`
	ResourceID    string                `json:"resourceId"`
	Policies      []AccessPolicyRequest `json:"policies"`
}

type AccessPolicyResponse struct {
	SubjectType   string `json:"subjectType"`
	SubjectID     string `json:"subjectId"`
	ResourceScope string `json:"resourceScope"`
	ResourceID    string `json:"resourceId"`
	Capability    string `json:"capability"`
	Effect        string `json:"effect"`
}

type AddVersionRequest struct {
	Content       string `json:"content"`
	ChangeSummary string `json:"changeSummary"`
}

type VersionDiffResponse struct {
	DocumentID            string   `json:"documentId"`
	FromVersion           int      `json:"fromVersion"`
	ToVersion             int      `json:"toVersion"`
	ContentChanged        bool     `json:"contentChanged"`
	MetadataChanged       []string `json:"metadataChanged"`
	ClassificationChanged bool     `json:"classificationChanged"`
	EffectiveAtChanged    bool     `json:"effectiveAtChanged"`
	ExpiryAtChanged       bool     `json:"expiryAtChanged"`
}

type AttachmentResponse struct {
	AttachmentID string `json:"attachmentId"`
	DocumentID   string `json:"documentId"`
	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType"`
	SizeBytes    int64  `json:"sizeBytes"`
	UploadedBy   string `json:"uploadedBy"`
	CreatedAt    string `json:"createdAt"`
}

type AttachmentDownloadURLResponse struct {
	AttachmentID string `json:"attachmentId"`
	DownloadURL  string `json:"downloadUrl"`
	ExpiresAt    string `json:"expiresAt"`
}

type apiErrorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
	TraceID string         `json:"trace_id"`
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{
		service:     service,
		downloadTTL: 5 * time.Minute,
	}
}

func (h *Handler) WithAttachmentDownloads(signer *security.AttachmentSigner, ttl time.Duration) *Handler {
	if signer != nil {
		h.signer = signer
	}
	if ttl > 0 {
		h.downloadTTL = ttl
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/document-families", h.handleDocumentFamilies)
	mux.HandleFunc("/api/v1/document-profiles", h.handleDocumentProfiles)
	mux.HandleFunc("/api/v1/document-profiles/", h.handleDocumentProfileSubRoutes)
	mux.HandleFunc("/api/v1/process-areas", h.handleProcessAreas)
	mux.HandleFunc("/api/v1/document-subjects", h.handleDocumentSubjects)
	mux.HandleFunc("/api/v1/document-types", h.handleDocumentTypes)
	mux.HandleFunc("/api/v1/access-policies", h.handleAccessPolicies)
	mux.HandleFunc("/api/v1/documents", h.handleDocuments)
	mux.HandleFunc("/api/v1/documents/", h.handleDocumentSubRoutes)
	mux.HandleFunc("/api/v1/attachments/", h.handleAttachmentDownloads)
}

func (h *Handler) handleDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateDocument(w, r)
	case http.MethodGet:
		h.handleListDocuments(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleDocumentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	traceID := requestTraceID(r)
	items, err := h.service.ListDocumentTypes(r.Context())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]DocumentTypeResponse, 0, len(items))
	for _, item := range items {
		out = append(out, DocumentTypeResponse{
			Code:               item.Code,
			Name:               item.Name,
			Description:        item.Description,
			ReviewIntervalDays: item.ReviewIntervalDays,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentFamilies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	traceID := requestTraceID(r)
	items, err := h.service.ListDocumentFamilies(r.Context())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]DocumentFamilyResponse, 0, len(items))
	for _, item := range items {
		out = append(out, DocumentFamilyResponse{
			Code:        item.Code,
			Name:        item.Name,
			Description: item.Description,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	traceID := requestTraceID(r)
	items, err := h.service.ListDocumentProfiles(r.Context())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]DocumentProfileResponse, 0, len(items))
	for _, item := range items {
		out = append(out, DocumentProfileResponse{
			Code:                item.Code,
			FamilyCode:          item.FamilyCode,
			Name:                item.Name,
			Description:         item.Description,
			ReviewIntervalDays:  item.ReviewIntervalDays,
			ActiveSchemaVersion: item.ActiveSchemaVersion,
			WorkflowProfile:     item.WorkflowProfile,
			ApprovalRequired:    item.ApprovalRequired,
			RetentionDays:       item.RetentionDays,
			ValidityDays:        item.ValidityDays,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentProfileSubRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/document-profiles/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Route not found", requestTraceID(r))
		return
	}

	switch {
	case parts[1] == "schema" && r.Method == http.MethodGet:
		h.handleDocumentProfileSchemas(w, r, parts[0])
	case parts[1] == "governance" && r.Method == http.MethodGet:
		h.handleDocumentProfileGovernance(w, r, parts[0])
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleDocumentProfileSchemas(w http.ResponseWriter, r *http.Request, profileCode string) {
	traceID := requestTraceID(r)
	items, err := h.service.ListDocumentProfileSchemas(r.Context(), profileCode)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]DocumentProfileSchemaResponse, 0, len(items))
	for _, item := range items {
		out = append(out, DocumentProfileSchemaResponse{
			ProfileCode:   item.ProfileCode,
			Version:       item.Version,
			IsActive:      item.IsActive,
			MetadataRules: item.MetadataRules,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentProfileGovernance(w http.ResponseWriter, r *http.Request, profileCode string) {
	traceID := requestTraceID(r)
	item, err := h.service.GetDocumentProfileGovernance(r.Context(), profileCode)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	writeJSON(w, http.StatusOK, DocumentProfileGovernanceResponse{
		ProfileCode:        item.ProfileCode,
		WorkflowProfile:    item.WorkflowProfile,
		ReviewIntervalDays: item.ReviewIntervalDays,
		ApprovalRequired:   item.ApprovalRequired,
		RetentionDays:      item.RetentionDays,
		ValidityDays:       item.ValidityDays,
	})
}

func (h *Handler) handleProcessAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	traceID := requestTraceID(r)
	items, err := h.service.ListProcessAreas(r.Context())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]ProcessAreaResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ProcessAreaResponse{
			Code:        item.Code,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentSubjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	traceID := requestTraceID(r)
	processArea := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("processArea")))
	items, err := h.service.ListSubjects(r.Context())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]SubjectResponse, 0, len(items))
	for _, item := range items {
		if processArea != "" && !strings.EqualFold(item.ProcessAreaCode, processArea) {
			continue
		}
		out = append(out, SubjectResponse{
			Code:            item.Code,
			ProcessAreaCode: item.ProcessAreaCode,
			Name:            item.Name,
			Description:     item.Description,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleAccessPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListAccessPolicies(w, r)
	case http.MethodPut:
		h.handleReplaceAccessPolicies(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleListAccessPolicies(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	resourceScope := r.URL.Query().Get("resourceScope")
	resourceID := r.URL.Query().Get("resourceId")

	items, err := h.service.ListAccessPolicies(r.Context(), resourceScope, resourceID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]AccessPolicyResponse, 0, len(items))
	for _, item := range items {
		out = append(out, AccessPolicyResponse{
			SubjectType:   item.SubjectType,
			SubjectID:     item.SubjectID,
			ResourceScope: item.ResourceScope,
			ResourceID:    item.ResourceID,
			Capability:    item.Capability,
			Effect:        item.Effect,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleReplaceAccessPolicies(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	var req ReplaceAccessPoliciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	policies := make([]domain.AccessPolicy, 0, len(req.Policies))
	for _, item := range req.Policies {
		policies = append(policies, domain.AccessPolicy{
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectID,
			Capability:  item.Capability,
			Effect:      item.Effect,
		})
	}

	if err := h.service.ReplaceAccessPolicies(r.Context(), req.ResourceScope, req.ResourceID, policies); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resourceScope": strings.ToLower(strings.TrimSpace(req.ResourceScope)),
		"resourceId":    strings.TrimSpace(req.ResourceID),
		"replacedCount": len(policies),
	})
}

func (h *Handler) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	effectiveAt, err := parseOptionalRFC3339(req.EffectiveAt)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid effectiveAt value", traceID)
		return
	}
	expiryAt, err := parseOptionalRFC3339(req.ExpiryAt)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid expiryAt value", traceID)
		return
	}

	docID, err := newDocumentID()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate document id", traceID)
		return
	}

	doc, err := h.service.CreateDocumentAuthorized(r.Context(), domain.CreateDocumentCommand{
		DocumentID:      docID,
		Title:           req.Title,
		DocumentType:    req.DocumentType,
		DocumentProfile: req.DocumentProfile,
		ProcessArea:     req.ProcessArea,
		Subject:         req.Subject,
		OwnerID:         req.OwnerID,
		BusinessUnit:    req.BusinessUnit,
		Department:      req.Department,
		Classification:  req.Classification,
		Tags:            req.Tags,
		EffectiveAt:     effectiveAt,
		ExpiryAt:        expiryAt,
		MetadataJSON:    req.Metadata,
		InitialContent:  req.InitialContent,
		TraceID:         traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, DocumentCreatedResponse{
		DocumentID:           doc.ID,
		Version:              1,
		Status:               doc.Status,
		DocumentType:         doc.DocumentType,
		DocumentProfile:      doc.DocumentProfile,
		DocumentFamily:       doc.DocumentFamily,
		ProfileSchemaVersion: doc.ProfileSchemaVersion,
		ProcessArea:          doc.ProcessArea,
		Subject:              doc.Subject,
	})
}

func (h *Handler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	docs, err := h.service.ListDocumentsAuthorized(r.Context())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		out = append(out, DocumentResponse{
			DocumentID:           doc.ID,
			Title:                doc.Title,
			DocumentType:         doc.DocumentType,
			DocumentProfile:      doc.DocumentProfile,
			DocumentFamily:       doc.DocumentFamily,
			ProfileSchemaVersion: doc.ProfileSchemaVersion,
			ProcessArea:          doc.ProcessArea,
			Subject:              doc.Subject,
			OwnerID:              doc.OwnerID,
			BusinessUnit:         doc.BusinessUnit,
			Department:           doc.Department,
			Classification:       doc.Classification,
			Status:               doc.Status,
			Tags:                 append([]string(nil), doc.Tags...),
			EffectiveAt:          formatOptionalTime(doc.EffectiveAt),
			ExpiryAt:             formatOptionalTime(doc.ExpiryAt),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentSubRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/documents/")
	parts := strings.Split(path, "/")
	if len(parts) == 3 && strings.TrimSpace(parts[0]) != "" && parts[1] == "versions" && parts[2] == "diff" && r.Method == http.MethodGet {
		h.handleDiffVersions(w, r, parts[0])
		return
	}
	if len(parts) == 4 && strings.TrimSpace(parts[0]) != "" && parts[1] == "attachments" && parts[3] == "download-url" && r.Method == http.MethodGet {
		h.handleCreateAttachmentDownloadURL(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "versions" && r.Method == http.MethodPost {
		h.handleAddVersion(w, r, parts[0])
		return
	}
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "attachments" {
		switch r.Method {
		case http.MethodPost:
			h.handleUploadAttachment(w, r, parts[0])
		case http.MethodGet:
			h.handleListAttachments(w, r, parts[0])
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.handleGetDocument(w, r, parts[0])
		return
	}
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "versions" || r.Method != http.MethodGet {
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Route not found", requestTraceID(r))
		return
	}

	h.handleListVersions(w, r, parts[0])
}

func (h *Handler) handleGetDocument(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	doc, err := h.service.GetDocumentAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, DocumentResponse{
		DocumentID:           doc.ID,
		Title:                doc.Title,
		DocumentType:         doc.DocumentType,
		DocumentProfile:      doc.DocumentProfile,
		DocumentFamily:       doc.DocumentFamily,
		ProfileSchemaVersion: doc.ProfileSchemaVersion,
		ProcessArea:          doc.ProcessArea,
		Subject:              doc.Subject,
		OwnerID:              doc.OwnerID,
		BusinessUnit:         doc.BusinessUnit,
		Department:           doc.Department,
		Classification:       doc.Classification,
		Status:               doc.Status,
		Tags:                 append([]string(nil), doc.Tags...),
		EffectiveAt:          formatOptionalTime(doc.EffectiveAt),
		ExpiryAt:             formatOptionalTime(doc.ExpiryAt),
	})
}

func (h *Handler) handleListVersions(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	versions, err := h.service.ListVersions(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	items := make([]VersionResponse, 0, len(versions))
	for _, v := range versions {
		items = append(items, VersionResponse{
			DocumentID:    v.DocumentID,
			Version:       v.Number,
			ContentHash:   v.ContentHash,
			ChangeSummary: v.ChangeSummary,
			CreatedAt:     v.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleAddVersion(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	var req AddVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	version, err := h.service.AddVersionAuthorized(r.Context(), domain.AddVersionCommand{
		DocumentID:    documentID,
		Content:       req.Content,
		ChangeSummary: req.ChangeSummary,
		TraceID:       traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, VersionResponse{
		DocumentID:    version.DocumentID,
		Version:       version.Number,
		ContentHash:   version.ContentHash,
		ChangeSummary: version.ChangeSummary,
		CreatedAt:     version.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleDiffVersions(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	fromVersion, err := parseRequiredIntQuery(r, "fromVersion")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid fromVersion value", traceID)
		return
	}
	toVersion, err := parseRequiredIntQuery(r, "toVersion")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid toVersion value", traceID)
		return
	}

	diff, err := h.service.DiffVersions(r.Context(), documentID, fromVersion, toVersion)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, VersionDiffResponse{
		DocumentID:            diff.DocumentID,
		FromVersion:           diff.FromVersion,
		ToVersion:             diff.ToVersion,
		ContentChanged:        diff.ContentChanged,
		MetadataChanged:       append([]string(nil), diff.MetadataChanged...),
		ClassificationChanged: diff.ClassificationChanged,
		EffectiveAtChanged:    diff.EffectiveAtChanged,
		ExpiryAtChanged:       diff.ExpiryAtChanged,
	})
}

func (h *Handler) handleUploadAttachment(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid multipart payload", traceID)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing file field", traceID)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to read attachment payload", traceID)
		return
	}
	if len(content) == 0 || len(content) > 10*1024*1024 {
		writeAPIError(w, http.StatusBadRequest, "INVALID_ATTACHMENT", "Attachment size must be between 1 byte and 10 MB", traceID)
		return
	}

	attachment, err := h.service.UploadAttachmentAuthorized(r.Context(), domain.UploadAttachmentCommand{
		DocumentID:  documentID,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Content:     content,
		UploadedBy:  iamdomain.UserIDFromContext(r.Context()),
		TraceID:     traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, mapAttachmentResponse(attachment))
}

func (h *Handler) handleListAttachments(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	items, err := h.service.ListAttachmentsAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]AttachmentResponse, 0, len(items))
	for _, item := range items {
		out = append(out, mapAttachmentResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleCreateAttachmentDownloadURL(w http.ResponseWriter, r *http.Request, documentID, attachmentID string) {
	traceID := requestTraceID(r)
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", traceID)
		return
	}
	attachment, err := h.service.GetAttachmentAuthorized(r.Context(), documentID, attachmentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	expiresAt := time.Now().UTC().Add(h.downloadTTL)
	downloadURL := h.signer.BuildDownloadURL("/api/v1/attachments/"+attachment.ID+"/content", attachment.ID, expiresAt)
	writeJSON(w, http.StatusOK, AttachmentDownloadURLResponse{
		AttachmentID: attachment.ID,
		DownloadURL:  downloadURL,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleAttachmentDownloads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.signer == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Attachment signer is not configured", requestTraceID(r))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/attachments/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "content" {
		writeAPIError(w, http.StatusNotFound, "ATTACHMENT_NOT_FOUND", "Attachment route not found", requestTraceID(r))
		return
	}

	attachmentID := parts[0]
	expiresAt := strings.TrimSpace(r.URL.Query().Get("expiresAt"))
	signature := strings.TrimSpace(r.URL.Query().Get("signature"))
	if !h.signer.Verify(attachmentID, expiresAt, signature) {
		writeAPIError(w, http.StatusForbidden, "ATTACHMENT_URL_INVALID", "Attachment URL is invalid or expired", requestTraceID(r))
		return
	}

	attachment, content, err := h.service.OpenAttachmentContent(r.Context(), attachmentID)
	if err != nil {
		h.writeDomainError(w, err, requestTraceID(r))
		return
	}

	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+attachment.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error, traceID string) {
	switch {
	case errors.Is(err, domain.ErrInvalidCommand):
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data", traceID)
	case errors.Is(err, domain.ErrInvalidDocumentType):
		writeAPIError(w, http.StatusBadRequest, "INVALID_DOCUMENT_TYPE", "Invalid document type", traceID)
	case errors.Is(err, domain.ErrInvalidAccessPolicy):
		writeAPIError(w, http.StatusBadRequest, "INVALID_ACCESS_POLICY", "Invalid access policy", traceID)
	case errors.Is(err, domain.ErrInvalidMetadata):
		writeAPIError(w, http.StatusBadRequest, "INVALID_METADATA", "Invalid metadata for document type", traceID)
	case errors.Is(err, domain.ErrDocumentNotFound):
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
	case errors.Is(err, domain.ErrDocumentAlreadyExists):
		writeAPIError(w, http.StatusConflict, "CONFLICT_ERROR", "Document already exists", traceID)
	case errors.Is(err, domain.ErrVersioningNotAllowed):
		writeAPIError(w, http.StatusConflict, "VERSIONING_NOT_ALLOWED", "Document cannot receive a new version in current status", traceID)
	case errors.Is(err, domain.ErrVersionNotFound):
		writeAPIError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "Version not found", traceID)
	case errors.Is(err, domain.ErrInvalidAttachment):
		writeAPIError(w, http.StatusBadRequest, "INVALID_ATTACHMENT", "Invalid attachment payload", traceID)
	case errors.Is(err, domain.ErrAttachmentNotFound):
		writeAPIError(w, http.StatusNotFound, "ATTACHMENT_NOT_FOUND", "Attachment not found", traceID)
	case errors.Is(err, domain.ErrAttachmentStoreUnavailable):
		writeAPIError(w, http.StatusInternalServerError, "ATTACHMENT_STORE_UNAVAILABLE", "Attachment storage is not configured", traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
	}
}

func requestTraceID(r *http.Request) string {
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id")); traceID != "" {
		return traceID
	}
	return "trace-local"
}

func newDocumentID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func writeAPIError(w http.ResponseWriter, status int, code, message, traceID string) {
	writeJSON(w, status, apiErrorEnvelope{
		Error: apiError{
			Code:    code,
			Message: message,
			Details: map[string]any{},
			TraceID: traceID,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parseRequiredIntQuery(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, domain.ErrInvalidCommand
	}
	return strconv.Atoi(raw)
}

func mapAttachmentResponse(item domain.Attachment) AttachmentResponse {
	return AttachmentResponse{
		AttachmentID: item.ID,
		DocumentID:   item.DocumentID,
		FileName:     item.FileName,
		ContentType:  item.ContentType,
		SizeBytes:    item.SizeBytes,
		UploadedBy:   item.UploadedBy,
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
	}
}
