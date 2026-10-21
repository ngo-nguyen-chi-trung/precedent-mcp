// Test plan: skill_response.go
//
// Logic flow
// | Function                       | Step | Action                                                           | Effect                      |
// |--------------------------------|------|------------------------------------------------------------------|-----------------------------|
// | Response.FromSkillListResponse | 1    | SkillEntry.FromSkillEntry per skill: uri, frontmatter, resources | skills [] when empty        |
// | Response.FromSkillListResponse | 2    | Complete result with cache hints                                 | The skill list is cacheable |
// | Response.FromSkillGetResponse  | 1    | SkillEntry.FromSkillEntry; complete result with cache hints      | The skill is cacheable      |
//
// Scenarios
// | Number | Function                       | Result  | Value class    | Input                                                         | Expected                                                          |
// |--------|--------------------------------|---------|----------------|---------------------------------------------------------------|-------------------------------------------------------------------|
// | 1      | Response.FromSkillListResponse | Success | Middle value   | skill://precedent-recall with name, description, digest, size | skills with uri, frontmatter, resources; cache hints (steps 1, 2) |
// | 2      | Response.FromSkillListResponse | Success | Boundary value | No skills                                                     | skills [] (step 1)                                                |
// | 3      | Response.FromSkillGetResponse  | Success | Middle value   | skill://precedent-recall                                      | skill with uri, frontmatter, resources; cache hints (step 1)      |
//
// Failure scenarios: none. Unknown skills arrive as gRPC errors and go through FromStatus.
//
// Cache hints source: MCP skills extension (ext-skills specification/stable/skills.mdx): ListSkillsResult and GetSkillResult extend CacheableResult, so ttlMs and cacheScope are REQUIRED.
//
// Branch coverage
// | Function                       | Condition    | True | False |
// |--------------------------------|--------------|------|-------|
// | Response.FromSkillListResponse | Skills given | 1    | 2     |
//
// Test mapping
// | Test                                | Scenarios                                                                                                                  |
// |-------------------------------------|----------------------------------------------------------------------------------------------------------------------------|
// | TestFromSkillListResponse/one skill | 1                                                                                                                          |
// | TestFromSkillListResponse/no skills | 2                                                                                                                          |
// | TestFromSkillGetResponse/one skill  | 3                                                                                                                          |
// | No test                             | SkillEntry.FromSkillEntry: proven by scenarios 1, 3; entries without resources or frontmatter: the relay always sends both |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"encoding/json"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const recallEntry = `{"uri":"skill://precedent-recall","frontmatter":{"name":"precedent-recall","description":"Rebuild context"},` +
	`"resources":[{"uri":"skill://precedent-recall","digest":"sha256:9b74c9897bac770ffc029102a200c5de9b74c9897bac770ffc029102a200c5de","size":42}]}`

var recall = &pb.SkillEntry{
	Uri:         "skill://precedent-recall",
	Frontmatter: &pb.SkillFrontmatter{Name: "precedent-recall", Description: "Rebuild context"},
	Resources: []*pb.SkillResource{{
		Uri:    "skill://precedent-recall",
		Digest: "sha256:9b74c9897bac770ffc029102a200c5de9b74c9897bac770ffc029102a200c5de",
		Size:   42,
	}},
}

func TestFromSkillListResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.SkillListResponse
		want          string
	}{
		{"one skill", &pb.SkillListResponse{Skills: []*pb.SkillEntry{recall}}, `{` + resultBase + `,` + cacheHints + `,"skills":[` + recallEntry + `]}`},
		{"no skills", &pb.SkillListResponse{}, `{` + resultBase + `,` + cacheHints + `,"skills":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromSkillListResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}

func TestFromSkillGetResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenResponse *pb.SkillGetResponse
		want          string
	}{
		{"one skill", &pb.SkillGetResponse{Skill: recall}, `{` + resultBase + `,` + cacheHints + `,"skill":` + recallEntry + `}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.FromSkillGetResponse(test.givenResponse)

			got, err := json.Marshal(res.Result)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}
