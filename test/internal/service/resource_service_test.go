// Test plan: resource_service.go
//
// Logic flow
// | Function                      | Step | Action                                                                   | Effect                                                                    |
// |-------------------------------|------|--------------------------------------------------------------------------|---------------------------------------------------------------------------|
// | ResourceService.ListResources | 1    | SkillStore.ListMetadata and ExampleStore.ListNames                       | Skills with invalid metadata are left out; never fails                    |
// | ResourceService.ListResources | 2    | ResourceListResponse.FromAssets                                          | Skill resources, then examples resources                                  |
// | ResourceService.ReadResource  | 1    | Relay pipe step ResourceURI                                              | Invalid URI → error InvalidArgument; nothing read                         |
// | ResourceService.ReadResource  | 2    | examples scheme → ExampleStore.Examples, skill scheme → SkillStore.Skill | The URI host is the name                                                  |
// | ResourceService.ReadResource  | 3    | Unknown name                                                             | error InvalidArgument, RPCError -32602 "resource not found", data {"uri"} |
// | ResourceService.ReadResource  | 4    | File cannot be read                                                      | That error (JSON-RPC internal error)                                      |
// | ResourceService.ReadResource  | 5    | ResourceReadResponse.FromText                                            | One text/markdown content with the requested URI                          |
//
// Scenarios
// | Number | Function                      | Result  | Value class  | Input                                                        | Expected                                                                                                    |
// |--------|-------------------------------|---------|--------------|--------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------|
// | 1      | ResourceService.ListResources | Success | Middle value | Skills sample and broken (invalid metadata); examples sample | skill://sample with description and size, then examples://sample for the user; broken left out (steps 1, 2) |
// | 2      | ResourceService.ReadResource  | Success | Middle value | skill://sample                                               | SKILL.md text with the URI and text/markdown (steps 2, 5)                                                   |
// | 3      | ResourceService.ReadResource  | Success | Middle value | examples://sample                                            | examples.md text with the URI and text/markdown (steps 2, 5)                                                |
// | 4      | ResourceService.ReadResource  | Failure | Middle value | skill://nope                                                 | error InvalidArgument; RPCError -32602 "resource not found", data {"uri":"skill://nope"} (step 3)           |
// | 5      | ResourceService.ReadResource  | Failure | Middle value | examples://nope                                              | error InvalidArgument; RPCError -32602 "resource not found", data {"uri":"examples://nope"} (step 3)        |
// | 6      | ResourceService.ReadResource  | Failure | Middle value | file://host                                                  | error InvalidArgument; RPCError -32602 "unsupported resource scheme" (step 1)                               |
// | 7      | ResourceService.ReadResource  | Failure | Middle value | skill://missing; SKILL.md missing                            | Nil response; error matching fs.ErrNotExist (step 4)                                                        |
//
// One wiring case for the ResourceURI step; URI rules are proven in validator/resource_test.go, resource encoding in model/resource_response_test.go.
//
// Branch coverage
// | Function                     | Condition             | True | False   |
// |------------------------------|-----------------------|------|---------|
// | ResourceService.ReadResource | Pipe returns an error | 6    | 2, 3    |
// | ResourceService.ReadResource | Scheme is examples    | 3, 5 | 2, 4, 7 |
// | ResourceService.ReadResource | Name not found        | 4, 5 | 2, 3    |
// | ResourceService.ReadResource | Other read error      | 7    | 2, 3    |
//
// Test mapping
// | Test                                                       | Scenarios |
// |------------------------------------------------------------|-----------|
// | TestResourceServiceListResources/valid skills and examples | 1         |
// | TestResourceServiceReadResource/skill text                 | 2         |
// | TestResourceServiceReadResource/examples text              | 3         |
// | TestResourceServiceReadResourceRejects/unknown skill       | 4         |
// | TestResourceServiceReadResourceRejects/unknown examples    | 5         |
// | TestResourceServiceReadResourceRejects/unsupported scheme  | 6         |
// | TestResourceServiceReadResourceError/SKILL.md missing      | 7         |
// | No test                                                    | None      |

package service_test

import . "github.com/ngo/precedent_mcp/internal/service"

