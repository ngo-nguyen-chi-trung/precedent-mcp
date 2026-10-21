// Test plan: tool.go
//
// Logic flow
// | Function  | Step | Action                                          | Effect                                                                                        |
// |-----------|------|-------------------------------------------------|-----------------------------------------------------------------------------------------------|
// | KnownTool | 1    | Read the validated entries from ToolStore.Tools | Store error → that error unchanged (JSON-RPC internal error); Schema stays nil                |
// | KnownTool | 2    | Name registered and defined in tools.yaml       | Otherwise → error InvalidArgument, RPCError -32602 "unknown tool: <name>"; Schema nil         |
// | KnownTool | 3    | Set Schema to the entry's compiled input schema | Pass: nil result, nil error                                                                   |
// | Arguments | 1    | Schema not set                                  | error InvalidArgument, RPCError -32602 "unknown tool: <name>"                                 |
// | Arguments | 2    | Arguments absent → {}                           | The tool later receives {}                                                                    |
// | Arguments | 3    | Parse the arguments as JSON                     | Not JSON → is_error invalid_argument "invalid args JSON: <cause>"                             |
// | Arguments | 4    | Validate against Schema                         | Violation → is_error invalid_argument "invalid args: <cause>"; the model can correct its call |
// | Arguments | 5    | Valid                                           | Pass: nil result, nil error                                                                   |
//
// Scenarios
// | Number | Function  | Result  | Value class    | Input                                      | Expected                                                                                    |
// |--------|-----------|---------|----------------|--------------------------------------------|---------------------------------------------------------------------------------------------|
// | 1      | KnownTool | Success | Middle value   | echo registered and in testdata/tools.yaml | Pass; Schema is the compiled tool:echo schema (step 3)                                      |
// | 2      | KnownTool | Failure | Middle value   | ping in tools.yaml, not registered         | error InvalidArgument; RPCError -32602 "unknown tool: ping"; Schema nil (step 2)            |
// | 3      | KnownTool | Failure | Middle value   | nope registered, not in tools.yaml         | error InvalidArgument; RPCError -32602 "unknown tool: nope"; Schema nil (step 2)            |
// | 4      | KnownTool | Failure | Boundary value | Empty tool name                            | error InvalidArgument; RPCError -32602 "unknown tool: "; Schema nil (step 2)                |
// | 5      | KnownTool | Failure | Middle value   | tools.yaml missing                         | error matching fs.ErrNotExist; no RPCError detail; Schema nil (step 1)                      |
// | 6      | Arguments | Success | Middle value   | echo; {"text":"hi"}                        | Pass; arguments unchanged (step 5)                                                          |
// | 7      | Arguments | Success | Boundary value | echo; {"text":"a"}; minLength 1            | Pass; arguments unchanged (step 5)                                                          |
// | 8      | Arguments | Success | Boundary value | ping; no arguments                         | Pass; arguments {} (step 2)                                                                 |
// | 9      | Arguments | Failure | Boundary value | echo; {"text":""}; below minLength         | is_error invalid_argument "invalid args: … at '/text': minLength: got 0, want 1" (step 4)   |
// | 10     | Arguments | Failure | Middle value   | echo; {}; required text missing            | is_error invalid_argument "invalid args: … at '': missing property 'text'" (step 4)         |
// | 11     | Arguments | Failure | Boundary value | echo; no arguments                         | Arguments {}; is_error "invalid args: … missing property 'text'" (steps 2, 4)               |
// | 12     | Arguments | Failure | Middle value   | echo; {"text":1}; wrong type               | is_error invalid_argument "invalid args: … at '/text': got number, want string" (step 4)    |
// | 13     | Arguments | Failure | Middle value   | echo; []; array, not an object             | is_error invalid_argument "invalid args: … at '': got array, want object" (step 4)          |
// | 14     | Arguments | Failure | Boundary value | echo; null                                 | is_error invalid_argument "invalid args: … at '': got null, want object" (step 4)           |
// | 15     | Arguments | Failure | Middle value   | echo; {"text": ; not JSON                  | is_error invalid_argument "invalid args JSON: unexpected EOF" (step 3)                      |
// | 16     | Arguments | Failure | Middle value   | nope; Schema not set; no arguments         | error InvalidArgument; RPCError -32602 "unknown tool: nope"; arguments stay absent (step 1) |
//
// Toy schemas: testdata/tools.yaml through asset.ToolStore, so the schema location is tool:<name>. Production tools.yaml constraints are covered by e2e.
//
// Branch coverage
// | Function  | Condition                     | True    | False   |
// |-----------|-------------------------------|---------|---------|
// | KnownTool | ToolStore.Tools error         | 5       | 1       |
// | KnownTool | Not registered or not defined | 2, 3, 4 | 1       |
// | Arguments | Schema nil                    | 16      | 6       |
// | Arguments | Arguments absent              | 8, 11   | 6       |
// | Arguments | Arguments not JSON            | 15      | 6       |
// | Arguments | Schema violation              | 9 to 14 | 6, 7, 8 |
//
// Test mapping
// | Test                                             | Scenarios |
// |--------------------------------------------------|-----------|
// | TestKnownTool/registered and defined tool passes | 1         |
// | TestKnownToolRejects/defined but not registered  | 2         |
// | TestKnownToolRejects/registered but not defined  | 3         |
// | TestKnownToolRejects/empty name                  | 4         |
// | TestKnownToolError/tool list missing             | 5         |
// | TestArguments/valid arguments pass               | 6         |
// | TestArguments/minimum length passes              | 7         |
// | TestArguments/absent arguments become {}         | 8         |
// | TestArgumentsRejects/below minimum length        | 9         |
// | TestArgumentsRejects/text missing                | 10        |
// | TestArgumentsRejects/absent arguments miss text  | 11        |
// | TestArgumentsRejects/wrong type                  | 12        |
// | TestArgumentsRejects/array                       | 13        |
// | TestArgumentsRejects/null                        | 14        |
// | TestArgumentsRejects/not JSON                    | 15        |
// | TestArgumentsUnknownTool/schema not set          | 16        |
// | No test                                          | None      |

