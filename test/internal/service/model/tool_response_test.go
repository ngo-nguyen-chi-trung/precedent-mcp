// Test plan: tool_response.go
//
// Logic flow
// | Function                         | Step | Action                                               | Effect                                             |
// |----------------------------------|------|------------------------------------------------------|----------------------------------------------------|
// | ToolListResponse.FromToolEntries | 1    | Encode each input schema as JSON                     | MCP inputSchema                                    |
// | ToolListResponse.FromToolEntries | 2    | Encode the output schema only when the entry has one | outputSchema is optional in MCP                    |
// | ToolListResponse.FromToolEntries | 3    | Collect the tools in entry order                     | Encoding failure → error (JSON-RPC internal error) |
//
// Scenarios
// | Number | Function                         | Result  | Value class    | Input                                           | Expected                                                                 |
// |--------|----------------------------------|---------|----------------|-------------------------------------------------|--------------------------------------------------------------------------|
// | 1      | ToolListResponse.FromToolEntries | Success | Middle value   | echo with output schema; ping without           | Both in order; schemas as JSON; ping has no output schema (steps 1 to 3) |
// | 2      | ToolListResponse.FromToolEntries | Success | Boundary value | No entries                                      | Non-nil response; tools empty (step 3)                                   |
// | 3      | ToolListResponse.FromToolEntries | Failure | Middle value   | Output schema property with an integer YAML key | error "json: unsupported type: map[interface {}]interface {}" (step 2)   |
//
// Branch coverage
// | Function                         | Condition           | True     | False    |
// |----------------------------------|---------------------|----------|----------|
// | ToolListResponse.FromToolEntries | Output schema given | 1 (echo) | 1 (ping) |
// | ToolListResponse.FromToolEntries | Encoding error      | 3        | 1        |
// | ToolListResponse.FromToolEntries | Entries given       | 1        | 2        |
//
// Test mapping
// | Test                                                  | Scenarios                                                                                                                                                                                       |
// |-------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | TestToolListResponse/schemas in entry order           | 1                                                                                                                                                                                               |
// | TestToolListResponse/no entries                       | 2                                                                                                                                                                                               |
// | TestToolListResponseError/output schema not encodable | 3                                                                                                                                                                                               |
// | No test                                               | Input schema encoding error: validator.InputSchemas encodes it first; UnknownToolError, InvalidArgsJSONError, InvalidArgsSchemaError: proven by validator/tool_test.go and tool_service_test.go |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestToolListResponse(t *testing.T) {
	tests := []struct {
		name         string
		givenEntries []assetmodel.ToolEntry
		want         *pb.ToolListResponse
	}{
		{
			name: "schemas in entry order",
			givenEntries: []assetmodel.ToolEntry{
				{
					Name:        "echo",
					Description: "Echo the input back.",
					InputSchema: assetmodel.ToolSchema{
						Type:       "object",
						Properties: map[string]any{"text": map[string]any{"type": "string", "minLength": 1}},
						Required:   []string{"text"},
					},
					OutputSchema: &assetmodel.OutputSchema{Type: "object", AnyOf: []assetmodel.ToolSchema{
						{Type: "object", Properties: map[string]any{"text": map[string]any{"type": "string"}}, Required: []string{"text"}},
					}},
				},
				{Name: "ping", Description: "No arguments.", InputSchema: assetmodel.ToolSchema{Type: "object", Properties: map[string]any{}}},
			},
			want: &pb.ToolListResponse{Tools: []*pb.Tool{
				{
					Name:         "echo",
					Description:  "Echo the input back.",
					InputSchema:  []byte(`{"type":"object","properties":{"text":{"minLength":1,"type":"string"}},"required":["text"]}`),
					OutputSchema: []byte(`{"type":"object","anyOf":[{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}]}`),
				},
				{Name: "ping", Description: "No arguments.", InputSchema: []byte(`{"type":"object","properties":{}}`)},
			}},
		},
		{
			name:         "no entries",
			givenEntries: nil,
			want:         &pb.ToolListResponse{Tools: []*pb.Tool{}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got ToolListResponse

			err := got.FromToolEntries(test.givenEntries)

			require.NoError(t, err)
			require.NotNil(t, got.ToolListResponse)
			assert.True(t, proto.Equal(test.want, got.ToolListResponse), "got %v", got.ToolListResponse)
		})
	}
}

func TestToolListResponseError(t *testing.T) {
	tests := []struct {
		name         string
		givenEntries []assetmodel.ToolEntry
		want         *pb.ToolListResponse
		wantError    string
	}{
		{
			name: "output schema not encodable",
			givenEntries: []assetmodel.ToolEntry{{
				Name:        "echo",
				InputSchema: assetmodel.ToolSchema{Type: "object", Properties: map[string]any{}},
				OutputSchema: &assetmodel.OutputSchema{Type: "object", AnyOf: []assetmodel.ToolSchema{
					{Type: "object", Properties: map[string]any{"text": map[any]any{1: "string"}}},
				}},
			}},
			want:      nil,
			wantError: "json: unsupported type: map[interface {}]interface {}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got ToolListResponse

			err := got.FromToolEntries(test.givenEntries)

			assert.EqualError(t, err, test.wantError)
			assert.Equal(t, test.want, got.ToolListResponse)
		})
	}
}
