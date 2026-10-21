// Test plan: resource_response.go
//
// Logic flow
// | Function                          | Step | Action                                                                 | Effect                                                                |
// |-----------------------------------|------|------------------------------------------------------------------------|-----------------------------------------------------------------------|
// | Response.FromResourceListResponse | 1    | Copy uri, name, description, mimeType, size; annotations only when set | Size 0 and no annotations are omitted                                 |
// | Response.FromResourceListResponse | 2    | Complete result with cache hints                                       | resources [] when empty                                               |
// | Response.FromResourceReadResponse | 1    | Copy uri, mimeType, text of each content                               | Read result is cacheable (Go SDK ReadResourceResult embeds Cacheable) |
//
// Scenarios
// | Number | Function                          | Result  | Value class    | Input                                                                                              | Expected                                                                                          |
// |--------|-----------------------------------|---------|----------------|----------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------|
// | 1      | Response.FromResourceListResponse | Success | Middle value   | skill://precedent-recall with description and size; examples://precedent-recall with audience user | Both in order; size only on the skill; annotations only on the examples; cache hints (steps 1, 2) |
// | 2      | Response.FromResourceListResponse | Success | Boundary value | No resources                                                                                       | resources [] (step 2)                                                                             |
// | 3      | Response.FromResourceReadResponse | Success | Middle value   | One content of skill://precedent-recall                                                            | contents with uri, mimeType text/markdown, text; cache hints (step 1)                             |
//
// Failure scenarios: none. Unknown resources arrive as gRPC errors and go through FromStatus.
//
// Branch coverage
// | Function                          | Condition         | True         | False     |
// |-----------------------------------|-------------------|--------------|-----------|
// | Response.FromResourceListResponse | Annotations given | 1 (examples) | 1 (skill) |
// | Response.FromResourceListResponse | Resources given   | 1            | 2         |
//
// Test mapping
// | Test                                                      | Scenarios |
// |-----------------------------------------------------------|-----------|
// | TestFromResourceListResponse/skill and examples resources | 1         |
// | TestFromResourceListResponse/no resources                 | 2         |
// | TestFromResourceReadResponse/skill text                   | 3         |
// | No test                                                   | None      |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"encoding/json"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromResourceListResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.ResourceListResponse
		want          string
	}{
		{"skill and examples resources",
			&pb.ResourceListResponse{Resources: []*pb.Resource{
				{Uri: "skill://precedent-recall", Name: "precedent-recall", Description: "Rebuild context", MimeType: "text/markdown", Size: 42},
				{Uri: "examples://precedent-recall", Name: "precedent-recall", Description: "Examples for the precedent-recall skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.", MimeType: "text/markdown",
					Annotations: &pb.Annotations{Audience: []string{"user"}}},
			}},
			`{` + resultBase + `,` + cacheHints + `,"resources":[` +
				`{"uri":"skill://precedent-recall","name":"precedent-recall","description":"Rebuild context","mimeType":"text/markdown","size":42},` +
				`{"uri":"examples://precedent-recall","name":"precedent-recall","description":"Examples for the precedent-recall skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.",` +
				`"mimeType":"text/markdown","annotations":{"audience":["user"]}}]}`},
		{"no resources", &pb.ResourceListResponse{}, `{` + resultBase + `,` + cacheHints + `,"resources":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromResourceListResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}

func TestFromResourceReadResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.ResourceReadResponse
		want          string
	}{
		{"skill text",
			&pb.ResourceReadResponse{Contents: []*pb.ResourceContents{{Uri: "skill://precedent-recall", MimeType: "text/markdown", Text: "# Recall\n"}}},
			`{` + resultBase + `,` + cacheHints + `,"contents":[{"uri":"skill://precedent-recall","mimeType":"text/markdown","text":"# Recall\n"}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromResourceReadResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}
