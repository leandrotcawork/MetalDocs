package resolvers

import (
	"crypto/sha256"
	"encoding/json"
)

func hashInputs(v any) ([]byte, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(payload)
	return sum[:], nil
}
