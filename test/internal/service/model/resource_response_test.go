// Test plan: resource_response.go
//
// Logic flow
// | Function                        | Step | Action                                                                                    | Effect                                                               |
// |---------------------------------|------|-------------------------------------------------------------------------------------------|----------------------------------------------------------------------|
// | ResourceListResponse.FromAssets | 1    | Per skill metadata: skill://<name> with name, description, text/markdown, size            | The skill text is a resource                                         |
// | ResourceListResponse.FromAssets | 2    | Per examples name: examples://<name> with name, description, text/markdown, audience user | Examples are meant for the user; DescriptionExamples names the skill |
// | ResourceListResponse.FromAssets | 3    | Skill resources first, then examples resources                                            | Empty input gives an empty, non-nil list                             |
// | ResourceReadResponse.FromText   | 1    | One content: the requested URI, text/markdown, the text                                   | The response echoes the URI the client asked for                     |
//
// Scenarios
// | Number | Function                        | Result  | Value class    | Input                                    | Expected                                                             |
// |--------|---------------------------------|---------|----------------|------------------------------------------|----------------------------------------------------------------------|
// | 1      | ResourceListResponse.FromAssets | Success | Middle value   | Skills sample and other; examples sample | skill://sample, skill://other, then examples://sample (steps 1 to 3) |
// | 2      | ResourceListResponse.FromAssets | Success | Boundary value | No skills; no examples                   | Non-nil response; resources empty (step 3)                           |
// | 3      | ResourceReadResponse.FromText   | Success | Middle value   | skill://sample and its text              | One content with the URI, text/markdown, the text (step 1)           |
//
// Branch coverage
// | Function                        | Condition      | True | False |
// |---------------------------------|----------------|------|-------|
// | ResourceListResponse.FromAssets | Skills given   | 1    | 2     |
// | ResourceListResponse.FromAssets | Examples given | 1    | 2     |
//
// Test mapping
// | Test                                          | Scenarios                                                                                                                                                                                            |
// |-----------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | TestResourceListResponse/skills then examples | 1                                                                                                                                                                                                    |
// | TestResourceListResponse/no resources         | 2                                                                                                                                                                                                    |
// | TestResourceReadResponse/one markdown content | 3                                                                                                                                                                                                    |
// | No test                                       | Resource URI error constructors: proven by validator/resource_test.go, validator/skill_test.go, resource_service_test.go and skill_service_test.go; their encoding errors cannot happen for a string |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestResourceListResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenMetadata []assetmodel.Metadata
		givenNames    []env.SkillName
		want          *pb.ResourceListResponse
	}{
		{"skills then examples", []assetmodel.Metadata{skillSampleMetadata, skillOtherMetadata}, []env.SkillName{"sample"},
			&pb.ResourceListResponse{Resources: []*pb.Resource{
				{Uri: "skill://sample", Name: "sample", Description: "Sample skill for tests", MimeType: "text/markdown", Size: 67},
				{Uri: "skill://other", Name: "other", Description: "Other skill for tests", MimeType: "text/markdown", Size: 120},
				{Uri: "examples://sample", Name: "sample", Description: "Examples for the sample skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.", MimeType: "text/markdown",
					Annotations: &pb.Annotations{Audience: []string{"user"}}},
			}}},
		{"no resources", nil, nil, &pb.ResourceListResponse{Resources: []*pb.Resource{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got ResourceListResponse

			got.FromAssets(test.givenMetadata, test.givenNames)

			require.NotNil(t, got.ResourceListResponse)
			assert.True(t, proto.Equal(test.want, got.ResourceListResponse), "got %v", got.ResourceListResponse)
		})
	}
}

func TestResourceReadResponse(t *testing.T) {
	tests := []struct {
		name      string
		givenURI  string
		givenText string
		want      *pb.ResourceReadResponse
	}{
		{"one markdown content", "skill://sample", "# Sample\n",
			&pb.ResourceReadResponse{Contents: []*pb.ResourceContents{{Uri: "skill://sample", MimeType: "text/markdown", Text: "# Sample\n"}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got ResourceReadResponse

			got.FromText(test.givenURI, test.givenText)

			assert.True(t, proto.Equal(test.want, got.ResourceReadResponse), "got %v", got.ResourceReadResponse)
		})
	}
}
