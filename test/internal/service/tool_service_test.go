// Test plan: tool_service.go
//
// Logic flow
// | Function              | Step | Action                                              | Effect                                                                                 |
// |-----------------------|------|-----------------------------------------------------|----------------------------------------------------------------------------------------|
// | ToolService.ListTools | 1    | ToolStore.Tools                                     | Store error → error (JSON-RPC internal error)                                          |
// | ToolService.ListTools | 2    | ToolListResponse.FromToolEntries                    | Tools in tools.yaml order                                                              |
// | ToolService.CallTool  | 1    | Relay pipe: KnownTool, then Arguments               | Unknown tool → error; invalid arguments → is_error; the tool is not called             |
// | ToolService.CallTool  | 2    | Call the registered tool with the arguments         | Its response returned unchanged, is_error included                                     |
// | ToolService.Register  | 1    | ToolStore.Tools                                     | Store error → error; the server cannot start                                           |
// | ToolService.Register  | 2    | Each tool needs a definition in tools.yaml          | Missing definition → error "tool <name> has no definition in tools.yaml"               |
// | ToolService.Register  | 3    | Keep each tool under its name                       | The tool is callable through CallTool                                                  |
// | Tool                  | 1    | Decode the arguments into the tool's request type   | Decode error → is_error invalid_argument "invalid args JSON: <cause>"; tool not called |
// | Tool                  | 2    | Call the tool with the same context and the request | Its response returned unchanged                                                        |
//
// Scenarios
// | Number | Function              | Result  | Value class  | Input                                            | Expected                                                                                                 |
// |--------|-----------------------|---------|--------------|--------------------------------------------------|----------------------------------------------------------------------------------------------------------|
// | 1      | ToolService.ListTools | Success | Middle value | testdata/tools.yaml                              | Tools echo, ping in file order (steps 1, 2)                                                              |
// | 2      | ToolService.ListTools | Failure | Middle value | Tool list missing                                | Nil response; error matching fs.ErrNotExist (step 1)                                                     |
// | 3      | ToolService.CallTool  | Success | Middle value | echo; {"text":"hi"}                              | Tool receives {"text":"hi"}; its response returned (step 2)                                              |
// | 4      | ToolService.CallTool  | Success | Middle value | echo; tool returns is_error                      | is_error response returned unchanged; nil error (step 2)                                                 |
// | 5      | ToolService.CallTool  | Failure | Middle value | nope; not registered                             | Nil response; error InvalidArgument; RPCError -32602 "unknown tool: nope"; tool not called (step 1)      |
// | 6      | ToolService.CallTool  | Failure | Middle value | echo; {}; required text missing                  | is_error invalid_argument "invalid args: … missing property 'text'"; tool not called (step 1)            |
// | 7      | ToolService.Register  | Success | Middle value | echo and ping; call echo                         | echo answers (steps 2, 3)                                                                                |
// | 8      | ToolService.Register  | Success | Middle value | echo and ping; call ping                         | ping answers (steps 2, 3)                                                                                |
// | 9      | ToolService.Register  | Failure | Middle value | Tool nope without definition                     | error "tool nope has no definition in tools.yaml" (step 2)                                               |
// | 10     | ToolService.Register  | Failure | Middle value | Tool list missing                                | error matching fs.ErrNotExist (step 1)                                                                   |
// | 11     | Tool                  | Success | Middle value | {"text":"hi"}                                    | Tool receives text hi; its response returned (steps 1, 2)                                                |
// | 12     | Tool                  | Success | Middle value | Context carrying a value                         | Tool receives the same context (step 2)                                                                  |
// | 13     | Tool                  | Failure | Middle value | {"text":1}; tools.yaml and request type disagree | is_error invalid_argument "invalid args JSON: json: cannot unmarshal number …"; tool not called (step 1) |
//
// One wiring case per relay pipe step; the rules of KnownTool and Arguments are proven in validator/tool_test.go, the encoding of the tool list in model/tool_response_test.go.
//
// Branch coverage
// | Function              | Condition                         | True | False  |
// |-----------------------|-----------------------------------|------|--------|
// | ToolService.ListTools | ToolStore.Tools error             | 2    | 1      |
// | ToolService.CallTool  | Pipe returns an error or a result | 5, 6 | 3, 4   |
// | ToolService.Register  | ToolStore.Tools error             | 10   | 7, 8   |
// | ToolService.Register  | Tool without definition           | 9    | 7, 8   |
// | Tool                  | Decode error                      | 13   | 11, 12 |
//
// Test mapping
// | Test                                                          | Scenarios |
// |---------------------------------------------------------------|-----------|
// | TestToolServiceListTools/tools in file order                  | 1         |
// | TestToolServiceListToolsError/tool list missing               | 2         |
// | TestToolServiceCallTool/arguments reach the tool              | 3         |
// | TestToolServiceCallTool/is_error result returned unchanged    | 4         |
// | TestToolServiceCallToolUnknownTool/name not registered        | 5         |
// | TestToolServiceCallToolInvalidArguments/required text missing | 6         |
// | TestToolServiceRegister/echo is callable                      | 7         |
// | TestToolServiceRegister/ping is callable                      | 8         |
// | TestToolServiceRegisterError/tool without definition          | 9         |
// | TestToolServiceRegisterStoreError/tool list missing           | 10        |
// | TestTool/arguments decoded for the tool                       | 11        |
// | TestTool/context reaches the tool                             | 12        |
// | TestToolError/number for a text field                         | 13        |
// | No test                                                       | None      |

