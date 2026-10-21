// Test plan: skill_response.go
//
// Logic flow
// | Function                       | Step | Action                                                            | Effect                                                                |
// |--------------------------------|------|-------------------------------------------------------------------|-----------------------------------------------------------------------|
// | SkillEntry.FromMetadata        | 1    | URI = skill://<name>, no path                                     | One URI names the skill in skills/list, skills/get and resources/read |
// | SkillEntry.FromMetadata        | 2    | Frontmatter = name, description; one resource = URI, digest, size | The entry carries the metadata.yaml values                            |
// | SkillListResponse.FromMetadata | 1    | One entry per metadata, in order                                  | Empty input gives an empty, non-nil list                              |
// | SkillGetResponse.FromMetadata  | 1    | The entry of the one metadata                                     | Same entry shape as skills/list                                       |
//
// Scenarios
// | Number | Function                       | Result  | Value class    | Input                     | Expected                                                                             |
// |--------|--------------------------------|---------|----------------|---------------------------|--------------------------------------------------------------------------------------|
// | 1      | SkillEntry.FromMetadata        | Success | Middle value   | sample metadata           | skill://sample; frontmatter sample; one resource with URI, digest, size (steps 1, 2) |
// | 2      | SkillListResponse.FromMetadata | Success | Middle value   | Metadata sample and other | Two entries in order (step 1)                                                        |
// | 3      | SkillListResponse.FromMetadata | Success | Boundary value | No metadata               | Non-nil response; skills empty (step 1)                                              |
// | 4      | SkillGetResponse.FromMetadata  | Success | Middle value   | sample metadata           | The sample entry (step 1)                                                            |
//
// Failure scenarios: none. These conversions have no error path.
//
// Branch coverage
// | Function                       | Condition      | True | False |
// |--------------------------------|----------------|------|-------|
// | SkillListResponse.FromMetadata | Metadata given | 2    | 3     |
//
// Test mapping
// | Test                                          | Scenarios |
// |-----------------------------------------------|-----------|
// | TestSkillEntryFromMetadata/one skill resource | 1         |
// | TestSkillListResponse/entries in order        | 2         |
// | TestSkillListResponse/no skills               | 3         |
// | TestSkillGetResponse/one skill entry          | 4         |
// | No test                                       | None      |

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

var (
	skillSampleMetadata = assetmodel.Metadata{
		Name:        "sample",
		Description: "Sample skill for tests",
		Digest:      "sha256:ceeb3cadcf67e1bfefe503cf608cd3f8b9cf473ae05e490b250ec3f32d7eb137",
		Size:        67,
	}
	skillOtherMetadata = assetmodel.Metadata{
		Name:        "other",
		Description: "Other skill for tests",
		Digest:      "sha256:1f2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3",
		Size:        120,
	}
	skillSampleEntry = &pb.SkillEntry{
		Uri:         "skill://sample",
		Frontmatter: &pb.SkillFrontmatter{Name: "sample", Description: "Sample skill for tests"},
		Resources: []*pb.SkillResource{{
			Uri:    "skill://sample",
			Digest: "sha256:ceeb3cadcf67e1bfefe503cf608cd3f8b9cf473ae05e490b250ec3f32d7eb137",
			Size:   67,
		}},
	}
	skillOtherEntry = &pb.SkillEntry{
		Uri:         "skill://other",
		Frontmatter: &pb.SkillFrontmatter{Name: "other", Description: "Other skill for tests"},
		Resources: []*pb.SkillResource{{
			Uri:    "skill://other",
			Digest: "sha256:1f2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3",
			Size:   120,
		}},
	}
)

func TestSkillEntryFromMetadata(t *testing.T) {
	tests := []struct {
		name          string
		givenMetadata assetmodel.Metadata
		want          *pb.SkillEntry
	}{
		{"one skill resource", skillSampleMetadata, skillSampleEntry},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got SkillEntry

			got.FromMetadata(test.givenMetadata)

			assert.True(t, proto.Equal(test.want, got.SkillEntry), "got %v", got.SkillEntry)
		})
	}
}

func TestSkillListResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenMetadata []assetmodel.Metadata
		want          *pb.SkillListResponse
	}{
		{"entries in order", []assetmodel.Metadata{skillSampleMetadata, skillOtherMetadata},
			&pb.SkillListResponse{Skills: []*pb.SkillEntry{skillSampleEntry, skillOtherEntry}}},
		{"no skills", nil, &pb.SkillListResponse{Skills: []*pb.SkillEntry{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got SkillListResponse

			got.FromMetadata(test.givenMetadata)

			require.NotNil(t, got.SkillListResponse)
			assert.True(t, proto.Equal(test.want, got.SkillListResponse), "got %v", got.SkillListResponse)
		})
	}
}

func TestSkillGetResponse(t *testing.T) {
	tests := []struct {
		name          string
		givenMetadata assetmodel.Metadata
		want          *pb.SkillGetResponse
	}{
		{"one skill entry", skillSampleMetadata, &pb.SkillGetResponse{Skill: skillSampleEntry}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got SkillGetResponse

			got.FromMetadata(test.givenMetadata)

			assert.True(t, proto.Equal(test.want, got.SkillGetResponse), "got %v", got.SkillGetResponse)
		})
	}
}
