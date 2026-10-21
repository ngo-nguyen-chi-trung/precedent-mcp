// Test plan: resource.go
//
// Logic flow
// | Function    | Step | Action                             | Effect                                                                                                 |
// |-------------|------|------------------------------------|--------------------------------------------------------------------------------------------------------|
// | ResourceURI | 1    | Parse the URI with net/url         | Parse error or no scheme → error InvalidArgument, RPCError -32602 "invalid resource URI", data {"uri"} |
// | ResourceURI | 2    | Scheme is skill or examples        | Other scheme → error InvalidArgument, RPCError -32602 "unsupported resource scheme", data {"uri"}      |
// | ResourceURI | 3    | Host holds the resource name       | Empty host → error InvalidArgument, RPCError -32602 "resource URI has no name", data {"uri"}           |
// | ResourceURI | 4    | No path, not even a trailing slash | Path → error InvalidArgument, RPCError -32602 "resource URI must not have a path", data {"uri"}        |
// | ResourceURI | 5    | Every part matches                 | Pass: nil result, nil error; Parsed holds the parsed URI                                               |
//
// Scenarios
// | Number | Function    | Result  | Value class    | Input                           | Expected                                                 |
// |--------|-------------|---------|----------------|---------------------------------|----------------------------------------------------------|
// | 1      | ResourceURI | Success | Middle value   | skill://sample                  | Pass; Parsed scheme skill, host sample (step 5)          |
// | 2      | ResourceURI | Success | Middle value   | examples://sample               | Pass; Parsed scheme examples, host sample (step 5)       |
// | 3      | ResourceURI | Failure | Boundary value | skill://                        | "resource URI has no name"; Parsed nil (step 3)          |
// | 4      | ResourceURI | Failure | Boundary value | examples://                     | "resource URI has no name"; Parsed nil (step 3)          |
// | 5      | ResourceURI | Failure | Boundary value | skill:sample; opaque, no host   | "resource URI has no name"; Parsed nil (step 3)          |
// | 6      | ResourceURI | Failure | Boundary value | skill://sample/; trailing slash | "resource URI must not have a path"; Parsed nil (step 4) |
// | 7      | ResourceURI | Failure | Middle value   | skill://sample/SKILL.md         | "resource URI must not have a path"; Parsed nil (step 4) |
// | 8      | ResourceURI | Failure | Middle value   | examples://sample/examples.md   | "resource URI must not have a path"; Parsed nil (step 4) |
// | 9      | ResourceURI | Failure | Middle value   | file://host                     | "unsupported resource scheme"; Parsed nil (step 2)       |
// | 10     | ResourceURI | Failure | Boundary value | Empty URI                       | "invalid resource URI"; Parsed nil (step 1)              |
// | 11     | ResourceURI | Failure | Middle value   | skill://%zz; bad escape         | "invalid resource URI"; Parsed nil (step 1)              |
//
// Branch coverage
// | Function    | Condition                         | True    | False |
// |-------------|-----------------------------------|---------|-------|
// | ResourceURI | Parse error or no scheme          | 10, 11  | 1, 2  |
// | ResourceURI | Scheme neither skill nor examples | 9       | 1, 2  |
// | ResourceURI | Empty host                        | 3, 4, 5 | 1, 2  |
// | ResourceURI | Path not empty                    | 6, 7, 8 | 1, 2  |
//
// Test mapping
// | Test                                         | Scenarios |
// |----------------------------------------------|-----------|
// | TestResourceURI/skill URI passes             | 1         |
// | TestResourceURI/examples URI passes          | 2         |
// | TestResourceURIRejects/skill without name    | 3         |
// | TestResourceURIRejects/examples without name | 4         |
// | TestResourceURIRejects/opaque URI            | 5         |
// | TestResourceURIRejects/trailing slash        | 6         |
// | TestResourceURIRejects/skill file path       | 7         |
// | TestResourceURIRejects/examples file path    | 8         |
// | TestResourceURIRejects/file scheme           | 9         |
// | TestResourceURIRejects/empty URI             | 10        |
// | TestResourceURIRejects/bad escape            | 11        |
// | No test                                      | None      |

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

func TestResourceURI(t *testing.T) {
	tests := []struct {
		name       string
		givenURI   string
		want       *pb.ResourceReadResponse
		wantParsed *url.URL
	}{
		{"skill URI passes", "skill://sample", nil, &url.URL{Scheme: "skill", Host: "sample"}},
		{"examples URI passes", "examples://sample", nil, &url.URL{Scheme: "examples", Host: "sample"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ResourceURIRequest{URI: test.givenURI}

			got, err := ResourceURI()(context.Background(), request)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantParsed, request.Parsed)
		})
	}
}

func TestResourceURIRejects(t *testing.T) {
	tests := []struct {
		name       string
		givenURI   string
		want       *pb.ResourceReadResponse
		wantCode   codes.Code
		wantDetail *pb.RPCError
		wantParsed *url.URL
	}{
		{"skill without name", "skill://", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI has no name", Data: []byte(`{"uri":"skill://"}`)}, nil},
		{"examples without name", "examples://", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI has no name", Data: []byte(`{"uri":"examples://"}`)}, nil},
		{"opaque URI", "skill:sample", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI has no name", Data: []byte(`{"uri":"skill:sample"}`)}, nil},
		{"trailing slash", "skill://sample/", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI must not have a path", Data: []byte(`{"uri":"skill://sample/"}`)}, nil},
		{"skill file path", "skill://sample/SKILL.md", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI must not have a path", Data: []byte(`{"uri":"skill://sample/SKILL.md"}`)}, nil},
		{"examples file path", "examples://sample/examples.md", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "resource URI must not have a path", Data: []byte(`{"uri":"examples://sample/examples.md"}`)}, nil},
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

			got, err := ResourceURI()(context.Background(), request)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
			assert.Equal(t, test.wantParsed, request.Parsed)
		})
	}
}
