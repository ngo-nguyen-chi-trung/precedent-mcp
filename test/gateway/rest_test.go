// Test plan: rest.go
//
// Logic flow
// | Function          | Step | Action                                                           | Effect                                                                      |
// |-------------------|------|------------------------------------------------------------------|-----------------------------------------------------------------------------|
// | NewGateway        | 1    | Pipe = Host, Bind, Legacy, Params, Headers                       | Host check before body parsing; Legacy before Params; Params before Headers |
// | NewGateway        | 2    | Route table: seven methods → register(client method, To…, From…) | initialize has no route; the Legacy step answers it                         |
// | register          | 1    | Call the gRPC method with the To… request                        | gRPC carries business input only                                            |
// | register          | 2    | gRPC error → model.FromStatus                                    | Relay protocol error or internal error                                      |
// | register          | 3    | Success → From… fills the result                                 | is_error tool results stay HTTP 200                                         |
// | CORSConfig        | 1    | AllowOrigins = env.GetAllowedOrigins(); wildcard; MCP headers    | One cors configuration for the startup check and the router                 |
// | CORSConfig        | 2    | cors.Config.Validate, run by cmd/gateway right after LoadEnv     | Invalid ALLOWED_ORIGINS entry → error; the gateway stops at startup         |
// | Router            | 1    | cors.New(CORSConfig())                                           | ALLOWED_ORIGINS replaces the loopback defaults                              |
// | Router            | 2    | Origin present and not allowed                                   | HTTP 403; relay not called; absent Origin passes                            |
// | Router            | 3    | Only POST /mcp has a handler                                     | Other methods → HTTP 405 with Allow: POST                                   |
// | Router            | 4    | SetTrustedProxies(nil): no proxy is trusted; error returned      | ClientIP is the connection peer; X-Forwarded-For is ignored                 |
// | Gateway.HandleMCP | 1    | Run the pipe                                                     | First failing step decides the JSON-RPC error and HTTP status               |
// | Gateway.HandleMCP | 2    | Pipe passed and method has no route                              | MethodNotFound -32601, HTTP 404                                             |
// | Gateway.HandleMCP | 3    | SetID(request id); error → its HTTP status; else HTTP 200        | Every response carries jsonrpc 2.0 and the id (null when invalid)           |
//
// Scenarios
// | Number | Function          | Result  | Value class    | Input                                                                     | Expected                                                                             |
// |--------|-------------------|---------|----------------|---------------------------------------------------------------------------|--------------------------------------------------------------------------------------|
// | 1      | Gateway.HandleMCP | Failure | Middle value   | Local 127.0.0.1:8080; Host evil.com:8080; body {"jsonrpc":                | 403; -32600 invalid Host header "evil.com:8080"; id null; not a parse error (step 1) |
// | 2      | Gateway.HandleMCP | Failure | Middle value   | Body {"jsonrpc":                                                          | 400; -32700; id null (step 1)                                                        |
// | 3      | Gateway.HandleMCP | Failure | Middle value   | jsonrpc 1.0 with id "x"                                                   | 400; -32600; id "x" echoed (steps 1, 3)                                              |
// | 4      | Gateway.HandleMCP | Failure | Middle value   | initialize 2024-11-05 without _meta                                       | 400; -32022; requested 2024-11-05 in data; Legacy before Params (step 1)             |
// | 5      | Gateway.HandleMCP | Failure | Middle value   | tools/list without _meta; headers present                                 | 400; -32602 (step 1)                                                                 |
// | 6      | Gateway.HandleMCP | Failure | Middle value   | tools/list with Mcp-Method server/discover                                | 400; -32020 (step 1)                                                                 |
// | 7      | Gateway.HandleMCP | Failure | Middle value   | prompts/list; valid _meta and headers                                     | 404; -32601 "method not found: prompts/list" (step 2)                                |
// | 8      | register          | Success | Middle value   | server/discover                                                           | 200; Discover receives Empty; discover result (steps 1, 3)                           |
// | 9      | register          | Success | Middle value   | tools/list                                                                | 200; ListTools receives Empty; tools with schemas and cache hints (steps 1, 3)       |
// | 10     | register          | Success | Middle value   | tools/call retrieve_context with arguments {"query":"tabs"}               | 200; CallTool receives name and arguments only; isError false (steps 1, 3)           |
// | 11     | register          | Failure | Middle value   | tools/call review_decision; relay returns is_error                        | 200; isError true; error data in structuredContent (step 3)                          |
// | 12     | register          | Success | Middle value   | resources/list                                                            | 200; ListResources receives Empty; resources (steps 1, 3)                            |
// | 13     | register          | Success | Middle value   | resources/read skill://precedent-recall                                   | 200; ReadResource receives the uri only; contents (steps 1, 3)                       |
// | 14     | register          | Success | Middle value   | skills/list                                                               | 200; ListSkills receives Empty; skills (steps 1, 3)                                  |
// | 15     | register          | Success | Middle value   | skills/get skill://precedent-recall                                       | 200; GetSkill receives the uri only; skill and cache hints (steps 1, 3)              |
// | 16     | register          | Failure | Middle value   | resources/read skill://nope; relay InvalidArgument with RPCError and data | 400; -32602 "resource not found"; data uri (step 2)                                  |
// | 17     | register          | Failure | Middle value   | tools/list; relay Unavailable                                             | 500; -32603 "internal error" (step 2)                                                |
// | 18     | Router            | Failure | Middle value   | GET /mcp                                                                  | 405; Allow: POST (step 3)                                                            |
// | 19     | Router            | Failure | Middle value   | Origin https://evil.com                                                   | 403; relay not called (step 2)                                                       |
// | 20     | Router            | Success | Middle value   | Origin http://localhost:3000                                              | 200; Access-Control-Allow-Origin echoes it (step 2)                                  |
// | 21     | Router            | Success | Middle value   | ALLOWED_ORIGINS=https://app.test; Origin https://app.test                 | 200; Access-Control-Allow-Origin echoes it (step 1)                                  |
// | 22     | Router            | Failure | Boundary value | ALLOWED_ORIGINS=https://app.test; Origin http://localhost:3000            | 403; the list replaces the defaults (step 1)                                         |
// | 23     | CORSConfig        | Success | Boundary value | ALLOWED_ORIGINS unset                                                     | Validate nil; the loopback defaults are valid (step 2)                               |
// | 24     | CORSConfig        | Success | Middle value   | ALLOWED_ORIGINS=https://app.test                                          | Validate nil (step 2)                                                                |
// | 25     | CORSConfig        | Failure | Middle value   | ALLOWED_ORIGINS=https://a.com, https://b.com                              | bad origin: origins must contain '*' or include http://,https:// (step 2)            |
// | 26     | CORSConfig        | Failure | Boundary value | ALLOWED_ORIGINS=https://a.com,https://b.com,                              | bad origin: origins must contain '*' or include http://,https:// (step 2)            |
// | 27     | Router            | Success | Middle value   | Peer 10.0.0.1:52000; X-Forwarded-For 203.0.113.9                          | ClientIP 10.0.0.1; the forwarded address is not trusted (step 4)                     |
// | 28     | Router            | Success | Boundary value | Peer 10.0.0.1:52000; no X-Forwarded-For                                   | ClientIP 10.0.0.1 (step 4)                                                           |
//
// Mocks fail on any call they do not expect, so every failure before the route proves the relay was not called.
//
// Router error path: SetTrustedProxies fails only on an unparsable proxy entry, and a nil list has none (gin v1.12.0 gin.go:415-416); Router returns the error and cmd/gateway stops on it.
//
// Branch coverage
// | Function          | Condition          | True    | False   |
// |-------------------|--------------------|---------|---------|
// | Gateway.HandleMCP | Pipe passes        | 7 to 17 | 1 to 6  |
// | Gateway.HandleMCP | Method has a route | 8 to 17 | 7       |
// | register          | gRPC call fails    | 16, 17  | 8 to 15 |
// | Router            | Origin allowed     | 20, 21  | 19, 22  |
// | CORSConfig        | Every entry valid  | 23, 24  | 25, 26  |
// | Router            | Forwarded header   | 27      | 28      |
//
// Test mapping
// | Test                                                    | Scenarios |
// |---------------------------------------------------------|-----------|
// | TestHandleMCPErrors/host check runs before body parsing | 1         |
// | TestHandleMCPErrors/malformed JSON                      | 2         |
// | TestHandleMCPErrors/wrong jsonrpc version echoes the id | 3         |
// | TestHandleMCPErrors/legacy initialize without _meta     | 4         |
// | TestHandleMCPErrors/missing _meta                       | 5         |
// | TestHandleMCPErrors/method header differs               | 6         |
// | TestHandleMCPErrors/unknown method                      | 7         |
// | TestRoutes/server/discover                              | 8         |
// | TestRoutes/tools/list                                   | 9         |
// | TestRoutes/tools/call                                   | 10        |
// | TestRoutes/tools/call is_error result                   | 11        |
// | TestRoutes/resources/list                               | 12        |
// | TestRoutes/resources/read                               | 13        |
// | TestRoutes/skills/list                                  | 14        |
// | TestRoutes/skills/get                                   | 15        |
// | TestRouteErrors/relay invalid argument                  | 16        |
// | TestRouteErrors/relay unavailable                       | 17        |
// | TestRouterMethodNotAllowed/GET is not allowed           | 18        |
// | TestRouterOrigin/foreign origin is forbidden            | 19        |
// | TestRouterOrigin/localhost origin is allowed            | 20        |
// | TestRouterOrigin/configured origin is allowed           | 21        |
// | TestRouterOrigin/configured list replaces the defaults  | 22        |
// | TestCORSConfig/default origins are valid                | 23        |
// | TestCORSConfig/configured origin is valid               | 24        |
// | TestCORSConfig/space after a comma is rejected          | 25        |
// | TestCORSConfig/trailing comma is rejected               | 26        |
// | TestRouterClientIP/forwarded address is not trusted     | 27        |
// | TestRouterClientIP/peer address without forwarding      | 28        |
// | No test                                                 | None      |

