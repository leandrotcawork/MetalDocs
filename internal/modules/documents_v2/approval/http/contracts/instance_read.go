package contracts

type InstanceResponse struct {
	InstanceID  string          `json:"instance_id"`
	DocumentID  string          `json:"document_id"`
	TenantID    string          `json:"tenant_id"`
	Status      string          `json:"status"`
	SubmittedBy string          `json:"submitted_by"`
	CreatedAt   string          `json:"created_at"`
	CompletedAt *string         `json:"completed_at,omitempty"`
	Stages      []StageInstance `json:"stages"`
	ETag        string          `json:"etag"`
}

type StageInstance struct {
	StageID   string          `json:"stage_id"`
	StageName string          `json:"stage_name"`
	Order     int             `json:"order"`
	Status    string          `json:"status"`
	Signoffs  []SignoffRecord `json:"signoffs"`
}

type SignoffRecord struct {
	SignoffID  string `json:"signoff_id"`
	ActorID    string `json:"actor_id"`
	Decision   string `json:"decision"`
	Reason     string `json:"reason,omitempty"`
	OccurredAt string `json:"occurred_at"`
}

type InboxItem struct {
	InstanceID   string `json:"instance_id"`
	DocumentID   string `json:"document_id"`
	DocumentName string `json:"document_name"`
	StageID      string `json:"stage_id"`
	StageName    string `json:"stage_name"`
	SubmittedBy  string `json:"submitted_by"`
	CreatedAt    string `json:"created_at"`
	AreaCode     string `json:"area_code"`
}

type InboxResponse struct {
	Items      []InboxItem `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
	HasMore    bool        `json:"has_more"`
}
