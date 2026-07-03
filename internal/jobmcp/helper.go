package jobmcp

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

func mustSchema(rawSchema []byte) *jsonschema.Schema {
	var s jsonschema.Schema
	if err := json.Unmarshal(rawSchema, &s); err != nil {
		panic(fmt.Sprintf("jobmcp tool schema: %v", err))
	}
	return &s
}