package gateway_test

import . "github.com/ngo/precedent_mcp/gateway"

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	validMeta   = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`
	resultBase  = `"resultType":"complete","_meta":{"io.modelcontextprotocol/serverInfo":{"name":"precedent-mcp","version":"1.0.0"}}`
	cacheHints  = `"ttlMs":300000,"cacheScope":"public"`
	toolsList   = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{` + validMeta + `}}`
	recallEntry = `{"uri":"skill://precedent-recall","frontmatter":{"name":"precedent-recall","description":"Rebuild context"},` +
		`"resources":[{"uri":"skill://precedent-recall","digest":"sha256:9b74c9897bac770ffc029102a200c5de9b74c9897bac770ffc029102a200c5de","size":42}]}`
)

var recall = &pb.SkillEntry{
	Uri:         "skill://precedent-recall",
	Frontmatter: &pb.SkillFrontmatter{Name: "precedent-recall", Description: "Rebuild context"},
	Resources: []*pb.SkillResource{{
		Uri:    "skill://precedent-recall",
		Digest: "sha256:9b74c9897bac770ffc029102a200c5de9b74c9897bac770ffc029102a200c5de",
		Size:   42,
	}},
}

var toolsListHeaders = map[string]string{
	string(env.HeaderProtocolVersion): string(env.VersionProtocol),
	string(env.HeaderMethod):          string(env.MethodToolsList),
}