package service_test

import . "github.com/ngo/precedent_mcp/internal/service"

import (
	"context"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/asset"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type toolEchoArgs struct {
	Text string `json:"text"`
}

func toolEcho(_ context.Context, a toolEchoArgs) *pb.ToolCallResponse {
	return &pb.ToolCallResponse{Data: []byte(a.Text)}
}

type toolContextKey struct{}

func toolContextValue(ctx context.Context, _ toolEchoArgs) *pb.ToolCallResponse {
	return &pb.ToolCallResponse{Data: []byte(ctx.Value(toolContextKey{}).(string))}
}

func TestToolServiceListTools(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		wantNames      []string
	}{
		{"tools in file order", "testdata", []string{"echo", "ping"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewToolService(asset.NewToolStore(test.givenDirectory), nil, nil)

			got, err := s.ListTools(context.Background(), &pb.Empty{})

			require.NoError(t, err)
			var gotNames []string
			for _, tool := range got.GetTools() {
				gotNames = append(gotNames, tool.GetName())
			}
			assert.Equal(t, test.wantNames, gotNames)
		})
	}
}

func TestToolServiceListToolsError(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		want           *pb.ToolListResponse
		wantError      error
	}{
		{"tool list missing", filepath.Join("testdata", "missing"), nil, fs.ErrNotExist},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewToolService(asset.NewToolStore(test.givenDirectory), nil, nil)

			got, err := s.ListTools(context.Background(), &pb.Empty{})

			assert.Equal(t, test.want, got)
			assert.ErrorIs(t, err, test.wantError)
		})
	}
}

func TestToolServiceCallTool(t *testing.T) {
	tests := []struct {
		name          string
		givenRequest  *pb.ToolCallRequest
		givenResult   *pb.ToolCallResponse
		wantArguments string
		want          *pb.ToolCallResponse
	}{
		{"arguments reach the tool",
			&pb.ToolCallRequest{Name: "echo", Arguments: []byte(`{"text":"hi"}`)},
			&pb.ToolCallResponse{Data: []byte(`{"text":"hi"}`)},
			`{"text":"hi"}`,
			&pb.ToolCallResponse{Data: []byte(`{"text":"hi"}`)}},
		{"is_error result returned unchanged",
			&pb.ToolCallRequest{Name: "echo", Arguments: []byte(`{"text":"hi"}`)},
			&pb.ToolCallResponse{Data: []byte(`{"code":"internal","message":"weaviate down"}`), IsError: true},
			`{"text":"hi"}`,
			&pb.ToolCallResponse{Data: []byte(`{"code":"internal","message":"weaviate down"}`), IsError: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotArguments []byte
			s := NewToolService(asset.NewToolStore("testdata"), nil, nil)
			require.NoError(t, s.Register(map[env.ToolName]ToolFunc{"echo": func(_ context.Context, args []byte) *pb.ToolCallResponse {
				gotArguments = args
				return test.givenResult
			}}))

			got, err := s.CallTool(context.Background(), test.givenRequest)

			require.NoError(t, err)
			assert.Equal(t, test.wantArguments, string(gotArguments))
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestToolServiceCallToolUnknownTool(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest *pb.ToolCallRequest
		want         *pb.ToolCallResponse
		wantCode     codes.Code
		wantDetail   *pb.RPCError
		wantCalled   bool
	}{
		{"name not registered", &pb.ToolCallRequest{Name: "nope", Arguments: []byte(`{}`)}, nil, codes.InvalidArgument,
			&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: nope"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotCalled := false
			s := NewToolService(asset.NewToolStore("testdata"), nil, nil)
			require.NoError(t, s.Register(map[env.ToolName]ToolFunc{"echo": func(context.Context, []byte) *pb.ToolCallResponse {
				gotCalled = true
				return &pb.ToolCallResponse{}
			}}))

			got, err := s.CallTool(context.Background(), test.givenRequest)

			gotStatus := status.Convert(err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantCode, gotStatus.Code())
			require.Len(t, gotStatus.Details(), 1)
			assert.True(t, proto.Equal(test.wantDetail, gotStatus.Details()[0].(*pb.RPCError)), "got %v", gotStatus.Details()[0])
			assert.Equal(t, test.wantCalled, gotCalled)
		})
	}
}

func TestToolServiceCallToolInvalidArguments(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest *pb.ToolCallRequest
		want         model.Error
		wantCalled   bool
	}{
		{"required text missing", &pb.ToolCallRequest{Name: "echo", Arguments: []byte(`{}`)},
			model.Error{
				Code:    env.ServiceErrorInvalidArgument,
				Message: "invalid args: jsonschema validation failed with 'tool:echo#'\n- at '': missing property 'text'",
			}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotCalled := false
			s := NewToolService(asset.NewToolStore("testdata"), nil, nil)
			require.NoError(t, s.Register(map[env.ToolName]ToolFunc{"echo": func(context.Context, []byte) *pb.ToolCallResponse {
				gotCalled = true
				return &pb.ToolCallResponse{}
			}}))

			got, err := s.CallTool(context.Background(), test.givenRequest)

			require.NoError(t, err)
			require.True(t, got.GetIsError())
			var gotError model.Error
			require.NoError(t, json.Unmarshal(got.GetData(), &gotError))
			assert.Equal(t, test.want, gotError)
			assert.Equal(t, test.wantCalled, gotCalled)
		})
	}
}

