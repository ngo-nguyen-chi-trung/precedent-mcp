// Test plan: skill_service.go
//
// Logic flow
// | Function                | Step | Action                         | Effect                                                                                   |
// |-------------------------|------|--------------------------------|------------------------------------------------------------------------------------------|
// | SkillService.ListSkills | 1    | SkillStore.ListMetadata        | Skills with invalid metadata are left out; never fails                                   |
// | SkillService.ListSkills | 2    | SkillListResponse.FromMetadata | One entry per valid skill                                                                |
// | SkillService.GetSkill   | 1    | Relay pipe step SkillURI       | Invalid URI → error InvalidArgument; nothing read                                        |
// | SkillService.GetSkill   | 2    | SkillStore.Metadata(URI host)  | Unknown name → error InvalidArgument, RPCError -32602 "resource not found", data {"uri"} |
// | SkillService.GetSkill   | 3    | Metadata cannot be loaded      | That error (JSON-RPC internal error)                                                     |
// | SkillService.GetSkill   | 4    | SkillGetResponse.FromMetadata  | The skill entry                                                                          |
//
// Scenarios
// | Number | Function                | Result  | Value class  | Input                                        | Expected                                                                                                        |
// |--------|-------------------------|---------|--------------|----------------------------------------------|-----------------------------------------------------------------------------------------------------------------|
// | 1      | SkillService.ListSkills | Success | Middle value | Skills sample and broken (no description)    | Only the sample entry (steps 1, 2)                                                                              |
// | 2      | SkillService.GetSkill   | Success | Middle value | skill://sample                               | The sample entry (steps 2, 4)                                                                                   |
// | 3      | SkillService.GetSkill   | Failure | Middle value | skill://nope                                 | Nil response; error InvalidArgument; RPCError -32602 "resource not found", data {"uri":"skill://nope"} (step 2) |
// | 4      | SkillService.GetSkill   | Failure | Middle value | examples://sample                            | Nil response; error InvalidArgument; RPCError -32602 "unsupported resource scheme" (step 1)                     |
// | 5      | SkillService.GetSkill   | Failure | Middle value | skill://broken; metadata without description | Nil response; description_length error for broken, not resource not found (step 3)                              |
//
// One wiring case for the SkillURI step; URI rules are proven in validator/skill_test.go, entry encoding in model/skill_response_test.go.
//
// Branch coverage
// | Function              | Condition             | True | False |
// |-----------------------|-----------------------|------|-------|
// | SkillService.GetSkill | Pipe returns an error | 4    | 2     |
// | SkillService.GetSkill | Skill not found       | 3    | 2     |
// | SkillService.GetSkill | Other metadata error  | 5    | 2     |
//
// Test mapping
// | Test                                                       | Scenarios |
// |------------------------------------------------------------|-----------|
// | TestSkillServiceListSkills/valid skills only               | 1         |
// | TestSkillServiceGetSkill/known skill                       | 2         |
// | TestSkillServiceGetSkillRejects/unknown skill              | 3         |
// | TestSkillServiceGetSkillRejects/unsupported scheme         | 4         |
// | TestSkillServiceGetSkillError/metadata without description | 5         |
// | No test                                                    | None      |

package service_test

import . "github.com/ngo/precedent_mcp/internal/service"

import (
	"context"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/asset"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var skillTestdataSkills = map[env.SkillName]assetmodel.SkillPaths{
	"broken": {Skill: "skills/broken/SKILL.md", Metadata: "skills/broken/metadata.yaml"},
	"sample": {Skill: "skills/sample/SKILL.md", Metadata: "skills/sample/metadata.yaml"},
}

var skillSampleEntry = &pb.SkillEntry{
	Uri:         "skill://sample",
	Frontmatter: &pb.SkillFrontmatter{Name: "sample", Description: "Sample skill for tests"},
	Resources: []*pb.SkillResource{{
		Uri:    "skill://sample",
		Digest: "sha256:ceeb3cadcf67e1bfefe503cf608cd3f8b9cf473ae05e490b250ec3f32d7eb137",
		Size:   67,
	}},
}

func TestSkillServiceListSkills(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		want           *pb.SkillListResponse
	}{
		{"valid skills only", "testdata", skillTestdataSkills, &pb.SkillListResponse{Skills: []*pb.SkillEntry{skillSampleEntry}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewSkillService(asset.NewSkillStore(test.givenDirectory, test.givenSkills))

			got, err := s.ListSkills(context.Background(), &pb.Empty{})

			require.NoError(t, err)
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestSkillServiceGetSkill(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenRequest   *pb.SkillGetRequest
		want           *pb.SkillGetResponse
	}{
		{"known skill", "testdata", skillTestdataSkills, &pb.SkillGetRequest{Uri: "skill://sample"}, &pb.SkillGetResponse{Skill: skillSampleEntry}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewSkillService(asset.NewSkillStore(test.givenDirectory, test.givenSkills))

			got, err := s.GetSkill(context.Background(), test.givenRequest)

			require.NoError(t, err)
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestSkillServiceGetSkillRejects(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenRequest   *pb.SkillGetRequest
		want           *pb.SkillGetResponse
		wantCode       codes.Code
		wantDetail     *pb.RPCError
	}{
		{"unknown skill", "testdata", skillTestdataSkills, &pb.SkillGetRequest{Uri: "skill://nope"}, nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource not found", Data: []byte(`{"uri":"skill://nope"}`)}},
		{"unsupported scheme", "testdata", skillTestdataSkills, &pb.SkillGetRequest{Uri: "examples://sample"}, nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unsupported resource scheme", Data: []byte(`{"uri":"examples://sample"}`)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewSkillService(asset.NewSkillStore(test.givenDirectory, test.givenSkills))

			got, err := s.GetSkill(context.Background(), test.givenRequest)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
		})
	}
}

func TestSkillServiceGetSkillError(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenRequest   *pb.SkillGetRequest
		want           *pb.SkillGetResponse
		wantError      error
	}{
		{"metadata without description", "testdata", skillTestdataSkills, &pb.SkillGetRequest{Uri: "skill://broken"}, nil,
			assetmodel.ErrDescriptionLength("broken", 0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewSkillService(asset.NewSkillStore(test.givenDirectory, test.givenSkills))

			got, err := s.GetSkill(context.Background(), test.givenRequest)

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantError, err)
		})
	}
}