package validator_test

import . "github.com/ngo/precedent_mcp/internal/service/validator"

import (
	"context"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"slices"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/asset"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestKnownTool(t *testing.T) {
	tests := []struct {
		name               string
		givenDirectory     string
		givenRegistered    map[env.ToolName]struct{}
		givenName          string
		want               *pb.ToolCallResponse
		wantSchemaLocation string
	}{
		{"registered and defined tool passes", "testdata", map[env.ToolName]struct{}{"echo": {}}, "echo", nil, "tool:echo#"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ToolCallRequest{ToolCallRequest: &pb.ToolCallRequest{Name: test.givenName}}

			got, err := KnownTool(test.givenRegistered, asset.NewToolStore(test.givenDirectory))(context.Background(), request)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			require.NotNil(t, request.Schema)
			assert.Equal(t, test.wantSchemaLocation, request.Schema.Location)
		})
	}
}

func TestKnownToolRejects(t *testing.T) {
	tests := []struct {
		name            string
		givenDirectory  string
		givenRegistered map[env.ToolName]struct{}
		givenName       string
		want            *pb.ToolCallResponse
		wantCode        codes.Code
		wantDetail      *pb.RPCError
		wantSchema      *jsonschema.Schema
	}{
		{"defined but not registered", "testdata", map[env.ToolName]struct{}{"echo": {}}, "ping", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: ping"}, nil},
		{"registered but not defined", "testdata", map[env.ToolName]struct{}{"nope": {}}, "nope", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: nope"}, nil},
		{"empty name", "testdata", map[env.ToolName]struct{}{"echo": {}}, "", nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: "}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ToolCallRequest{ToolCallRequest: &pb.ToolCallRequest{Name: test.givenName}}

			got, err := KnownTool(test.givenRegistered, asset.NewToolStore(test.givenDirectory))(context.Background(), request)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
			assert.Equal(t, test.wantSchema, request.Schema)
		})
	}
}

func TestKnownToolError(t *testing.T) {
	tests := []struct {
		name            string
		givenDirectory  string
		givenRegistered map[env.ToolName]struct{}
		givenName       string
		want            *pb.ToolCallResponse
		wantError       error
		wantDetails     []any
		wantSchema      *jsonschema.Schema
	}{
		{"tool list missing", filepath.Join("testdata", "missing"), map[env.ToolName]struct{}{"echo": {}}, "echo", nil, fs.ErrNotExist, []any{}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ToolCallRequest{ToolCallRequest: &pb.ToolCallRequest{Name: test.givenName}}

			got, err := KnownTool(test.givenRegistered, asset.NewToolStore(test.givenDirectory))(context.Background(), request)

			assert.Equal(t, test.want, got)
			assert.ErrorIs(t, err, test.wantError)
			assert.Equal(t, test.wantDetails, status.Convert(err).Details())
			assert.Equal(t, test.wantSchema, request.Schema)
		})
	}
}

