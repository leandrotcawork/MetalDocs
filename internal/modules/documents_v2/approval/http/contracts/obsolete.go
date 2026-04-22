package contracts

type ObsoleteRequest struct {
	Reason         string `json:"reason"`
	IdempotencyKey string
	IfMatchVersion int
}

func (r ObsoleteRequest) Validate() error {
	return validateRequired("reason", r.Reason)
}

type ObsoleteResponse struct {
	DocumentID string `json:"document_id"`
}
