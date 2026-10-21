// Test plan: capability_response.go
//
// Logic flow
// | Function                      | Step | Action                                                                          | Effect                                                                |
// |-------------------------------|------|---------------------------------------------------------------------------------|-----------------------------------------------------------------------|
// | Response.FromDiscoverResponse | 1    | One empty object per extension                                                  | Extension names become capability keys                                |
// | Response.FromDiscoverResponse | 2    | Complete result with server information, cache hints, versions and capabilities | Discover result is cacheable (Go SDK DiscoverResult embeds Cacheable) |
//
// Scenarios
// | Number | Function                      | Result  | Value class  | Input                                                        | Expected                                                                                                                                                           |
// |--------|-------------------------------|---------|--------------|--------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | 1      | Response.FromDiscoverResponse | Success | Middle value | Version 2026-07-28; extension io.modelcontextprotocol/skills | resultType complete; serverInfo precedent-mcp 1.0.0; ttlMs 300000; cacheScope public; supportedVersions; tools {}, resources {}, extensions {skills: {}} (steps 1, 2) |
//
// Failure scenarios: none. Discover failures arrive as gRPC errors and go through FromStatus.
//
// Branch coverage
// | Function                      | Condition        | True | False |
// |-------------------------------|------------------|------|-------|
// | Response.FromDiscoverResponse | Extensions given | 1    | —     |
//
// Test mapping
// | Test                                                  | Scenarios                                                  |
// |-------------------------------------------------------|------------------------------------------------------------|
// | TestFromDiscoverResponse/version and skills extension | 1                                                          |
// | No test                                               | No extensions: the relay always sends the skills extension |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"encoding/json"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	resultBase = `"resultType":"complete","_meta":{"io.modelcontextprotocol/serverInfo":{"name":"precedent-mcp","version":"1.0.0"}}`
	cacheHints = `"ttlMs":300000,"cacheScope":"public"`
)

func TestFromDiscoverResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.DiscoverResponse
		want          string
	}{
		{"version and skills extension",
			&pb.DiscoverResponse{SupportedVersions: []string{"2026-07-28"}, Extensions: []string{"io.modelcontextprotocol/skills"}},
			`{` + resultBase + `,` + cacheHints + `,"supportedVersions":["2026-07-28"],` +
				`"capabilities":{"tools":{},"resources":{},"extensions":{"io.modelcontextprotocol/skills":{}}}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromDiscoverResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}
