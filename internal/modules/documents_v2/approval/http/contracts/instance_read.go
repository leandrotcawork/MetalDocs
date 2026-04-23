package contracts

type InstanceResponse struct {
	ID          string          `json:"id"`
	DocumentID  string          `json:"document_id"`
	RouteID     string          `json:"route_id"`
	TenantID    string          `json:"tenant_id"`
	Status      string          `json:"status"`
	SubmittedBy string          `json:"submitted_by"`
	SubmittedAt string          `json:"submitted_at"`
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
	InstanceID     string `json:"instance_id"`
	DocumentID     string `json:"document_id"`
	DocumentTitle  string `json:"document_title"`
	AreaCode       string `json:"area_code"`
	SubmittedBy    string `json:"submitted_by"`
	SubmittedAt    string `json:"submitted_at"`
	StageLabel     string `json:"stage_label"`
	QuorumProgress string `json:"quorum_progress"`
}

type InboxResponse struct {
	Items []InboxItem `json:"items"`
	Total int         `json:"total"`
}
