//go:build e2e

// Test plan: e2e session start and gateway protocol checks
//
// A business client discovers the server and lists the tools before it loads a skill or calls a tool.
//
// Logic flow
// | Step | Client call     | Path                                                                                                             | Expected                                                                                               |
// |------|-----------------|------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------|
// | P1   | Any POST /mcp   | cors: Origin host in localhost, 127.0.0.1, ::1 or ALLOWED_ORIGINS                                                | Disallowed Origin: HTTP 403                                                                            |
// | P2   | Any POST /mcp   | Gateway pipe Bind: JSON body; id string or integer; jsonrpc 2.0; method present                                  | -32700 with id null; -32600 with id null or echoed                                                     |
// | P3   | Any POST /mcp   | Gateway pipe Legacy: method initialize                                                                           | -32022 with supported and requested versions                                                           |
// | P4   | Any POST /mcp   | Gateway pipe Params: _meta protocolVersion and clientCapabilities; version 2026-07-28                            | -32602 on missing _meta; -32022 on another version                                                     |
// | P5   | Any POST /mcp   | Gateway pipe Headers: MCP-Protocol-Version, Mcp-Method mirror the body; tools/call, resources/read need Mcp-Name | -32020                                                                                                 |
// | P6   | Any POST /mcp   | Gateway route table                                                                                              | Unknown method: HTTP 404; -32601                                                                       |
// | S1   | server/discover | CapabilityService.Discover                                                                                       | HTTP 200; supportedVersions [2026-07-28]; skills extension; public cache                               |
// | S2   | tools/list      | ToolService.ListTools from assets/tools.yaml                                                                     | HTTP 200; tool names in assets/tools.yaml order; every tool has description, inputSchema, outputSchema |
// | S3   | tools/call      | ToolService.CallTool: KnownTool, Arguments, tool function                                                        | HTTP 200; one text block equal to structuredContent; unknown tool: HTTP 400; -32602                    |
//
// Scenarios
// | Number | Step   | Result  | Value class    | Input                                                              | Expected                                                            |
// |--------|--------|---------|----------------|--------------------------------------------------------------------|---------------------------------------------------------------------|
// | 1      | S1     | Success | Middle value   | server/discover                                                    | As in S1                                                            |
// | 2      | S2     | Success | Middle value   | tools/list                                                         | As in S2                                                            |
// | 3      | S3     | Success | Middle value   | get_context_thread with a new UUID and Mcp-Name get_context_thread | As in S3; isError false; entries []                                 |
// | 4      | P5     | Failure | Boundary value | Scenario 3 without Mcp-Name                                        | HTTP 400; -32020                                                    |
// | 5      | P2     | Success | Boundary value | server/discover with id "request-1"                                | HTTP 200; id "request-1"                                            |
// | 6      | P1     | Success | Middle value   | server/discover with Origin http://localhost:3000                  | HTTP 200; Access-Control-Allow-Origin echoed                        |
// | 7      | P1     | Failure | Middle value   | server/discover with Origin https://evil.com                       | HTTP 403; no Access-Control-Allow-Origin                            |
// | 8      | P1     | Failure | Middle value   | HTTP GET on /mcp                                                   | HTTP 405                                                            |
// | 9      | P2     | Failure | Middle value   | Body "{"                                                           | HTTP 400; -32700 "parse error"; id null                             |
// | 10     | P2     | Failure | Boundary value | server/discover without id; a notification                         | HTTP 400; -32600 "invalid request"; id null                         |
// | 11     | P2     | Failure | Boundary value | server/discover with id 1.5                                        | HTTP 400; -32600 "invalid request"; id null                         |
// | 12     | P2     | Failure | Middle value   | server/discover with jsonrpc "1.0"                                 | HTTP 400; -32600 "invalid request"; id echoed                       |
// | 13     | P3     | Failure | Middle value   | initialize with protocolVersion 2025-06-18                         | HTTP 400; -32022; data supported [2026-07-28], requested 2025-06-18 |
// | 14     | P4     | Failure | Boundary value | tools/list without _meta                                           | HTTP 400; -32602 "missing or invalid _meta"                         |
// | 15     | P4     | Failure | Middle value   | tools/list with _meta protocolVersion 2025-06-18                   | HTTP 400; -32022; data supported [2026-07-28], requested 2025-06-18 |
// | 16     | P5     | Failure | Middle value   | tools/list with Mcp-Method tools/call                              | HTTP 400; -32020                                                    |
// | 17     | P5     | Failure | Boundary value | tools/list without MCP-Protocol-Version                            | HTTP 400; -32020                                                    |
// | 18     | P5     | Failure | Middle value   | Scenario 3 with Mcp-Name add_context                               | HTTP 400; -32020                                                    |
// | 19     | P6     | Failure | Middle value   | prompts/list                                                       | HTTP 404; -32601 "method not found: prompts/list"                   |
// | 20     | S3     | Failure | Middle value   | tools/call of no_such_tool                                         | HTTP 400; -32602 "unknown tool: no_such_tool"                       |
//
// Branch coverage
// | Step | Condition                                       | True  | False  |
// |------|-------------------------------------------------|-------|--------|
// | P1   | Origin header sent                              | 6, 7  | 1      |
// | P1   | Origin host allowed                             | 6     | 7      |
// | P1   | HTTP method is POST                             | 1     | 8      |
// | P2   | Body is JSON                                    | 1     | 9      |
// | P2   | id is a string or an integer                    | 1, 5  | 10, 11 |
// | P2   | jsonrpc is 2.0                                  | 1     | 12     |
// | P3   | Method is initialize                            | 13    | 1      |
// | P4   | _meta present                                   | 1     | 14     |
// | P4   | _meta protocolVersion is 2026-07-28             | 1     | 15     |
// | P5   | MCP-Protocol-Version and Mcp-Method mirror body | 1     | 16, 17 |
// | P5   | Mcp-Name sent                                   | 3, 18 | 4      |
// | P5   | Mcp-Name mirrors name or uri                    | 3     | 18     |
// | P6   | Method in the route table                       | 1     | 19     |
// | S3   | Tool known                                      | 3     | 20     |
//
// Test mapping
// | Test                                               | Scenarios |
// |----------------------------------------------------|-----------|
// | TestDiscover/version, capabilities and cache hints | 1         |
// | TestToolsList/tools in tools.yaml order            | 2         |
// | TestToolCall/with Mcp-Name                         | 3         |
// | TestRejected/missing Mcp-Name                      | 4         |
// | TestStringID/string id is echoed                   | 5         |
// | TestOrigin/allowed host                            | 6         |
// | TestOrigin/disallowed host                         | 7         |
// | TestGetNotAllowed/GET on /mcp                      | 8         |
// | TestParseError/body is not JSON                    | 9         |
// | TestRejected/notification                          | 10        |
// | TestRejected/fraction id                           | 11        |
// | TestRejected/jsonrpc 1.0                           | 12        |
// | TestRejected/initialize                            | 13        |
// | TestRejected/missing _meta                         | 14        |
// | TestRejected/unsupported _meta version             | 15        |
// | TestRejected/Mcp-Method mismatch                   | 16        |
// | TestRejected/missing MCP-Protocol-Version          | 17        |
// | TestRejected/Mcp-Name mismatch                     | 18        |
// | TestRejected/unknown method                        | 19        |
// | TestRejected/unknown tool                          | 20        |
// | No test                                            | None      |

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s *E2ESuite) TestDiscover() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "version, capabilities and cache hints",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"}},
			wantStatus:   http.StatusOK,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(1),
				"result": map[string]any{
					"resultType":        "complete",
					"_meta":             serverMeta,
					"ttlMs":             float64(300000),
					"cacheScope":        "public",
					"supportedVersions": []any{"2026-07-28"},
					"capabilities": map[string]any{
						"tools":      map[string]any{},
						"resources":  map[string]any{},
						"extensions": map[string]any{"io.modelcontextprotocol/skills": map[string]any{}},
					},
				},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestToolsList() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "tools in tools.yaml order",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/list"}},
			wantStatus:   http.StatusOK,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(1),
				"result": map[string]any{
					"resultType": "complete",
					"_meta":      serverMeta,
					"ttlMs":      float64(300000),
					"cacheScope": "public",
					"tools": []any{
						map[string]any{"name": "add_context"},
						map[string]any{"name": "retrieve_context"},
						map[string]any{"name": "get_context_thread"},
						map[string]any{"name": "record_decision"},
						map[string]any{"name": "retrieve_decisions"},
						map[string]any{"name": "review_decision"},
					},
				},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			wantTools := test.want["result"].(map[string]any)["tools"].([]any)
			gotTools := got["result"].(map[string]any)["tools"].([]any)
			s.Require().Len(gotTools, len(wantTools))
			for i, tool := range wantTools {
				for _, field := range []string{"description", "inputSchema", "outputSchema"} {
					s.Require().NotEmpty(gotTools[i].(map[string]any)[field], field)
					tool.(map[string]any)[field] = gotTools[i].(map[string]any)[field]
				}
			}
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestToolCall() {
	thread := uuid.NewString()
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name: "with Mcp-Name",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "arguments": map[string]any{"thread_id": thread}, "_meta": meta,
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"thread_id":"` + thread + `","entries":[]}`}},
				"structuredContent": map[string]any{"thread_id": thread, "entries": []any{}},
				"isError":           false,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.want, got)
			s.NoError(s.schemas["get_context_thread"].Validate(got["result"].(map[string]any)["structuredContent"]))
		})
	}
}

func (s *E2ESuite) TestStringID() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "string id is echoed",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": "request-1", "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"}},
			wantStatus:   http.StatusOK,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "request-1",
				"result": map[string]any{
					"resultType":        "complete",
					"_meta":             serverMeta,
					"ttlMs":             float64(300000),
					"cacheScope":        "public",
					"supportedVersions": []any{"2026-07-28"},
					"capabilities": map[string]any{
						"tools":      map[string]any{},
						"resources":  map[string]any{},
						"extensions": map[string]any{"io.modelcontextprotocol/skills": map[string]any{}},
					},
				},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestOrigin() {
	tests := []struct {
		name            string
		givenRequest    map[string]any
		givenHeader     http.Header
		wantStatus      int
		wantAllowOrigin string
	}{
		{
			name:         "allowed host",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"},
				"Origin": {"http://localhost:3000"}},
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "http://localhost:3000",
		},
		{
			name:         "disallowed host",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"},
				"Origin": {"https://evil.com"}},
			wantStatus:      http.StatusForbidden,
			wantAllowOrigin: "",
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader

			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()

			s.Equal(test.wantStatus, res.StatusCode)
			s.Equal(test.wantAllowOrigin, res.Header.Get("Access-Control-Allow-Origin"))
		})
	}
}

func (s *E2ESuite) TestGetNotAllowed() {
	tests := []struct {
		name       string
		wantStatus int
	}{
		{"GET on /mcp", http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			res, err := s.client.Get(s.url)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()

			s.Equal(test.wantStatus, res.StatusCode)
		})
	}
}

func (s *E2ESuite) TestParseError() {
	tests := []struct {
		name       string
		givenBody  string
		wantStatus int
		want       map[string]any
	}{
		{"body is not JSON", "{", http.StatusBadRequest, map[string]any{
			"jsonrpc": "2.0",
			"id":      nil,
			"error":   map[string]any{"code": float64(-32700), "message": "parse error"},
		}},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			res, err := s.client.Post(s.url, "application/json", strings.NewReader(test.givenBody))
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestRejected() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "notification",
			givenRequest: map[string]any{"jsonrpc": "2.0", "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": nil,
				"error": map[string]any{"code": float64(-32600), "message": "invalid request"}},
		},
		{
			name:         "fraction id",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1.5, "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": nil,
				"error": map[string]any{"code": float64(-32600), "message": "invalid request"}},
		},
		{
			name:         "jsonrpc 1.0",
			givenRequest: map[string]any{"jsonrpc": "1.0", "id": 1, "method": "server/discover", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32600), "message": "invalid request"}},
		},
		{
			name:         "initialize",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-06-18"}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code":    float64(-32022),
				"message": "initialize is not supported; use protocol version 2026-07-28",
				"data":    map[string]any{"supported": []any{"2026-07-28"}, "requested": "2025-06-18"},
			}},
		},
		{
			name:         "missing _meta",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/list"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32602), "message": "missing or invalid _meta"}},
		},
		{
			name: "unsupported _meta version",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{"_meta": map[string]any{
				"io.modelcontextprotocol/protocolVersion":    "2025-06-18",
				"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2025-06-18"}, "Mcp-Method": {"tools/list"}},
			wantStatus:  http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code":    float64(-32022),
				"message": "unsupported protocol version; use 2026-07-28",
				"data":    map[string]any{"supported": []any{"2026-07-28"}, "requested": "2025-06-18"},
			}},
		},
		{
			name:         "Mcp-Method mismatch",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32020), "message": "MCP headers missing or do not match the request body"}},
		},
		{
			name:         "missing MCP-Protocol-Version",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Method": {"tools/list"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32020), "message": "MCP headers missing or do not match the request body"}},
		},
		{
			name: "missing Mcp-Name",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "arguments": map[string]any{"thread_id": uuid.NewString()}, "_meta": meta,
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"}},
			wantStatus:  http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32020), "message": "MCP headers missing or do not match the request body"}},
		},
		{
			name: "Mcp-Name mismatch",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "arguments": map[string]any{"thread_id": uuid.NewString()}, "_meta": meta,
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32020), "message": "MCP headers missing or do not match the request body"}},
		},
		{
			name:         "unknown method",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "prompts/list", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"prompts/list"}},
			wantStatus:   http.StatusNotFound,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32601), "message": "method not found: prompts/list"}},
		},
		{
			name: "unknown tool",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "no_such_tool", "arguments": map[string]any{}, "_meta": meta,
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"no_such_tool"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32602), "message": "unknown tool: no_such_tool"}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Equal(test.want, got)
		})
	}
}
