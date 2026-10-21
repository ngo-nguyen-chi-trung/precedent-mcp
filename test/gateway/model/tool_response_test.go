// Test plan: tool_response.go
//
// Logic flow
// | Function                      | Step | Action                                                                | Effect                              |
// |-------------------------------|------|-----------------------------------------------------------------------|-------------------------------------|
// | Response.FromToolListResponse | 1    | Copy name, description, input and output schema as raw JSON           | Schemas pass unchanged              |
// | Response.FromToolListResponse | 2    | Complete result with cache hints                                      | tools [] when empty                 |
// | Response.FromToolCallResponse | 1    | Data as text content and as structuredContent; isError from the relay | Complete result without cache hints |
//
// Scenarios
// | Number | Function                      | Result  | Value class    | Input                                                          | Expected                                                                 |
// |--------|-------------------------------|---------|----------------|----------------------------------------------------------------|--------------------------------------------------------------------------|
// | 1      | Response.FromToolListResponse | Success | Middle value   | add_context and retrieve_context with input and output schemas | Both in order; schemas unchanged; cache hints (steps 1, 2)               |
// | 2      | Response.FromToolListResponse | Success | Boundary value | No tools                                                       | tools [] (step 2)                                                        |
// | 3      | Response.FromToolCallResponse | Success | Middle value   | Data {"results":[]}; is_error false                            | content text and structuredContent hold the data; isError false (step 1) |
// | 4      | Response.FromToolCallResponse | Failure | Middle value   | Data {"code":"decision_not_found",…}; is_error true            | isError true; the data in content and structuredContent (step 1)         |
//
// Outside the gateway: a tool without output_schema; every tool in tools.yaml has one.
//
// Branch coverage
// | Function                      | Condition   | True | False |
// |-------------------------------|-------------|------|-------|
// | Response.FromToolListResponse | Tools given | 1    | 2     |
// | Response.FromToolCallResponse | is_error    | 4    | 3     |
//
// Test mapping
// | Test                                        | Scenarios |
// |---------------------------------------------|-----------|
// | TestFromToolListResponse/tools with schemas | 1         |
// | TestFromToolListResponse/no tools           | 2         |
// | TestFromToolCallResponse/success data       | 3         |
// | TestFromToolCallResponse/is_error data      | 4         |
// | No test                                     | None      |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"encoding/json"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromToolListResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.ToolListResponse
		want          string
	}{
		{"tools with schemas",
			&pb.ToolListResponse{Tools: []*pb.Tool{
				{Name: "add_context", Description: "Save context entries.",
					InputSchema: []byte(`{"type":"object","required":["entries"]}`), OutputSchema: []byte(`{"type":"object"}`)},
				{Name: "retrieve_context", Description: "Search context entries.",
					InputSchema: []byte(`{"type":"object","required":["query"]}`), OutputSchema: []byte(`{"type":"object"}`)},
			}},
			`{` + resultBase + `,` + cacheHints + `,"tools":[` +
				`{"name":"add_context","description":"Save context entries.","inputSchema":{"type":"object","required":["entries"]},"outputSchema":{"type":"object"}},` +
				`{"name":"retrieve_context","description":"Search context entries.","inputSchema":{"type":"object","required":["query"]},"outputSchema":{"type":"object"}}]}`},
		{"no tools", &pb.ToolListResponse{}, `{` + resultBase + `,` + cacheHints + `,"tools":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromToolListResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}

func TestFromToolCallResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.ToolCallResponse
		want          string
	}{
		{"success data", &pb.ToolCallResponse{Data: []byte(`{"results":[]}`)},
			`{` + resultBase + `,"content":[{"type":"text","text":"{\"results\":[]}"}],"structuredContent":{"results":[]},"isError":false}`},
		{"is_error data",
			&pb.ToolCallResponse{
				Data:    []byte(`{"code":"decision_not_found","message":"decision not found: 33333333-3333-3333-3333-333333333333"}`),
				IsError: true,
			},
			`{` + resultBase + `,"content":[{"type":"text","text":"{\"code\":\"decision_not_found\",\"message\":\"decision not found: 33333333-3333-3333-3333-333333333333\"}"}],` +
				`"structuredContent":{"code":"decision_not_found","message":"decision not found: 33333333-3333-3333-3333-333333333333"},"isError":true}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromToolCallResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}
