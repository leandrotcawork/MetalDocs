package httptestsupport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	iamdomain "metaldocs/internal/modules/iam/domain"
)

// NewAuthRequest builds an httptest request with IAM auth context populated,
// mirroring what IAM middleware does in production.
func NewAuthRequest(t *testing.T, method, path string, body io.Reader, userID string, roles ...iamdomain.Role) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), userID, roles))
}
