package contracts

type CancelRequest struct {
	Reason         string `json:"reason"`
	IdempotencyKey string
	IfMatchVersion int
}

func (r CancelRequest) Validate() error {
	return validateRequired("reason", r.Reason)
}

type CancelResponse struct {
	DocumentID string `json:"document_id"`
}