import (
	"context"
	"io/fs"
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

var resourceTestdataExamples = map[env.SkillName]env.AssetPath{
	"sample": "skills/sample/examples.md",
}

func TestResourceServiceListResources(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenExamples  map[env.SkillName]env.AssetPath
		want           *pb.ResourceListResponse
	}{
		{"valid skills and examples", "testdata", skillTestdataSkills, resourceTestdataExamples,
			&pb.ResourceListResponse{Resources: []*pb.Resource{
				{Uri: "skill://sample", Name: "sample", Description: "Sample skill for tests", MimeType: "text/markdown", Size: 67},
				{Uri: "examples://sample", Name: "sample", Description: "Examples for the sample skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.", MimeType: "text/markdown",
					Annotations: &pb.Annotations{Audience: []string{"user"}}},
			}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewResourceService(asset.NewSkillStore(test.givenDirectory, test.givenSkills), asset.NewExampleStore(test.givenDirectory, test.givenExamples))

			got, err := s.ListResources(context.Background(), &pb.Empty{})

			require.NoError(t, err)
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestResourceServiceReadResource(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenExamples  map[env.SkillName]env.AssetPath
		givenRequest   *pb.ResourceReadRequest
		want           *pb.ResourceReadResponse
	}{
		{"skill text", "testdata", skillTestdataSkills, resourceTestdataExamples, &pb.ResourceReadRequest{Uri: "skill://sample"},
			&pb.ResourceReadResponse{Contents: []*pb.ResourceContents{{Uri: "skill://sample", MimeType: "text/markdown",
				Text: "---\nname: sample\ndescription: Sample skill for tests\n---\n\n# Sample\n"}}}},
		{"examples text", "testdata", skillTestdataSkills, resourceTestdataExamples, &pb.ResourceReadRequest{Uri: "examples://sample"},
			&pb.ResourceReadResponse{Contents: []*pb.ResourceContents{{Uri: "examples://sample", MimeType: "text/markdown",
				Text: "# Examples for sample\n"}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewResourceService(asset.NewSkillStore(test.givenDirectory, test.givenSkills), asset.NewExampleStore(test.givenDirectory, test.givenExamples))

			got, err := s.ReadResource(context.Background(), test.givenRequest)

			require.NoError(t, err)
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestResourceServiceReadResourceRejects(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenExamples  map[env.SkillName]env.AssetPath
		givenRequest   *pb.ResourceReadRequest
		want           *pb.ResourceReadResponse
		wantCode       codes.Code
		wantDetail     *pb.RPCError
	}{
		{"unknown skill", "testdata", skillTestdataSkills, resourceTestdataExamples, &pb.ResourceReadRequest{Uri: "skill://nope"}, nil,
			codes.InvalidArgument, &pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource not found", Data: []byte(`{"uri":"skill://nope"}`)}},
		{"unknown examples", "testdata", skillTestdataSkills, resourceTestdataExamples, &pb.ResourceReadRequest{Uri: "examples://nope"}, nil,
			codes.InvalidArgument, &pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource not found", Data: []byte(`{"uri":"examples://nope"}`)}},
		{"unsupported scheme", "testdata", skillTestdataSkills, resourceTestdataExamples, &pb.ResourceReadRequest{Uri: "file://host"}, nil,
			codes.InvalidArgument, &pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unsupported resource scheme", Data: []byte(`{"uri":"file://host"}`)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewResourceService(asset.NewSkillStore(test.givenDirectory, test.givenSkills), asset.NewExampleStore(test.givenDirectory, test.givenExamples))

			got, err := s.ReadResource(context.Background(), test.givenRequest)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
		})
	}
}

func TestResourceServiceReadResourceError(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]assetmodel.SkillPaths
		givenExamples  map[env.SkillName]env.AssetPath
		givenRequest   *pb.ResourceReadRequest
		want           *pb.ResourceReadResponse
		wantError      error
	}{
		{"SKILL.md missing", "testdata",
			map[env.SkillName]assetmodel.SkillPaths{"missing": {Skill: "skills/missing/SKILL.md", Metadata: "skills/missing/metadata.yaml"}},
			resourceTestdataExamples, &pb.ResourceReadRequest{Uri: "skill://missing"}, nil, fs.ErrNotExist},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewResourceService(asset.NewSkillStore(test.givenDirectory, test.givenSkills), asset.NewExampleStore(test.givenDirectory, test.givenExamples))

			got, err := s.ReadResource(context.Background(), test.givenRequest)

			assert.Equal(t, test.want, got)
			assert.ErrorIs(t, err, test.wantError)
		})
	}
}
