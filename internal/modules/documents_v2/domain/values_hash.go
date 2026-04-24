package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func ComputeValuesHash(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		v, _ := json.Marshal(values[k])
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write(v)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
