// Test plan: tool_request.go
//
// Logic flow
// | Function                  | Step | Action                                | Effect                                                           |
// |---------------------------|------|---------------------------------------|------------------------------------------------------------------|
// | Request.ToToolCallRequest | 1    | Copy params.name and params.arguments | gRPC carries business input only; the relay checks the arguments |
//
// Scenarios
// | Number | Function                  | Result  | Value class    | Input                                      | Expected                                              |
// |--------|---------------------------|---------|----------------|--------------------------------------------|-------------------------------------------------------|
// | 1      | Request.ToToolCallRequest | Success | Middle value   | name add_context; arguments with one entry | Name add_context; the same argument bytes (step 1)    |
// | 2      | Request.ToToolCallRequest | Success | Boundary value | name retrieve_context; no arguments        | Arguments nil; the relay defaults them to {} (step 1) |
// | 3      | Request.ToToolCallRequest | Success | Middle value   | arguments [] (not an object)               | Copied unchanged; the relay rejects them (step 1)     |
//
// Failure scenarios: none. Argument checks belong to the relay pipe.
//
// Branch coverage
// | Function                  | Condition | True | False |
// |---------------------------|-----------|------|-------|
// | Request.ToToolCallRequest | None      | —    | —     |
//
// Test mapping
// | Test                                          | Scenarios |
// |-----------------------------------------------|-----------|
// | TestToToolCallRequest/arguments copied        | 1         |
// | TestToToolCallRequest/no arguments            | 2         |
// | TestToToolCallRequest/arguments not an object | 3         |
// | No test                                       | None      |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"encoding/json"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestToToolCallRequest(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest Request
		want         *pb.ToolCallRequest
	}{
		{"arguments copied",
			Request{Method: env.MethodToolsCall, ParamsBody: ParamsBody{Params: Params{
				Name:      "add_context",
				Arguments: json.RawMessage(`{"entries":[{"thread_id":"22222222-2222-2222-2222-222222222222","context":"Preference: tabs"}]}`),
			}}},
			&pb.ToolCallRequest{
				Name:      "add_context",
				Arguments: []byte(`{"entries":[{"thread_id":"22222222-2222-2222-2222-222222222222","context":"Preference: tabs"}]}`),
			}},
		{"no arguments",
			Request{Method: env.MethodToolsCall, ParamsBody: ParamsBody{Params: Params{Name: "retrieve_context"}}},
			&pb.ToolCallRequest{Name: "retrieve_context"}},
		{"arguments not an object",
			Request{Method: env.MethodToolsCall, ParamsBody: ParamsBody{Params: Params{Name: "retrieve_context", Arguments: json.RawMessage(`[]`)}}},
			&pb.ToolCallRequest{Name: "retrieve_context", Arguments: []byte(`[]`)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenRequest.ToToolCallRequest()

			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}
