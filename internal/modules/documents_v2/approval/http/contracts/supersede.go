package contracts

type SupersedeRequest struct {
	SupersededDocumentID string `json:"superseded_document_id"`
	IdempotencyKey       string
	IfMatchVersion       int
}

func (r SupersedeRequest) Validate() error {
	return validateUUID("superseded_document_id", r.SupersededDocumentID)
}

type SupersedeResponse struct {
	DocumentID   string `json:"document_id"`
	SupersededID string `json:"superseded_id"`
}