type clients struct {
	capabilities *MockCapabilityServiceClient
	tools        *MockToolServiceClient
	resources    *MockResourceServiceClient
	skills       *MockSkillServiceClient
}

func newRouter(t *testing.T) (*gin.Engine, clients) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c := clients{
		capabilities: NewMockCapabilityServiceClient(t),
		tools:        NewMockToolServiceClient(t),
		resources:    NewMockResourceServiceClient(t),
		skills:       NewMockSkillServiceClient(t),
	}
	router, err := NewGateway(c.capabilities, c.tools, c.resources, c.skills).Router()
	require.NoError(t, err)
	return router, c
}

func protoArg[T proto.Message](want T) any {
	return mock.MatchedBy(func(got T) bool { return proto.Equal(want, got) })
}

func TestHandleMCPErrors(t *testing.T) {
	tests := []struct {
		name           string
		givenBody      string
		givenHeaders   map[string]string
		givenLocalAddr net.Addr
		givenHost      string
		wantStatus     int
		want           string
	}{
		{
			name:           "host check runs before body parsing",
			givenBody:      `{"jsonrpc":`,
			givenLocalAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080},
			givenHost:      "evil.com:8080",
			wantStatus:     http.StatusForbidden,
			want:           `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"invalid Host header \"evil.com:8080\""}}`,
		},
		{
			name:       "malformed JSON",
			givenBody:  `{"jsonrpc":`,
			givenHost:  "localhost:8080",
			wantStatus: http.StatusBadRequest,
			want:       `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`,
		},
		{
			name:       "wrong jsonrpc version echoes the id",
			givenBody:  `{"jsonrpc":"1.0","id":"x","method":"tools/list"}`,
			givenHost:  "localhost:8080",
			wantStatus: http.StatusBadRequest,
			want:       `{"jsonrpc":"2.0","id":"x","error":{"code":-32600,"message":"invalid request"}}`,
		},
		{
			name:       "legacy initialize without _meta",
			givenBody:  `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			givenHost:  "localhost:8080",
			wantStatus: http.StatusBadRequest,
			want: `{"jsonrpc":"2.0","id":1,"error":{"code":-32022,"message":"initialize is not supported; use protocol version 2026-07-28",` +
				`"data":{"supported":["2026-07-28"],"requested":"2024-11-05"}}}`,
		},
		{
			name:         "missing _meta",
			givenBody:    `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
			givenHeaders: toolsListHeaders,
			givenHost:    "localhost:8080",
			wantStatus:   http.StatusBadRequest,
			want:         `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"missing or invalid _meta"}}`,
		},
		{
			name:      "method header differs",
			givenBody: toolsList,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodDiscover),
			},
			givenHost:  "localhost:8080",
			wantStatus: http.StatusBadRequest,
			want:       `{"jsonrpc":"2.0","id":1,"error":{"code":-32020,"message":"MCP headers missing or do not match the request body"}}`,
		},
		{
			name:      "unknown method",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"prompts/list","params":{` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          "prompts/list",
			},
			givenHost:  "localhost:8080",
			wantStatus: http.StatusNotFound,
			want:       `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found: prompts/list"}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, _ := newRouter(t)
			request := httptest.NewRequest(http.MethodPost, string(env.RouteMCP), strings.NewReader(test.givenBody))
			for key, value := range test.givenHeaders {
				request.Header.Set(key, value)
			}
			request.Host = test.givenHost
			request = request.WithContext(context.WithValue(request.Context(), http.LocalAddrContextKey, test.givenLocalAddr))
			got := httptest.NewRecorder()

			router.ServeHTTP(got, request)

			assert.Equal(t, test.wantStatus, got.Code)
			assert.JSONEq(t, test.want, got.Body.String())
		})
	}
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name         string
		givenBody    string
		givenHeaders map[string]string
		givenMocks   func(c clients)
		wantStatus   int
		want         string
	}{
		{
			name:      "server/discover",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodDiscover),
			},
			givenMocks: func(c clients) {
				c.capabilities.EXPECT().Discover(mock.Anything, protoArg(&pb.Empty{})).Return(&pb.DiscoverResponse{
					SupportedVersions: []string{"2026-07-28"},
					Extensions:        []string{"io.modelcontextprotocol/skills"},
				}, nil)
			},
			wantStatus: http.StatusOK,
			want: `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` + cacheHints + `,"supportedVersions":["2026-07-28"],` +
				`"capabilities":{"tools":{},"resources":{},"extensions":{"io.modelcontextprotocol/skills":{}}}}}`,
		},
		{
			name:         "tools/list",
			givenBody:    toolsList,
			givenHeaders: toolsListHeaders,
			givenMocks: func(c clients) {
				c.tools.EXPECT().ListTools(mock.Anything, protoArg(&pb.Empty{})).Return(&pb.ToolListResponse{Tools: []*pb.Tool{
					{Name: "add_context", Description: "Save context entries.",
						InputSchema: []byte(`{"type":"object","required":["entries"]}`), OutputSchema: []byte(`{"type":"object"}`)},
				}}, nil)
			},
			wantStatus: http.StatusOK,
			want: `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` + cacheHints + `,"tools":[` +
				`{"name":"add_context","description":"Save context entries.","inputSchema":{"type":"object","required":["entries"]},"outputSchema":{"type":"object"}}]}}`,
		},
		{
			name:      "tools/call",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"retrieve_context","arguments":{"query":"tabs"},` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodToolsCall),
				string(env.HeaderName):            "retrieve_context",
			},
			givenMocks: func(c clients) {
				c.tools.EXPECT().
					CallTool(mock.Anything, protoArg(&pb.ToolCallRequest{Name: "retrieve_context", Arguments: []byte(`{"query":"tabs"}`)})).
					Return(&pb.ToolCallResponse{Data: []byte(`{"results":[]}`)}, nil)
			},
			wantStatus: http.StatusOK,
			want: `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,"content":[{"type":"text","text":"{\"results\":[]}"}],` +
				`"structuredContent":{"results":[]},"isError":false}}`,
		},
		{
			name: "tools/call is_error result",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"review_decision",` +
				`"arguments":{"decision_id":"33333333-3333-3333-3333-333333333333","score":4,"reason":"worked"},` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodToolsCall),
				string(env.HeaderName):            "review_decision",
			},
			givenMocks: func(c clients) {
				c.tools.EXPECT().CallTool(mock.Anything, protoArg(&pb.ToolCallRequest{
					Name:      "review_decision",
					Arguments: []byte(`{"decision_id":"33333333-3333-3333-3333-333333333333","score":4,"reason":"worked"}`),
				})).Return(&pb.ToolCallResponse{
					Data:    []byte(`{"code":"decision_not_found","message":"decision not found: 33333333-3333-3333-3333-333333333333"}`),
					IsError: true,
				}, nil)
			},
			wantStatus: http.StatusOK,
			want: `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` +
				`"content":[{"type":"text","text":"{\"code\":\"decision_not_found\",\"message\":\"decision not found: 33333333-3333-3333-3333-333333333333\"}"}],` +
				`"structuredContent":{"code":"decision_not_found","message":"decision not found: 33333333-3333-3333-3333-333333333333"},"isError":true}}`,
		},
		{
			name:      "resources/list",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"resources/list","params":{` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodResourcesList),
			},
			givenMocks: func(c clients) {
				c.resources.EXPECT().ListResources(mock.Anything, protoArg(&pb.Empty{})).Return(&pb.ResourceListResponse{Resources: []*pb.Resource{
					{Uri: "skill://precedent-recall", Name: "precedent-recall", Description: "Rebuild context", MimeType: "text/markdown", Size: 42},
				}}, nil)
			},
			wantStatus: http.StatusOK,
			want: `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` + cacheHints + `,"resources":[` +
				`{"uri":"skill://precedent-recall","name":"precedent-recall","description":"Rebuild context","mimeType":"text/markdown","size":42}]}}`,
		},
		{
			name:      "resources/read",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"skill://precedent-recall",` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodResourcesRead),
				string(env.HeaderName):            "skill://precedent-recall",
			},
			givenMocks: func(c clients) {
				c.resources.EXPECT().ReadResource(mock.Anything, protoArg(&pb.ResourceReadRequest{Uri: "skill://precedent-recall"})).Return(&pb.ResourceReadResponse{
					Contents: []*pb.ResourceContents{{Uri: "skill://precedent-recall", MimeType: "text/markdown", Text: "# Recall\n"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			want: `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` + cacheHints + `,` +
				`"contents":[{"uri":"skill://precedent-recall","mimeType":"text/markdown","text":"# Recall\n"}]}}`,
		},
		{
			name:      "skills/list",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"skills/list","params":{` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodSkillsList),
			},
			givenMocks: func(c clients) {
				c.skills.EXPECT().ListSkills(mock.Anything, protoArg(&pb.Empty{})).Return(&pb.SkillListResponse{Skills: []*pb.SkillEntry{recall}}, nil)
			},
			wantStatus: http.StatusOK,
			want:       `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` + cacheHints + `,"skills":[` + recallEntry + `]}}`,
		},
		{
			name:      "skills/get",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"skills/get","params":{"uri":"skill://precedent-recall",` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodSkillsGet),
			},
			givenMocks: func(c clients) {
				c.skills.EXPECT().GetSkill(mock.Anything, protoArg(&pb.SkillGetRequest{Uri: "skill://precedent-recall"})).Return(&pb.SkillGetResponse{Skill: recall}, nil)
			},
			wantStatus: http.StatusOK,
			want:       `{"jsonrpc":"2.0","id":1,"result":{` + resultBase + `,` + cacheHints + `,"skill":` + recallEntry + `}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, c := newRouter(t)
			test.givenMocks(c)
			request := httptest.NewRequest(http.MethodPost, string(env.RouteMCP), strings.NewReader(test.givenBody))
			for key, value := range test.givenHeaders {
				request.Header.Set(key, value)
			}
			got := httptest.NewRecorder()

			router.ServeHTTP(got, request)

			assert.Equal(t, test.wantStatus, got.Code)
			assert.JSONEq(t, test.want, got.Body.String())
		})
	}
}

func TestRouteErrors(t *testing.T) {
	tests := []struct {
		name         string
		givenBody    string
		givenHeaders map[string]string
		givenMocks   func(t *testing.T, c clients)
		wantStatus   int
		want         string
	}{
		{
			name:      "relay invalid argument",
			givenBody: `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"skill://nope",` + validMeta + `}}`,
			givenHeaders: map[string]string{
				string(env.HeaderProtocolVersion): string(env.VersionProtocol),
				string(env.HeaderMethod):          string(env.MethodResourcesRead),
				string(env.HeaderName):            "skill://nope",
			},
			givenMocks: func(t *testing.T, c clients) {
				st, err := status.New(codes.InvalidArgument, "resource not found").WithDetails(&pb.RPCError{
					Code:    int32(env.CodeInvalidParams),
					Message: "resource not found",
					Data:    []byte(`{"uri":"skill://nope"}`),
				})
				require.NoError(t, err)
				c.resources.EXPECT().ReadResource(mock.Anything, protoArg(&pb.ResourceReadRequest{Uri: "skill://nope"})).Return(nil, st.Err())
			},
			wantStatus: http.StatusBadRequest,
			want:       `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"resource not found","data":{"uri":"skill://nope"}}}`,
		},
		{
			name:         "relay unavailable",
			givenBody:    toolsList,
			givenHeaders: toolsListHeaders,
			givenMocks: func(t *testing.T, c clients) {
				c.tools.EXPECT().ListTools(mock.Anything, protoArg(&pb.Empty{})).Return(nil, status.Error(codes.Unavailable, "conn refused"))
			},
			wantStatus: http.StatusInternalServerError,
			want:       `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"internal error"}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, c := newRouter(t)
			test.givenMocks(t, c)
			request := httptest.NewRequest(http.MethodPost, string(env.RouteMCP), strings.NewReader(test.givenBody))
			for key, value := range test.givenHeaders {
				request.Header.Set(key, value)
			}
			got := httptest.NewRecorder()

			router.ServeHTTP(got, request)

			assert.Equal(t, test.wantStatus, got.Code)
			assert.JSONEq(t, test.want, got.Body.String())
		})
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	tests := []struct {
		name        string
		givenMethod string
		wantStatus  int
		wantAllow   string
	}{
		{"GET is not allowed", http.MethodGet, http.StatusMethodNotAllowed, http.MethodPost},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, _ := newRouter(t)
			request := httptest.NewRequest(test.givenMethod, string(env.RouteMCP), nil)
			got := httptest.NewRecorder()

			router.ServeHTTP(got, request)

			assert.Equal(t, test.wantStatus, got.Code)
			assert.Equal(t, test.wantAllow, got.Header().Get("Allow"))
		})
	}
}

