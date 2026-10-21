// Test plan: skill.go
//
// Logic flow
// | Function | Step | Action                             | Effect                                                                                                 |
// |----------|------|------------------------------------|--------------------------------------------------------------------------------------------------------|
// | SkillURI | 1    | Parse the URI with net/url         | Parse error or no scheme → error InvalidArgument, RPCError -32602 "invalid resource URI", data {"uri"} |
// | SkillURI | 2    | Scheme is skill                    | Other scheme → error InvalidArgument, RPCError -32602 "unsupported resource scheme", data {"uri"}      |
// | SkillURI | 3    | Host holds the skill name          | Empty host → error InvalidArgument, RPCError -32602 "resource URI has no name", data {"uri"}           |
// | SkillURI | 4    | No path, not even a trailing slash | Path → error InvalidArgument, RPCError -32602 "resource URI must not have a path", data {"uri"}        |
// | SkillURI | 5    | Every part matches                 | Pass: nil result, nil error; Parsed holds the parsed URI                                               |
//
// Scenarios
// | Number | Function | Result  | Value class    | Input                                      | Expected                                                 |
// |--------|----------|---------|----------------|--------------------------------------------|----------------------------------------------------------|
// | 1      | SkillURI | Success | Middle value   | skill://sample                             | Pass; Parsed scheme skill, host sample (step 5)          |
// | 2      | SkillURI | Failure | Boundary value | skill://                                   | "resource URI has no name"; Parsed nil (step 3)          |
// | 3      | SkillURI | Failure | Boundary value | skill:sample; opaque, no host              | "resource URI has no name"; Parsed nil (step 3)          |
// | 4      | SkillURI | Failure | Boundary value | skill://sample/; trailing slash            | "resource URI must not have a path"; Parsed nil (step 4) |
// | 5      | SkillURI | Failure | Middle value   | skill://sample/SKILL.md                    | "resource URI must not have a path"; Parsed nil (step 4) |
// | 6      | SkillURI | Failure | Boundary value | examples://sample; a resource, not a skill | "unsupported resource scheme"; Parsed nil (step 2)       |
// | 7      | SkillURI | Failure | Middle value   | file://host                                | "unsupported resource scheme"; Parsed nil (step 2)       |
// | 8      | SkillURI | Failure | Boundary value | Empty URI                                  | "invalid resource URI"; Parsed nil (step 1)              |
// | 9      | SkillURI | Failure | Middle value   | skill://%zz; bad escape                    | "invalid resource URI"; Parsed nil (step 1)              |
//
// Branch coverage
// | Function | Condition                | True | False |
// |----------|--------------------------|------|-------|
// | SkillURI | Parse error or no scheme | 8, 9 | 1     |
// | SkillURI | Scheme is not skill      | 6, 7 | 1     |
// | SkillURI | Empty host               | 2, 3 | 1     |
// | SkillURI | Path not empty           | 4, 5 | 1     |
//
// Test mapping
// | Test                                   | Scenarios |
// |----------------------------------------|-----------|
// | TestSkillURI/skill URI passes          | 1         |
// | TestSkillURIRejects/skill without name | 2         |
// | TestSkillURIRejects/opaque URI         | 3         |
// | TestSkillURIRejects/trailing slash     | 4         |
// | TestSkillURIRejects/skill file path    | 5         |
// | TestSkillURIRejects/examples scheme    | 6         |
// | TestSkillURIRejects/file scheme        | 7         |
// | TestSkillURIRejects/empty URI          | 8         |
// | TestSkillURIRejects/bad escape         | 9         |
// | No test                                | None      |

package validator_test

import . "github.com/ngo/precedent_mcp/internal/service/validator"

import (
	"context"
	"net/url"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestSkillURI(t *testing.T) {
	tests := []struct {
		name       string
		givenURI   string
		want       *pb.SkillGetResponse
		wantParsed *url.URL
	}{
		{"skill URI passes", "skill://sample", nil, &url.URL{Scheme: "skill", Host: "sample"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ResourceURIRequest{URI: test.givenURI}

			got, err := SkillURI()(context.Background(), request)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantParsed, request.Parsed)
		})
	}
}

func TestSkillURIRejects(t *testing.T) {
	tests := []struct {
		name       string
		givenURI   string
		want       *pb.SkillGetResponse
		wantCode   codes.Code
		wantDetail *pb.RPCError
		wantParsed *url.URL
	}{
		{"skill without name", "skill://", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI has no name", Data: []byte(`{"uri":"skill://"}`)}, nil},
		{"opaque URI", "skill:sample", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI has no name", Data: []byte(`{"uri":"skill:sample"}`)}, nil},
		{"trailing slash", "skill://sample/", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI must not have a path", Data: []byte(`{"uri":"skill://sample/"}`)}, nil},
		{"skill file path", "skill://sample/SKILL.md", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI must not have a path", Data: []byte(`{"uri":"skill://sample/SKILL.md"}`)}, nil},
		{"examples scheme", "examples://sample", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unsupported resource scheme", Data: []byte(`{"uri":"examples://sample"}`)}, nil},
		{"file scheme", "file://host", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unsupported resource scheme", Data: []byte(`{"uri":"file://host"}`)}, nil},
		{"empty URI", "", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "invalid resource URI", Data: []byte(`{"uri":""}`)}, nil},
		{"bad escape", "skill://%zz", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "invalid resource URI", Data: []byte(`{"uri":"skill://%zz"}`)}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ResourceURIRequest{URI: test.givenURI}

			got, err := SkillURI()(context.Background(), request)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
			assert.Equal(t, test.wantParsed, request.Parsed)
		})
	}
}