func TestToolServiceRegister(t *testing.T) {
	tests := []struct {
		name         string
		givenTools   map[env.ToolName]ToolFunc
		givenRequest *pb.ToolCallRequest
		want         *pb.ToolCallResponse
	}{
		{"echo is callable",
			map[env.ToolName]ToolFunc{
				"echo": func(context.Context, []byte) *pb.ToolCallResponse {
					return &pb.ToolCallResponse{Data: []byte("echo answer")}
				},
				"ping": func(context.Context, []byte) *pb.ToolCallResponse {
					return &pb.ToolCallResponse{Data: []byte("ping answer")}
				},
			},
			&pb.ToolCallRequest{Name: "echo", Arguments: []byte(`{"text":"hi"}`)},
			&pb.ToolCallResponse{Data: []byte("echo answer")}},
		{"ping is callable",
			map[env.ToolName]ToolFunc{
				"echo": func(context.Context, []byte) *pb.ToolCallResponse {
					return &pb.ToolCallResponse{Data: []byte("echo answer")}
				},
				"ping": func(context.Context, []byte) *pb.ToolCallResponse {
					return &pb.ToolCallResponse{Data: []byte("ping answer")}
				},
			},
			&pb.ToolCallRequest{Name: "ping"},
			&pb.ToolCallResponse{Data: []byte("ping answer")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewToolService(asset.NewToolStore("testdata"), nil, nil)

			err := s.Register(test.givenTools)

			require.NoError(t, err)
			got, err := s.CallTool(context.Background(), test.givenRequest)
			require.NoError(t, err)
			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestToolServiceRegisterError(t *testing.T) {
	tests := []struct {
		name       string
		givenTools map[env.ToolName]ToolFunc
		wantError  string
	}{
		{"tool without definition", map[env.ToolName]ToolFunc{"nope": Tool(toolEcho)}, "tool nope has no definition in tools.yaml"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewToolService(asset.NewToolStore("testdata"), nil, nil)

			err := s.Register(test.givenTools)

			assert.EqualError(t, err, test.wantError)
		})
	}
}

func TestToolServiceRegisterStoreError(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenTools     map[env.ToolName]ToolFunc
		wantError      error
	}{
		{"tool list missing", filepath.Join("testdata", "missing"), map[env.ToolName]ToolFunc{"echo": Tool(toolEcho)}, fs.ErrNotExist},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewToolService(asset.NewToolStore(test.givenDirectory), nil, nil)

			err := s.Register(test.givenTools)

			assert.ErrorIs(t, err, test.wantError)
		})
	}
}

func TestTool(t *testing.T) {
	tests := []struct {
		name              string
		givenContextValue string
		givenArguments    []byte
		givenTool         func(context.Context, toolEchoArgs) *pb.ToolCallResponse
		want              *pb.ToolCallResponse
	}{
		{"arguments decoded for the tool", "", []byte(`{"text":"hi"}`), toolEcho, &pb.ToolCallResponse{Data: []byte("hi")}},
		{"context reaches the tool", "request scope", []byte(`{"text":"hi"}`), toolContextValue, &pb.ToolCallResponse{Data: []byte("request scope")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), toolContextKey{}, test.givenContextValue)

			got := Tool(test.givenTool)(ctx, test.givenArguments)

			assert.True(t, proto.Equal(test.want, got), "got %v", got)
		})
	}
}

func TestToolError(t *testing.T) {
	tests := []struct {
		name           string
		givenArguments []byte
		want           model.Error
		wantCalled     bool
	}{
		{"number for a text field", []byte(`{"text":1}`), model.Error{
			Code:    env.ServiceErrorInvalidArgument,
			Message: "invalid args JSON: json: cannot unmarshal number into Go struct field toolEchoArgs.text of type string",
		}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotCalled := false
			tool := Tool(func(context.Context, toolEchoArgs) *pb.ToolCallResponse {
				gotCalled = true
				return &pb.ToolCallResponse{}
			})

			got := tool(context.Background(), test.givenArguments)

			require.True(t, got.GetIsError())
			var gotError model.Error
			require.NoError(t, json.Unmarshal(got.GetData(), &gotError))
			assert.Equal(t, test.want, gotError)
			assert.Equal(t, test.wantCalled, gotCalled)
		})
	}
}