func TestRouterOrigin(t *testing.T) {
	tests := []struct {
		name                string
		givenAllowedOrigins string
		givenOrigin         string
		givenMocks          func(c clients)
		wantStatus          int
		wantAllowOrigin     string
	}{
		{"foreign origin is forbidden", "", "https://evil.com", func(clients) {}, http.StatusForbidden, ""},
		{"localhost origin is allowed", "", "http://localhost:3000", func(c clients) {
			c.tools.EXPECT().ListTools(mock.Anything, protoArg(&pb.Empty{})).Return(&pb.ToolListResponse{}, nil)
		}, http.StatusOK, "http://localhost:3000"},
		{"configured origin is allowed", "https://app.test", "https://app.test", func(c clients) {
			c.tools.EXPECT().ListTools(mock.Anything, protoArg(&pb.Empty{})).Return(&pb.ToolListResponse{}, nil)
		}, http.StatusOK, "https://app.test"},
		{"configured list replaces the defaults", "https://app.test", "http://localhost:3000", func(clients) {}, http.StatusForbidden, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(string(env.AllowedOrigins), test.givenAllowedOrigins)
			router, c := newRouter(t)
			test.givenMocks(c)
			request := httptest.NewRequest(http.MethodPost, string(env.RouteMCP), strings.NewReader(toolsList))
			for key, value := range toolsListHeaders {
				request.Header.Set(key, value)
			}
			request.Header.Set("Origin", test.givenOrigin)
			got := httptest.NewRecorder()

			router.ServeHTTP(got, request)

			assert.Equal(t, test.wantStatus, got.Code)
			assert.Equal(t, test.wantAllowOrigin, got.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestCORSConfig(t *testing.T) {
	tests := []struct {
		name             string
		givenEnvironment map[env.EnvKey]string
		want             error
	}{
		{"default origins are valid", map[env.EnvKey]string{}, nil},
		{"configured origin is valid", map[env.EnvKey]string{env.AllowedOrigins: "https://app.test"}, nil},
		{"space after a comma is rejected", map[env.EnvKey]string{env.AllowedOrigins: "https://a.com, https://b.com"},
			errors.New("bad origin: origins must contain '*' or include http://,https://")},
		{"trailing comma is rejected", map[env.EnvKey]string{env.AllowedOrigins: "https://a.com,https://b.com,"},
			errors.New("bad origin: origins must contain '*' or include http://,https://")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(string(env.AllowedOrigins), "")
			require.NoError(t, os.Unsetenv(string(env.AllowedOrigins)))
			for key, value := range test.givenEnvironment {
				t.Setenv(string(key), value)
			}

			got := CORSConfig().Validate()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestRouterClientIP(t *testing.T) {
	tests := []struct {
		name            string
		givenRemoteAddr string
		givenHeaders    map[string]string
		want            string
	}{
		{"forwarded address is not trusted", "10.0.0.1:52000", map[string]string{"X-Forwarded-For": "203.0.113.9"}, "10.0.0.1"},
		{"peer address without forwarding", "10.0.0.1:52000", map[string]string{}, "10.0.0.1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, _ := newRouter(t)
			c := gin.CreateTestContextOnly(httptest.NewRecorder(), router)
			c.Request = httptest.NewRequest(http.MethodPost, string(env.RouteMCP), nil)
			c.Request.RemoteAddr = test.givenRemoteAddr
			for key, value := range test.givenHeaders {
				c.Request.Header.Set(key, value)
			}

			got := c.ClientIP()

			assert.Equal(t, test.want, got)
		})
	}
}
