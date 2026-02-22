package docs

import _ "embed"

//go:embed openapi.yaml
var specYAML []byte

// SpecYAML returns the embedded OpenAPI 3.0 specification.
func SpecYAML() []byte {
	return specYAML
}
