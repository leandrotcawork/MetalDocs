// Package schemas exposes the MDDM JSON Schema as an embedded byte slice.
// The same shared/schemas/mddm.schema.json file is consumed by both the Go
// validator (this package) and the TypeScript AJV validator.
package schemas

import _ "embed"

//go:embed mddm.schema.json
var MDDMSchema []byte
