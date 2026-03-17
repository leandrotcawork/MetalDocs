package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type AttachmentSigner struct {
	secret []byte
}

func NewAttachmentSigner(secret string) *AttachmentSigner {
	return &AttachmentSigner{secret: []byte(secret)}
}

func (s *AttachmentSigner) Sign(attachmentID string, expiresAt time.Time) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(attachmentID + "|" + expiresAt.UTC().Format(time.RFC3339)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *AttachmentSigner) Verify(attachmentID, expiresAtRFC3339, signature string) bool {
	expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAtRFC3339))
	if err != nil {
		return false
	}
	if time.Now().UTC().After(expiresAt.UTC()) {
		return false
	}
	expected := s.Sign(attachmentID, expiresAt)
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(strings.TrimSpace(signature))))
}

func (s *AttachmentSigner) BuildDownloadURL(basePath, attachmentID string, expiresAt time.Time) string {
	values := url.Values{}
	values.Set("expiresAt", expiresAt.UTC().Format(time.RFC3339))
	values.Set("signature", s.Sign(attachmentID, expiresAt))
	return fmt.Sprintf("%s?%s", strings.TrimSpace(basePath), values.Encode())
}
