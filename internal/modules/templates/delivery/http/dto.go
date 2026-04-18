package http

type createTemplateRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createTemplateResponse struct {
	ID        string `json:"id"`
	VersionID string `json:"version_id"`
}

type saveDraftRequest struct {
	ExpectedLockVersion int    `json:"expected_lock_version"`
	DocxStorageKey      string `json:"docx_storage_key"`
	SchemaStorageKey    string `json:"schema_storage_key"`
	DocxContentHash     string `json:"docx_content_hash"`
	SchemaContentHash   string `json:"schema_content_hash"`
}

type publishRequest struct {
	DocxKey   string `json:"docx_key"`
	SchemaKey string `json:"schema_key"`
}