func TestArguments(t *testing.T) {
	tests := []struct {
		name           string
		givenName      string
		givenArguments []byte
		want           *pb.ToolCallResponse
		wantArguments  string
	}{
		{"valid arguments pass", "echo", []byte(`{"text":"hi"}`), nil, `{"text":"hi"}`},
		{"minimum length passes", "echo", []byte(`{"text":"a"}`), nil, `{"text":"a"}`},
		{"absent arguments become {}", "ping", nil, nil, `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, err := asset.NewToolStore("testdata").Tools()
			require.NoError(t, err)
			entry := entries[slices.IndexFunc(entries, func(e assetmodel.ToolEntry) bool { return e.Name == test.givenName })]
			request := &model.ToolCallRequest{
				ToolCallRequest: &pb.ToolCallRequest{Name: test.givenName, Arguments: test.givenArguments},
				Schema:          entry.Compiled,
			}

			got, err := Arguments()(context.Background(), request)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantArguments, string(request.GetArguments()))
		})
	}
}

func TestArgumentsRejects(t *testing.T) {
	tests := []struct {
		name           string
		givenName      string
		givenArguments []byte
		want           model.Error
		wantArguments  string
	}{
		{"below minimum length", "echo", []byte(`{"text":""}`), model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '/text': minLength: got 0, want 1"}, `{"text":""}`},
		{"text missing", "echo", []byte(`{}`), model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '': missing property 'text'"}, `{}`},
		{"absent arguments miss text", "echo", nil, model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '': missing property 'text'"}, `{}`},
		{"wrong type", "echo", []byte(`{"text":1}`), model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '/text': got number, want string"}, `{"text":1}`},
		{"array", "echo", []byte(`[]`), model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '': got array, want object"}, `[]`},
		{"null", "echo", []byte(`null`), model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '': got null, want object"}, `null`},
		{"not JSON", "echo", []byte(`{"text":`), model.Error{Code: env.ServiceErrorInvalidArgument,
			Message: "invalid args JSON: unexpected EOF"}, `{"text":`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, err := asset.NewToolStore("testdata").Tools()
			require.NoError(t, err)
			entry := entries[slices.IndexFunc(entries, func(e assetmodel.ToolEntry) bool { return e.Name == test.givenName })]
			request := &model.ToolCallRequest{
				ToolCallRequest: &pb.ToolCallRequest{Name: test.givenName, Arguments: test.givenArguments},
				Schema:          entry.Compiled,
			}

			got, err := Arguments()(context.Background(), request)

			require.NoError(t, err)
			require.True(t, got.GetIsError())
			var gotError model.Error
			require.NoError(t, json.Unmarshal(got.GetData(), &gotError))
			assert.Equal(t, test.want, gotError)
			assert.Equal(t, test.wantArguments, string(request.GetArguments()))
		})
	}
}

func TestArgumentsUnknownTool(t *testing.T) {
	tests := []struct {
		name           string
		givenName      string
		givenArguments []byte
		want           *pb.ToolCallResponse
		wantCode       codes.Code
		wantDetail     *pb.RPCError
		wantArguments  []byte
	}{
		{"schema not set", "nope", nil, nil, codes.InvalidArgument, &pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: nope"}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &model.ToolCallRequest{ToolCallRequest: &pb.ToolCallRequest{Name: test.givenName, Arguments: test.givenArguments}}

			got, err := Arguments()(context.Background(), request)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
			assert.Equal(t, test.wantArguments, request.GetArguments())
		})
	}
}
