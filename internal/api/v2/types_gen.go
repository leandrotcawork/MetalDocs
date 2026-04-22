package apiv2

type ProfileResponse struct {
	Code                     string  `json:"code"`
	TenantID                 string  `json:"tenantId"`
	FamilyCode               string  `json:"familyCode"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	ReviewIntervalDays       int     `json:"reviewIntervalDays"`
	DefaultTemplateVersionID *string `json:"defaultTemplateVersionId,omitempty"`
	OwnerUserID              *string `json:"ownerUserId,omitempty"`
	EditableByRole           string  `json:"editableByRole"`
	ArchivedAt               *string `json:"archivedAt,omitempty"`
	CreatedAt                string  `json:"createdAt"`
}

type AreaResponse struct {
	Code                string  `json:"code"`
	TenantID            string  `json:"tenantId"`
	Name                string  `json:"name"`
	Description         string  `json:"description"`
	ParentCode          *string `json:"parentCode,omitempty"`
	OwnerUserID         *string `json:"ownerUserId,omitempty"`
	DefaultApproverRole *string `json:"defaultApproverRole,omitempty"`
	ArchivedAt          *string `json:"archivedAt,omitempty"`
	CreatedAt           string  `json:"createdAt"`
}

type ControlledDocumentResponse struct {
	ID                        string  `json:"id"`
	TenantID                  string  `json:"tenantId"`
	ProfileCode               string  `json:"profileCode"`
	ProcessAreaCode           string  `json:"processAreaCode"`
	DepartmentCode            *string `json:"departmentCode,omitempty"`
	Code                      string  `json:"code"`
	SequenceNum               *int    `json:"sequenceNum,omitempty"`
	Title                     string  `json:"title"`
	OwnerUserID               string  `json:"ownerUserId"`
	OverrideTemplateVersionID *string `json:"overrideTemplateVersionId,omitempty"`
	Status                    string  `json:"status"`
	CreatedAt                 string  `json:"createdAt"`
	UpdatedAt                 string  `json:"updatedAt"`
}

type MembershipResponse struct {
	UserID        string  `json:"userId"`
	TenantID      string  `json:"tenantId"`
	AreaCode      string  `json:"areaCode"`
	Role          string  `json:"role"`
	EffectiveFrom string  `json:"effectiveFrom"`
	EffectiveTo   *string `json:"effectiveTo,omitempty"`
	GrantedBy     *string `json:"grantedBy,omitempty"`
}

type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	TraceID string         `json:"trace_id,omitempty"`
}
