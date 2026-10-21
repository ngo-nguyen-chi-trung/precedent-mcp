// Test plan: builder.go
//
// Logic flow
// | Function     | Step | Action                                                                                                 | Effect                                              |
// |--------------|------|--------------------------------------------------------------------------------------------------------|-----------------------------------------------------|
// | Pipeline.Run | 1    | Run the steps in order with the same request                                                           | Each step sees the request as earlier steps left it |
// | Pipeline.Run | 2    | A step returns an error                                                                                | Stop; return that error; later steps not run        |
// | Pipeline.Run | 3    | Every step passes                                                                                      | nil                                                 |
// | NewBuilder   | 1    | Create a builder with no steps                                                                         | Build gives an empty pipeline                       |
// | Builder      | 1    | Host, Bind, Legacy, Params, Headers append Request.BindHost, Bind, BindLegacy, BindParams, BindHeaders | Steps run in the order they were added              |
// | Builder      | 2    | Build returns the steps as a Pipeline                                                                  | Pipeline ready to Run                               |
//
// Scenarios
// | Number | Function     | Result  | Value class    | Input                                                                                        | Expected                                     |
// |--------|--------------|---------|----------------|----------------------------------------------------------------------------------------------|----------------------------------------------|
// | 1      | Pipeline.Run | Success | Boundary value | Empty pipeline                                                                               | nil (step 3)                                 |
// | 2      | Pipeline.Run | Success | Middle value   | Two passing steps                                                                            | nil; both ran in order (steps 1, 3)          |
// | 3      | Pipeline.Run | Failure | Middle value   | First step fails                                                                             | Its error; second step not run (step 2)      |
// | 4      | Pipeline.Run | Failure | Boundary value | Last step fails after a pass                                                                 | Its error; both ran (steps 1, 2)             |
// | 5      | NewBuilder   | Success | Boundary value | NewBuilder().Build(); body {"jsonrpc":                                                       | nil; no step runs (step 1)                   |
// | 6      | Builder      | Failure | Middle value   | Host(); loopback connection; Host evil.com:8080                                              | InvalidHost (step 1)                         |
// | 7      | Builder      | Failure | Middle value   | Bind(); body {"jsonrpc":                                                                     | ParseError (step 1)                          |
// | 8      | Builder      | Failure | Middle value   | Bind().Legacy(); initialize 2024-11-05                                                       | LegacyInitialize (step 1)                    |
// | 9      | Builder      | Failure | Middle value   | Bind().Params(); tools/list without _meta                                                    | MissingMeta (step 1)                         |
// | 10     | Builder      | Failure | Middle value   | Bind().Headers(); tools/call without MCP headers                                             | HeaderMismatch (step 1)                      |
// | 11     | Builder      | Failure | Boundary value | Bind().Params().Legacy(); initialize without _meta                                           | MissingMeta; the call order decides (step 1) |
// | 12     | Builder      | Success | Middle value   | Host().Bind().Legacy().Params().Headers(); valid tools/call; Host localhost:8080 on loopback | nil (steps 1, 2)                             |
//
// Branch coverage
// | Function     | Condition               | True    | False |
// |--------------|-------------------------|---------|-------|
// | Pipeline.Run | A step returns an error | 3, 4    | 1, 2  |
// | Pipeline.Run | Steps given             | 2 to 4  | 1     |
// | Builder      | Steps added             | 6 to 12 | 5     |
//
// Test mapping
// | Test                                             | Scenarios |
// |--------------------------------------------------|-----------|
// | TestPipelineRun/empty pipeline                   | 1         |
// | TestPipelineRun/all steps pass in order          | 2         |
// | TestPipelineRun/first failure stops              | 3         |
// | TestPipelineRun/last failure is returned         | 4         |
// | TestBuilder/no steps                             | 5         |
// | TestBuilder/host step                            | 6         |
// | TestBuilder/bind step                            | 7         |
// | TestBuilder/legacy step                          | 8         |
// | TestBuilder/params step                          | 9         |
// | TestBuilder/headers step                         | 10        |
// | TestBuilder/call order decides                   | 11        |
// | TestBuilder/full pipeline passes a valid request | 12        |
// | No test                                          | None      |

package validator_test

import . "github.com/ngo/precedent_mcp/gateway/validator"

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngo/precedent_mcp/gateway/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
)

