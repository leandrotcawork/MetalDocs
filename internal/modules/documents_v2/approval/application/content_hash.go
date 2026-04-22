package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"golang.org/x/text/unicode/norm"
)

// ErrFloatInFormData returned when form_data contains float64 values (spec rejects floats).
var ErrFloatInFormData = errors.New("content hash: float64 values are not allowed in form_data")

// ContentHashInput holds the fields canonicalized into the content hash.
type ContentHashInput struct {
	TenantID       string         `json:"tenant_id"`
	DocumentID     string         `json:"document_id"`
	RevisionNumber int            `json:"revision_number"`
	FormData       map[string]any `json:"form_data"`
}

// ComputeContentHash returns the lowercase hex SHA-256 of the canonical JSON encoding.
//
// Canonical JSON rules:
//   - Keys sorted byte-wise (UTF-8)
//   - Strings NFC-normalized
//   - No whitespace
//   - null for missing optionals (never omit)
//   - Floats rejected — ErrFloatInFormData
func ComputeContentHash(input ContentHashInput) (string, error) {
	if err := validateNoFloats(input.FormData); err != nil {
		return "", err
	}

	canonical, err := canonicalize(map[string]any{
		"tenant_id":       input.TenantID,
		"document_id":     input.DocumentID,
		"revision_number": input.RevisionNumber,
		"form_data":       input.FormData,
	})
	if err != nil {
		return "", fmt.Errorf("content hash: canonicalize: %w", err)
	}

	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalize produces deterministic, whitespace-free JSON with sorted keys.
func canonicalize(v any) ([]byte, error) {
	switch val := v.(type) {
	case nil:
		return []byte("null"), nil
	case bool:
		if val {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case int:
		return json.Marshal(val)
	case int64:
		return json.Marshal(val)
	case float64:
		return nil, ErrFloatInFormData
	case string:
		// NFC normalize.
		normalized := norm.NFC.String(val)
		return json.Marshal(normalized)
	case []any:
		out := []byte("[")
		for i, elem := range val {
			b, err := canonicalize(elem)
			if err != nil {
				return nil, err
			}
			if i > 0 {
				out = append(out, ',')
			}
			out = append(out, b...)
		}
		return append(out, ']'), nil
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		out := []byte("{")
		for i, k := range keys {
			kb, _ := json.Marshal(norm.NFC.String(k))
			vb, err := canonicalize(val[k])
			if err != nil {
				return nil, err
			}
			if i > 0 {
				out = append(out, ',')
			}
			out = append(out, kb...)
			out = append(out, ':')
			out = append(out, vb...)
		}
		return append(out, '}'), nil
	default:
		// Fallback for json.Number etc.
		return json.Marshal(val)
	}
}

func validateNoFloats(m map[string]any) error {
	return walkAny(m)
}

func walkAny(v any) error {
	switch val := v.(type) {
	case float64:
		return ErrFloatInFormData
	case map[string]any:
		for _, vv := range val {
			if err := walkAny(vv); err != nil {
				return err
			}
		}
	case []any:
		for _, vv := range val {
			if err := walkAny(vv); err != nil {
				return err
			}
		}
	}
	return nil
}
