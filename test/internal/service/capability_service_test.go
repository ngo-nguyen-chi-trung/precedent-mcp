// Test plan: capability_service.go
//
// Logic flow
// | Function                   | Step | Action                                                | Effect                                                    |
// |----------------------------|------|-------------------------------------------------------|-----------------------------------------------------------|
// | CapabilityService.Discover | 1    | Return the supported protocol versions and extensions | Clients learn version 2026-07-28 and the skills extension |
//
// Scenarios
// | Number | Function                   | Result  | Value class  | Input         | Expected                                                                              |
// |--------|----------------------------|---------|--------------|---------------|---------------------------------------------------------------------------------------|
// | 1      | CapabilityService.Discover | Success | Middle value | Empty request | Supported versions [2026-07-28]; extensions [io.modelcontextprotocol/skills] (step 1) |
//
// Failure scenarios: none. Discover reads only constants; protocol version checks run in the gateway pipe.
//
// Branch coverage
// | Function                   | Condition | True | False |
// |----------------------------|-----------|------|-------|
// | CapabilityService.Discover | None      | —    | —     |
//
// Test mapping
// | Test                                                       | Scenarios |
// |------------------------------------------------------------|-----------|
// | TestCapabilityServiceDiscover/version and skills extension | 1         |
// | No test                                                    | None      |

package service_test

import . "github.com/ngo/precedent_mcp/internal/service"

import (
	"context"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestCapabilityServiceDiscover(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest *pb.Empty
		want         *pb.DiscoverResponse
	}{
		{"version and skills extension", &pb.Empty{},
			&pb.DiscoverResponse{SupportedVersions: []string{"2026-07-28"}, Extensions: []string{"io.modelcontextprotocol/skills"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewCapabilityService().Discover(context.Background(), test.givenRequest)

			require.NoError(t, err)
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}