const (
	validMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`
	toolCall  = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"add_context",` + validMeta + `}}`
)

var (
	loopback       = &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
	parseError     = &model.RPCError{HTTPStatus: http.StatusBadRequest, Error: model.Error{Code: env.CodeParseError, Message: "parse error"}}
	missingMeta    = &model.RPCError{HTTPStatus: http.StatusBadRequest, Error: model.Error{Code: env.CodeInvalidParams, Message: "missing or invalid _meta"}}
	headerMismatch = &model.RPCError{HTTPStatus: http.StatusBadRequest, Error: model.Error{
		Code:    env.CodeHeaderMismatch,
		Message: "MCP headers missing or do not match the request body",
	}}
)

func TestPipelineRun(t *testing.T) {
	tests := []struct {
		name             string
		givenStepResults []*model.RPCError
		want             *model.RPCError
		wantRan          []int
	}{
		{"empty pipeline", nil, nil, nil},
		{"all steps pass in order", []*model.RPCError{nil, nil}, nil, []int{0, 1}},
		{"first failure stops", []*model.RPCError{parseError, nil}, parseError, []int{0}},
		{"last failure is returned", []*model.RPCError{nil, headerMismatch}, headerMismatch, []int{0, 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotRan []int
			var pipeline Pipeline
			for i, result := range test.givenStepResults {
				pipeline = append(pipeline, func(*model.Request, *gin.Context) *model.RPCError {
					gotRan = append(gotRan, i)
					return result
				})
			}

			got := pipeline.Run(ginContext(t, toolCall, nil), &model.Request{})

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantRan, gotRan)
		})
	}
}

func TestBuilder(t *testing.T) {
	tests := []struct {
		name           string
		givenPipeline  Pipeline
		givenBody      string
		givenHeaders   map[string]string
		givenLocalAddr net.Addr
		givenHost      string
		want           *model.RPCError
	}{
		{
			name:          "no steps",
			givenPipeline: NewBuilder().Build(),
			givenBody:     `{"jsonrpc":`,
			want:          nil,
		},
		{
			name:           "host step",
			givenPipeline:  NewBuilder().Host().Build(),
			givenBody:      toolCall,
			givenLocalAddr: loopback,
			givenHost:      "evil.com:8080",
			want: &model.RPCError{HTTPStatus: http.StatusForbidden, Error: model.Error{
				Code:    env.CodeInvalidRequest,
				Message: `invalid Host header "evil.com:8080"`,
			}},
		},
		{
			name:          "bind step",
			givenPipeline: NewBuilder().Bind().Build(),
			givenBody:     `{"jsonrpc":`,
			want:          parseError,
		},
		{
			name:          "legacy step",
			givenPipeline: NewBuilder().Bind().Legacy().Build(),
			givenBody:     `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			want: &model.RPCError{HTTPStatus: http.StatusBadRequest, Error: model.Error{
				Code:    env.CodeUnsupportedVersion,
				Message: "initialize is not supported; use protocol version 2026-07-28",
				Data:    map[string]any{"supported": []string{"2026-07-28"}, "requested": "2024-11-05"},
			}},
		},
		{
			name:          "params step",
			givenPipeline: NewBuilder().Bind().Params().Build(),
			givenBody:     `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			want:          missingMeta,
		},
		{
			name:          "headers step",
			givenPipeline: NewBuilder().Bind().Headers().Build(),
			givenBody:     toolCall,
			want:          headerMismatch,
		},
		{
			name:          "call order decides",
			givenPipeline: NewBuilder().Bind().Params().Legacy().Build(),
			givenBody:     `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			want:          missingMeta,
		},
		{
			name:          "full pipeline passes a valid request",
			givenPipeline: NewBuilder().Host().Bind().Legacy().Params().Headers().Build(),
			givenBody:     toolCall,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodToolsCall),
				string(env.HeaderName):            "add_context",
			},
			givenLocalAddr: loopback,
			givenHost:      "localhost:8080",
			want:           nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, test.givenBody, test.givenHeaders)
			c.Request.Host = test.givenHost
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), http.LocalAddrContextKey, test.givenLocalAddr))

			got := test.givenPipeline.Run(c, &model.Request{})

			assert.Equal(t, test.want, got)
		})
	}
}
