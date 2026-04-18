package formval

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Gojsonschema struct{}

func NewGojsonschema() *Gojsonschema { return &Gojsonschema{} }

func (g *Gojsonschema) Validate(schemaJSON string, formData json.RawMessage) (bool, []string, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("inline.json", bytes.NewReader([]byte(schemaJSON))); err != nil {
		return false, nil, fmt.Errorf("compile schema resource: %w", err)
	}
	schema, err := compiler.Compile("inline.json")
	if err != nil {
		return false, nil, fmt.Errorf("compile schema: %w", err)
	}

	var payload any
	if err := json.Unmarshal(formData, &payload); err != nil {
		return false, nil, fmt.Errorf("decode form_data: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		if verr, ok := err.(*jsonschema.ValidationError); ok {
			return false, flattenValidationErrors(verr), nil
		}
		return false, nil, err
	}
	return true, nil, nil
}

func flattenValidationErrors(err *jsonschema.ValidationError) []string {
	if err == nil {
		return nil
	}
	msgs := []string{err.Error()}
	for _, cause := range err.Causes {
		msgs = append(msgs, flattenValidationErrors(cause)...)
	}
	return msgs
}
