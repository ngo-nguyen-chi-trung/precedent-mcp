// Test plan: resource_request.go
//
// Logic flow
// | Function                      | Step | Action          | Effect                                                     |
// |-------------------------------|------|-----------------|------------------------------------------------------------|
// | Request.ToResourceReadRequest | 1    | Copy params.uri | gRPC carries business input only; the relay checks the URI |
//
// Scenarios
// | Number | Function                      | Result  | Value class    | Input                        | Expected                              |
// |--------|-------------------------------|---------|----------------|------------------------------|---------------------------------------|
// | 1      | Request.ToResourceReadRequest | Success | Middle value   | uri skill://precedent-recall | Uri skill://precedent-recall (step 1) |
// | 2      | Request.ToResourceReadRequest | Success | Boundary value | No uri                       | Uri ""; the relay rejects it (step 1) |
//
// Failure scenarios: none. URI checks belong to the relay pipe.
//
// Branch coverage
// | Function                      | Condition | True | False |
// |-------------------------------|-----------|------|-------|
// | Request.ToResourceReadRequest | None      | —    | —     |
//
// Test mapping
// | Test                                 | Scenarios |
// |--------------------------------------|-----------|
// | TestToResourceReadRequest/uri copied | 1         |
// | TestToResourceReadRequest/no uri     | 2         |
// | No test                              | None      |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestToResourceReadRequest(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest Request
		want         *pb.ResourceReadRequest
	}{
		{"uri copied", Request{Method: env.MethodResourcesRead, ParamsBody: ParamsBody{Params: Params{URI: "skill://precedent-recall"}}},
			&pb.ResourceReadRequest{Uri: "skill://precedent-recall"}},
		{"no uri", Request{Method: env.MethodResourcesRead}, &pb.ResourceReadRequest{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenRequest.ToResourceReadRequest()

			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}
