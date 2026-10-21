// Test plan: tool.go
//
// Logic flow
// | Function        | Step | Action                                                                            | Effect                                               |
// |-----------------|------|-----------------------------------------------------------------------------------|------------------------------------------------------|
// | ToolStore.Tools | 1    | Return the cached list when present                                               | No file read until DurationCache ends                |
// | ToolStore.Tools | 2    | Read <dir>/tools.yaml                                                             | Missing file → the file system error; nothing cached |
// | ToolStore.Tools | 3    | Decode the tools list; YAML anchors resolved; keys outside the tool model dropped | Malformed YAML → the YAML error; nothing cached      |
// | ToolStore.Tools | 4    | validator.InputSchemas compiles every input schema                                | Failure → that error; nothing cached                 |
// | ToolStore.Tools | 5    | Cache and return the entries in file order                                        | Every entry carries its compiled input schema        |
//
// Scenarios
// | Number | Function        | Result  | Value class    | Input                                                                     | Expected                                                                                                                  |
// |--------|-----------------|---------|----------------|---------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------|
// | 1      | ToolStore.Tools | Success | Middle value   | testdata/tools.yaml: echo with a shared anchor in its output schema; ping | sampleTools in file order; anchor resolved; dropped keys absent; ping without output schema; Compiled set (steps 2, 3, 5) |
// | 2      | ToolStore.Tools | Success | Boundary value | tools.yaml holds tools: []                                                | Empty list (step 5)                                                                                                       |
// | 3      | ToolStore.Tools | Success | Boundary value | Read; tools.yaml removed; read again                                      | Second read returns the first list (step 1)                                                                               |
// | 4      | ToolStore.Tools | Success | Middle value   | First read fails: no tools.yaml; file written; read again                 | Second read returns the written tool; the failure was not cached (step 2)                                                 |
// | 5      | ToolStore.Tools | Failure | Boundary value | No tools.yaml                                                             | "open <dir>/tools.yaml: no such file or directory"; no list (step 2)                                                      |
// | 6      | ToolStore.Tools | Failure | Middle value   | tools.yaml with an unclosed flow sequence                                 | YAML error "yaml: line 1: did not find expected ',' or ']'"; no list (step 3)                                             |
// | 7      | ToolStore.Tools | Failure | Middle value   | input_schema type "5"                                                     | Metaschema error naming tool:echo; no list (step 4)                                                                       |
//
// Outside the unit: production assets/tools.yaml content; e2e covers it.
//
// Branch coverage
// | Function        | Condition             | True | False |
// |-----------------|-----------------------|------|-------|
// | ToolStore.Tools | Cache holds the list  | 3    | 1, 4  |
// | ToolStore.Tools | tools.yaml readable   | 1, 2 | 4, 5  |
// | ToolStore.Tools | YAML decodes          | 1, 2 | 6     |
// | ToolStore.Tools | Input schemas compile | 1, 2 | 7     |
//
// Test mapping
// | Test                                                            | Scenarios                                      |
// |-----------------------------------------------------------------|------------------------------------------------|
// | TestToolStoreTools/tools in file order                          | 1                                              |
// | TestToolStoreTools/empty tools list                             | 2                                              |
// | TestToolStoreToolsCache/second read uses the cache              | 3                                              |
// | TestToolStoreToolsFailureNotCached/file written after a failure | 4                                              |
// | TestToolStoreToolsErrors/no tools.yaml                          | 5                                              |
// | TestToolStoreToolsErrors/malformed YAML                         | 6                                              |
// | TestToolStoreToolsErrors/input schema not valid                 | 7                                              |
// | No test                                                         | Cache expiry: needs DurationCache of real time |

package asset_test

import . "github.com/ngo/precedent_mcp/internal/asset"

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleTools = []model.ToolEntry{
	{
		Name:        "echo",
		Description: "Echo the input back.",
		InputSchema: model.ToolSchema{
			Type:                 "object",
			Properties:           map[string]any{"text": map[string]any{"type": "string", "minLength": 1}},
			Required:             []string{"text"},
			AdditionalProperties: new(false),
		},
		OutputSchema: &model.OutputSchema{
			Type: "object",
			AnyOf: []model.ToolSchema{
				{Type: "object", Properties: map[string]any{"text": map[string]any{"type": "string"}}, Required: []string{"text"}},
				{Type: "object", Properties: map[string]any{"error": map[string]any{"type": "string"}}, Required: []string{"error"}},
			},
		},
	},
	{
		Name:        "ping",
		Description: "No arguments.",
		InputSchema: model.ToolSchema{Type: "object", Properties: map[string]any{}},
	},
}

const pingToolList = "tools:\n- name: ping\n  description: No arguments.\n  input_schema:\n    type: object\n    properties: {}\n"

var pingTools = []model.ToolEntry{
	{Name: "ping", Description: "No arguments.", InputSchema: model.ToolSchema{Type: "object", Properties: map[string]any{}}},
}

func TestToolStoreTools(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		want           []model.ToolEntry
	}{
		{"tools in file order", "testdata", sampleTools},
		{"empty tools list", filepath.Join("testdata", "empty"), []model.ToolEntry{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewToolStore(test.givenDirectory).Tools()

			require.NoError(t, err)
			for i := range got {
				require.NotNil(t, got[i].Compiled, got[i].Name)
				got[i].Compiled = nil
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestToolStoreToolsCache(t *testing.T) {
	tests := []struct {
		name          string
		givenToolList string
		want          []model.ToolEntry
	}{
		{"second read uses the cache", pingToolList, pingTools},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, string(env.PathToolList))
			require.NoError(t, os.WriteFile(path, []byte(test.givenToolList), 0o644))
			store := NewToolStore(directory)
			_, firstErr := store.Tools()
			require.NoError(t, firstErr)
			require.NoError(t, os.Remove(path))

			got, err := store.Tools()

			require.NoError(t, err)
			for i := range got {
				require.NotNil(t, got[i].Compiled, got[i].Name)
				got[i].Compiled = nil
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestToolStoreToolsFailureNotCached(t *testing.T) {
	tests := []struct {
		name          string
		givenToolList string
		want          []model.ToolEntry
	}{
		{"file written after a failure", pingToolList, pingTools},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			store := NewToolStore(directory)
			_, firstErr := store.Tools()
			require.Error(t, firstErr)
			require.NoError(t, os.WriteFile(filepath.Join(directory, string(env.PathToolList)), []byte(test.givenToolList), 0o644))

			got, err := store.Tools()

			require.NoError(t, err)
			for i := range got {
				require.NotNil(t, got[i].Compiled, got[i].Name)
				got[i].Compiled = nil
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestToolStoreToolsErrors(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		want           []model.ToolEntry
		wantError      string
	}{
		{"no tools.yaml", filepath.Join("testdata", "missing"), nil, "open testdata/missing/tools.yaml: no such file or directory"},
		{"malformed YAML", filepath.Join("testdata", "invalid"), nil, "yaml: line 1: did not find expected ',' or ']'"},
		{"input schema not valid", filepath.Join("testdata", "invalid-schema"), nil,
			"\"tool:echo#\" is not valid against metaschema: jsonschema validation failed with 'https://json-schema.org/draft/2020-12/schema#'\n" +
				"- at '': 'allOf' failed\n  - at '/type': 'anyOf' failed\n" +
				"    - at '/type': value must be one of 'array', 'boolean', 'integer', 'null', 'number', 'object', 'string'\n" +
				"    - at '/type': got string, want array"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewToolStore(test.givenDirectory).Tools()

			assert.EqualError(t, err, test.wantError)
			assert.Equal(t, test.want, got)
		})
	}
}
