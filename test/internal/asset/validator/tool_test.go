// Test plan: tool.go
//
// Logic flow
// | Function     | Step | Action                                                | Effect                                                                  |
// |--------------|------|-------------------------------------------------------|-------------------------------------------------------------------------|
// | InputSchemas | 1    | One compiler with format assertion for the whole list | format uuid is enforced on arguments                                    |
// | InputSchemas | 2    | Per entry: encode the input schema as JSON            | Not encodable (a mapping key that is not text) → the encode error       |
// | InputSchemas | 3    | Register the schema as tool:<name>                    | A second tool with the same name → error naming tool:<name>             |
// | InputSchemas | 4    | Compile the schema                                    | Not a valid JSON Schema → the metaschema error; later entries untouched |
// | InputSchemas | 5    | Store the compiled schema on the entry                | Every entry can validate its arguments                                  |
//
// Scenarios
// | Number | Function     | Result  | Value class    | Input                                   | Expected                                                                       |
// |--------|--------------|---------|----------------|-----------------------------------------|--------------------------------------------------------------------------------|
// | 1      | InputSchemas | Success | Middle value   | echo with required text                 | nil; Compiled accepts {"text":"hi"} and rejects {} (steps 4, 5)                |
// | 2      | InputSchemas | Success | Middle value   | lookup with id of format uuid           | nil; Compiled accepts a UUID and rejects "abc" (step 1)                        |
// | 3      | InputSchemas | Success | Boundary value | No entries                              | nil                                                                            |
// | 4      | InputSchemas | Failure | Middle value   | property bad holds a mapping with key 1 | "json: unsupported type: map[interface {}]interface {}" (step 2)               |
// | 5      | InputSchemas | Failure | Middle value   | Two tools named echo                    | "resource for \"tool:echo\" already exists"; second echo not compiled (step 3) |
// | 6      | InputSchemas | Failure | Middle value   | Type "5", then a valid entry            | Metaschema error for tool:bad; the valid entry not compiled (step 4)           |
//
// Branch coverage
// | Function     | Condition                    | True | False |
// |--------------|------------------------------|------|-------|
// | InputSchemas | Input schema encodes         | 1, 2 | 4     |
// | InputSchemas | Tool name not yet registered | 1, 2 | 5     |
// | InputSchemas | Input schema compiles        | 1, 2 | 6     |
// | InputSchemas | Entries given                | 1, 2 | 3     |
//
// Test mapping
// | Test                                                 | Scenarios                                                  |
// |------------------------------------------------------|------------------------------------------------------------|
// | TestInputSchemas/required property enforced          | 1                                                          |
// | TestInputSchemas/uuid format asserted                | 2                                                          |
// | TestInputSchemas/no entries                          | 3                                                          |
// | TestInputSchemasInvalid/mapping key that is not text | 4                                                          |
// | TestInputSchemasInvalid/duplicate tool name          | 5                                                          |
// | TestInputSchemasInvalid/type is not a JSON type      | 6                                                          |
// | No test                                              | jsonschema.UnmarshalJSON error: input is JSON just encoded |

package validator_test

import . "github.com/ngo/precedent_mcp/internal/asset/validator"

import (
	"testing"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var echoSchema = model.ToolSchema{
	Type:       "object",
	Properties: map[string]any{"text": map[string]any{"type": "string"}},
	Required:   []string{"text"},
}

func TestInputSchemas(t *testing.T) {
	tests := []struct {
		name                  string
		givenEntries          []model.ToolEntry
		givenValidArguments   map[string]any
		givenInvalidArguments map[string]any
	}{
		{"required property enforced",
			[]model.ToolEntry{{Name: "echo", InputSchema: echoSchema}},
			map[string]any{"text": "hi"}, map[string]any{}},
		{"uuid format asserted",
			[]model.ToolEntry{{Name: "lookup", InputSchema: model.ToolSchema{Type: "object",
				Properties: map[string]any{"id": map[string]any{"type": "string", "format": "uuid"}}}}},
			map[string]any{"id": "11111111-1111-1111-1111-111111111111"}, map[string]any{"id": "abc"}},
		{"no entries", []model.ToolEntry{}, nil, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := InputSchemas()(test.givenEntries)

			require.NoError(t, err)
			for _, entry := range test.givenEntries {
				require.NotNil(t, entry.Compiled, entry.Name)
				assert.NoError(t, entry.Compiled.Validate(test.givenValidArguments))
				assert.Error(t, entry.Compiled.Validate(test.givenInvalidArguments))
			}
		})
	}
}

func TestInputSchemasInvalid(t *testing.T) {
	tests := []struct {
		name         string
		givenEntries []model.ToolEntry
		wantError    string
		wantCompiled []bool
	}{
		{"mapping key that is not text",
			[]model.ToolEntry{{Name: "echo", InputSchema: model.ToolSchema{Type: "object", Properties: map[string]any{"bad": map[any]any{1: "x"}}}}},
			"json: unsupported type: map[interface {}]interface {}",
			[]bool{false}},
		{"duplicate tool name",
			[]model.ToolEntry{{Name: "echo", InputSchema: echoSchema}, {Name: "echo", InputSchema: echoSchema}},
			`resource for "tool:echo" already exists`,
			[]bool{true, false}},
		{"type is not a JSON type",
			[]model.ToolEntry{
				{Name: "bad", InputSchema: model.ToolSchema{Type: "5", Properties: map[string]any{}}},
				{Name: "good", InputSchema: model.ToolSchema{Type: "object", Properties: map[string]any{}}},
			},
			"\"tool:bad#\" is not valid against metaschema: jsonschema validation failed with 'https://json-schema.org/draft/2020-12/schema#'\n" +
				"- at '': 'allOf' failed\n  - at '/type': 'anyOf' failed\n" +
				"    - at '/type': value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'\n" +
				"    - at '/type': got string, want array",
			[]bool{false, false}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := InputSchemas()(test.givenEntries)

			assert.EqualError(t, err, test.wantError)
			gotCompiled := make([]bool, 0, len(test.givenEntries))
			for _, entry := range test.givenEntries {
				gotCompiled = append(gotCompiled, entry.Compiled != nil)
			}
			assert.Equal(t, test.wantCompiled, gotCompiled)
		})
	}
}
