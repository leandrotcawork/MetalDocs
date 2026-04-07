package mddm

import (
	"encoding/json"
	"fmt"

	schemas "metaldocs/shared/schemas"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var compiledSchema *jsonschema.Schema

func init() {
	c := jsonschema.NewCompiler()
	var schemaDoc any
	if err := json.Unmarshal(schemas.MDDMSchema, &schemaDoc); err != nil {
		panic(fmt.Errorf("parse mddm schema: %w", err))
	}
	if err := c.AddResource("https://metaldocs.local/schemas/mddm.schema.json", schemaDoc); err != nil {
		panic(fmt.Errorf("add schema resource: %w", err))
	}
	compiled, err := c.Compile("https://metaldocs.local/schemas/mddm.schema.json")
	if err != nil {
		panic(fmt.Errorf("compile mddm schema: %w", err))
	}
	compiledSchema = compiled
}

// ValidateMDDMBytes validates raw JSON bytes against the MDDM schema.
func ValidateMDDMBytes(data []byte) error {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	return compiledSchema.Validate(doc)
}
