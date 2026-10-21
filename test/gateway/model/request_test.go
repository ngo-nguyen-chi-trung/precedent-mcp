// Test plan: request.go
//
// Logic flow
// | Function            | Step | Action                                                                      | Effect                                                                                         |
// |---------------------|------|-----------------------------------------------------------------------------|------------------------------------------------------------------------------------------------|
// | Request.ToEmpty     | 1    | Build an empty protobuf request                                             | Parameterless gRPC methods receive no protocol data                                            |
// | Request.BindHost    | 1    | Connection local address is a loopback address                              | Otherwise no check: the rule protects localhost servers only                                   |
// | Request.BindHost    | 2    | Host header hostname is localhost or a loopback IP                          | Pass                                                                                           |
// | Request.BindHost    | 3    | Otherwise                                                                   | InvalidHost: HTTP 403, -32600 `invalid Host header "<host>"` (DNS rebinding)                   |
// | Request.Bind        | 1    | Decode the body as one JSON-RPC request                                     | Not JSON → ParseError -32700, HTTP 400, id null                                                |
// | Request.Bind        | 2    | id is a string or an integer                                                | Absent (notification), null, fraction or other type → InvalidRequest -32600, HTTP 400, id null |
// | Request.Bind        | 3    | jsonrpc is "2.0"; method is a string; every field has its JSON type         | Otherwise → InvalidRequest -32600, HTTP 400, id kept; also inside params                       |
// | Request.BindLegacy  | 1    | method is initialize                                                        | LegacyInitialize -32022, HTTP 400, data supported and requested protocolVersion                |
// | Request.BindParams  | 1    | params._meta holds a protocolVersion string and a clientCapabilities object | Missing or invalid → MissingMeta -32602, HTTP 400                                              |
// | Request.BindParams  | 2    | protocolVersion equals VersionProtocol 2026-07-28                           | Otherwise → UnsupportedVersion -32022, HTTP 400, data supported and requested                  |
// | Request.BindHeaders | 1    | MCP-Protocol-Version and Mcp-Method headers present                         | Missing → HeaderMismatch -32020, HTTP 400                                                      |
// | Request.BindHeaders | 2    | MCP-Protocol-Version equals _meta protocolVersion; Mcp-Method equals method | Otherwise → HeaderMismatch                                                                     |
// | Request.BindHeaders | 3    | Mcp-Name required on tools/call, resources/read; equals params.name or uri  | Missing on those methods or different → HeaderMismatch; other methods may omit it              |
//
// Scenarios
// | Number | Function            | Result  | Value class    | Input                                                                      | Expected                                          |
// |--------|---------------------|---------|----------------|----------------------------------------------------------------------------|---------------------------------------------------|
// | 1      | Request.ToEmpty     | Success | Middle value   | tools/list request                                                         | Empty message (step 1)                            |
// | 2      | Request.Bind        | Success | Middle value   | id "a"                                                                     | nil; ID "a" (steps 1 to 3)                        |
// | 3      | Request.Bind        | Success | Middle value   | id 7                                                                       | nil; ID 7 (step 2)                                |
// | 4      | Request.Bind        | Success | Boundary value | id 0                                                                       | nil; ID 0 (step 2)                                |
// | 5      | Request.Bind        | Success | Boundary value | id ""                                                                      | nil; ID "" (step 2)                               |
// | 6      | Request.Bind        | Failure | Middle value   | Body {"jsonrpc":                                                           | ParseError; ID nil (step 1)                       |
// | 7      | Request.Bind        | Failure | Boundary value | Empty body                                                                 | ParseError; ID nil (step 1)                       |
// | 8      | Request.Bind        | Failure | Middle value   | Body [] (batch)                                                            | InvalidRequest; ID nil (step 2)                   |
// | 9      | Request.Bind        | Failure | Boundary value | No id (notification)                                                       | InvalidRequest; ID nil (step 2)                   |
// | 10     | Request.Bind        | Failure | Boundary value | id null                                                                    | InvalidRequest; ID nil (step 2)                   |
// | 11     | Request.Bind        | Failure | Boundary value | id 1.5                                                                     | InvalidRequest; ID nil (step 2)                   |
// | 12     | Request.Bind        | Failure | Middle value   | id true                                                                    | InvalidRequest; ID nil (step 2)                   |
// | 13     | Request.Bind        | Failure | Middle value   | jsonrpc "1.0", id 1                                                        | InvalidRequest; ID 1 kept (step 3)                |
// | 14     | Request.Bind        | Failure | Boundary value | No jsonrpc, id 1                                                           | InvalidRequest; ID 1 kept (step 3)                |
// | 15     | Request.Bind        | Failure | Boundary value | No method, id 1                                                            | InvalidRequest; ID 1 kept (step 3)                |
// | 16     | Request.Bind        | Failure | Middle value   | method 5, id 1                                                             | InvalidRequest; ID 1 kept (step 3)                |
// | 17     | Request.Bind        | Failure | Middle value   | params._meta protocolVersion 5 (a number), id 1                            | InvalidRequest; ID 1 kept (step 3)                |
// | 18     | Request.BindLegacy  | Failure | Middle value   | initialize, protocolVersion 2024-11-05                                     | LegacyInitialize, requested 2024-11-05 (step 1)   |
// | 19     | Request.BindLegacy  | Failure | Boundary value | initialize without protocolVersion                                         | LegacyInitialize, requested "" (step 1)           |
// | 20     | Request.BindLegacy  | Success | Middle value   | tools/list                                                                 | nil (step 1)                                      |
// | 21     | Request.BindParams  | Success | Middle value   | Valid _meta                                                                | nil (step 2)                                      |
// | 22     | Request.BindParams  | Success | Boundary value | clientCapabilities {}                                                      | nil; an empty object is present (step 1)          |
// | 23     | Request.BindParams  | Failure | Boundary value | No params                                                                  | MissingMeta (step 1)                              |
// | 24     | Request.BindParams  | Failure | Boundary value | params {} without _meta                                                    | MissingMeta (step 1)                              |
// | 25     | Request.BindParams  | Failure | Middle value   | _meta without protocolVersion                                              | MissingMeta (step 1)                              |
// | 26     | Request.BindParams  | Failure | Boundary value | protocolVersion ""                                                         | MissingMeta (step 1)                              |
// | 27     | Request.BindParams  | Failure | Middle value   | _meta without clientCapabilities                                           | MissingMeta (step 1)                              |
// | 28     | Request.BindParams  | Failure | Middle value   | protocolVersion 2024-11-05                                                 | UnsupportedVersion, requested 2024-11-05 (step 2) |
// | 29     | Request.BindHeaders | Success | Middle value   | tools/list; both headers match; no Mcp-Name                                | nil (step 3)                                      |
// | 30     | Request.BindHeaders | Success | Middle value   | tools/call add_context; Mcp-Name add_context                               | nil (step 3)                                      |
// | 31     | Request.BindHeaders | Success | Middle value   | resources/read skill://precedent-recall; Mcp-Name skill://precedent-recall | nil (step 3)                                      |
// | 32     | Request.BindHeaders | Failure | Boundary value | No MCP-Protocol-Version                                                    | HeaderMismatch (step 1)                           |
// | 33     | Request.BindHeaders | Failure | Boundary value | No Mcp-Method                                                              | HeaderMismatch (step 1)                           |
// | 34     | Request.BindHeaders | Failure | Middle value   | MCP-Protocol-Version 2024-11-05; _meta 2026-07-28                          | HeaderMismatch (step 2)                           |
// | 35     | Request.BindHeaders | Failure | Middle value   | Mcp-Method server/discover; method tools/list                              | HeaderMismatch (step 2)                           |
// | 36     | Request.BindHeaders | Failure | Middle value   | tools/call add_context; Mcp-Name review_decision                           | HeaderMismatch (step 3)                           |
// | 37     | Request.BindHeaders | Failure | Boundary value | tools/list without name or uri; Mcp-Name x                                 | HeaderMismatch (step 3)                           |
// | 38     | Request.BindHeaders | Failure | Boundary value | tools/call add_context without Mcp-Name                                    | HeaderMismatch (step 3)                           |
// | 39     | Request.BindHeaders | Failure | Boundary value | resources/read skill://precedent-recall without Mcp-Name                   | HeaderMismatch (step 3)                           |
// | 40     | Request.BindHeaders | Success | Boundary value | skills/get skill://precedent-recall without Mcp-Name                       | nil; skills/get does not require it (step 3)      |
// | 41     | Request.BindHost    | Success | Middle value   | Local 127.0.0.1:8080; Host localhost:8080                                  | nil (step 2)                                      |
// | 42     | Request.BindHost    | Success | Boundary value | Local 127.0.0.1:8080; Host localhost                                       | nil; no port (step 2)                             |
// | 43     | Request.BindHost    | Success | Middle value   | Local 127.0.0.1:8080; Host 127.0.0.1:8080                                  | nil (step 2)                                      |
// | 44     | Request.BindHost    | Success | Boundary value | Local [::1]:8080; Host [::1]:8080                                          | nil (step 2)                                      |
// | 45     | Request.BindHost    | Success | Middle value   | Local 192.168.1.5:8080; Host evil.com:8080                                 | nil; not a loopback connection (step 1)           |
// | 46     | Request.BindHost    | Success | Boundary value | No local address; Host evil.com:8080                                       | nil (step 1)                                      |
// | 47     | Request.BindHost    | Failure | Middle value   | Local 127.0.0.1:8080; Host evil.com:8080                                   | InvalidHost "evil.com:8080" (step 3)              |
// | 48     | Request.BindHost    | Failure | Boundary value | Local 127.0.0.1:8080; Host empty                                           | InvalidHost "" (step 3)                           |
//
// Business sources: JSON-RPC 2.0 specification (id, parse error, invalid request); Go SDK v1.8.0 mcp/streamable.go SEP-2575 checks (_meta, protocol version, MCP headers, HTTP 400) and internal/util/net.go IsLoopback (Host check); memory thin-gateway (notifications are errors).
//
// Mcp-Name source: MCP specification 2026-07-28, Streamable HTTP, Standard Request Headers (Mcp-Name required for tools/call, resources/read, prompts/get) and Server Validation (a missing required header is HeaderMismatch -32020, HTTP 400).
//
// Branch coverage
// | Function            | Condition                    | True               | False        |
// |---------------------|------------------------------|--------------------|--------------|
// | Request.Bind        | Body is JSON                 | 2 to 5, 8 to 17    | 6, 7         |
// | Request.Bind        | id is a string or an integer | 2 to 5, 13 to 17   | 8 to 12      |
// | Request.Bind        | jsonrpc, method, types valid | 2 to 5             | 13 to 17     |
// | Request.BindLegacy  | method is initialize         | 18, 19             | 20           |
// | Request.BindParams  | _meta complete and valid     | 21, 22, 28         | 23 to 27     |
// | Request.BindParams  | Supported protocol version   | 21, 22             | 28           |
// | Request.BindHeaders | Both headers present         | 29 to 31, 34 to 40 | 32, 33       |
// | Request.BindHeaders | Headers equal the body       | 29 to 31, 36, 37   | 34, 35       |
// | Request.BindHeaders | Mcp-Name sent                | 30, 31, 36, 37     | 29, 38 to 40 |
// | Request.BindHeaders | tools/call or resources/read | 30, 31, 36, 38, 39 | 29, 37, 40   |
// | Request.BindHost    | Loopback connection          | 41 to 44, 47, 48   | 45, 46       |
// | Request.BindHost    | Host is loopback             | 41 to 44           | 47, 48       |
//
// Test mapping
// | Test                                                    | Scenarios |
// |---------------------------------------------------------|-----------|
// | TestToEmpty/tools list request                          | 1         |
// | TestBind/string id                                      | 2         |
// | TestBind/integer id                                     | 3         |
// | TestBind/zero id                                        | 4         |
// | TestBind/empty string id                                | 5         |
// | TestBindErrors/malformed JSON                           | 6         |
// | TestBindErrors/empty body                               | 7         |
// | TestBindErrors/batch body                               | 8         |
// | TestBindErrors/notification without id                  | 9         |
// | TestBindErrors/null id                                  | 10        |
// | TestBindErrors/fraction id                              | 11        |
// | TestBindErrors/boolean id                               | 12        |
// | TestBindErrors/jsonrpc 1.0 keeps the id                 | 13        |
// | TestBindErrors/missing jsonrpc keeps the id             | 14        |
// | TestBindErrors/missing method keeps the id              | 15        |
// | TestBindErrors/numeric method keeps the id              | 16        |
// | TestBindErrors/numeric protocol version keeps the id    | 17        |
// | TestBindLegacy/initialize with a version                | 18        |
// | TestBindLegacy/initialize without a version             | 19        |
// | TestBindLegacy/other method                             | 20        |
// | TestBindParams/valid _meta                              | 21        |
// | TestBindParams/empty client capabilities                | 22        |
// | TestBindParamsErrors/no params                          | 23        |
// | TestBindParamsErrors/no _meta                           | 24        |
// | TestBindParamsErrors/no protocol version                | 25        |
// | TestBindParamsErrors/empty protocol version             | 26        |
// | TestBindParamsErrors/no client capabilities             | 27        |
// | TestBindParamsErrors/unsupported protocol version       | 28        |
// | TestBindHeaders/headers match without Mcp-Name          | 29        |
// | TestBindHeaders/Mcp-Name equals params.name             | 30        |
// | TestBindHeaders/Mcp-Name equals params.uri              | 31        |
// | TestBindHeadersErrors/no protocol version header        | 32        |
// | TestBindHeadersErrors/no method header                  | 33        |
// | TestBindHeadersErrors/protocol version header differs   | 34        |
// | TestBindHeadersErrors/method header differs             | 35        |
// | TestBindHeadersErrors/Mcp-Name differs from params.name | 36        |
// | TestBindHeadersErrors/Mcp-Name without name or uri      | 37        |
// | TestBindHeadersErrors/tools/call without Mcp-Name       | 38        |
// | TestBindHeadersErrors/resources/read without Mcp-Name   | 39        |
// | TestBindHeaders/skills/get without Mcp-Name             | 40        |
// | TestBindHost/localhost with port                        | 41        |
// | TestBindHost/localhost without port                     | 42        |
// | TestBindHost/IPv4 loopback                              | 43        |
// | TestBindHost/IPv6 loopback                              | 44        |
// | TestBindHost/LAN connection is outside the check        | 45        |
// | TestBindHost/no local address                           | 46        |
// | TestBindHostErrors/foreign host on loopback             | 47        |
// | TestBindHostErrors/empty host on loopback               | 48        |
// | No test                                                 | None      |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"context"
	"net"
	"net/http"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`
	toolsList = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{` + validMeta + `}}`
	toolCall  = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"add_context",` + validMeta + `}}`
	readSkill = `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"skill://precedent-recall",` + validMeta + `}}`
	getSkill  = `{"jsonrpc":"2.0","id":1,"method":"skills/get","params":{"uri":"skill://precedent-recall",` + validMeta + `}}`
)

var (
	wantParseError     = &RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{Code: env.CodeParseError, Message: "parse error"}}
	wantInvalidRequest = &RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{Code: env.CodeInvalidRequest, Message: "invalid request"}}
	wantMissingMeta    = &RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{Code: env.CodeInvalidParams, Message: "missing or invalid _meta"}}
	wantHeaderMismatch = &RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{
		Code:    env.CodeHeaderMismatch,
		Message: "MCP headers missing or do not match the request body",
	}}
)

func TestToEmpty(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest Request
		want         *pb.Empty
	}{
		{"tools list request", Request{JSONRPC: string(env.VersionJSONRPC), ID: float64(1), Method: env.MethodToolsList}, &pb.Empty{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenRequest.ToEmpty()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBind(t *testing.T) {
	tests := []struct {
		name      string
		givenBody string
		want      *RPCError
		wantID    any
	}{
		{"string id", `{"jsonrpc":"2.0","id":"a","method":"tools/list"}`, nil, "a"},
		{"integer id", `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`, nil, float64(7)},
		{"zero id", `{"jsonrpc":"2.0","id":0,"method":"tools/list"}`, nil, float64(0)},
		{"empty string id", `{"jsonrpc":"2.0","id":"","method":"tools/list"}`, nil, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var r Request

			got := r.Bind(ginContext(t, test.givenBody, nil))

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantID, r.ID)
		})
	}
}

func TestBindErrors(t *testing.T) {
	tests := []struct {
		name      string
		givenBody string
		want      *RPCError
		wantID    any
	}{
		{"malformed JSON", `{"jsonrpc":`, wantParseError, nil},
		{"empty body", ``, wantParseError, nil},
		{"batch body", `[]`, wantInvalidRequest, nil},
		{"notification without id", `{"jsonrpc":"2.0","method":"tools/list"}`, wantInvalidRequest, nil},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"tools/list"}`, wantInvalidRequest, nil},
		{"fraction id", `{"jsonrpc":"2.0","id":1.5,"method":"tools/list"}`, wantInvalidRequest, nil},
		{"boolean id", `{"jsonrpc":"2.0","id":true,"method":"tools/list"}`, wantInvalidRequest, nil},
		{"jsonrpc 1.0 keeps the id", `{"jsonrpc":"1.0","id":1,"method":"tools/list"}`, wantInvalidRequest, float64(1)},
		{"missing jsonrpc keeps the id", `{"id":1,"method":"tools/list"}`, wantInvalidRequest, float64(1)},
		{"missing method keeps the id", `{"jsonrpc":"2.0","id":1}`, wantInvalidRequest, float64(1)},
		{"numeric method keeps the id", `{"jsonrpc":"2.0","id":1,"method":5}`, wantInvalidRequest, float64(1)},
		{"numeric protocol version keeps the id",
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":5,"io.modelcontextprotocol/clientCapabilities":{}}}}`,
			wantInvalidRequest, float64(1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var r Request

			got := r.Bind(ginContext(t, test.givenBody, nil))

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantID, r.ID)
		})
	}
}

func TestBindLegacy(t *testing.T) {
	tests := []struct {
		name      string
		givenBody string
		want      *RPCError
	}{
		{"initialize with a version", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			&RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{
				Code:    env.CodeUnsupportedVersion,
				Message: "initialize is not supported; use protocol version 2026-07-28",
				Data:    map[string]any{"supported": []string{"2026-07-28"}, "requested": "2024-11-05"},
			}}},
		{"initialize without a version", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
			&RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{
				Code:    env.CodeUnsupportedVersion,
				Message: "initialize is not supported; use protocol version 2026-07-28",
				Data:    map[string]any{"supported": []string{"2026-07-28"}, "requested": ""},
			}}},
		{"other method", toolsList, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, test.givenBody, nil)
			var r Request
			require.Nil(t, r.Bind(c))

			got := r.BindLegacy(c)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBindParams(t *testing.T) {
	tests := []struct {
		name      string
		givenBody string
		want      *RPCError
	}{
		{"valid _meta", toolsList, nil},
		{"empty client capabilities",
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, test.givenBody, nil)
			var r Request
			require.Nil(t, r.Bind(c))

			got := r.BindParams(c)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBindParamsErrors(t *testing.T) {
	tests := []struct {
		name      string
		givenBody string
		want      *RPCError
	}{
		{"no params", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, wantMissingMeta},
		{"no _meta", `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`, wantMissingMeta},
		{"no protocol version",
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{}}}}`, wantMissingMeta},
		{"empty protocol version",
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"","io.modelcontextprotocol/clientCapabilities":{}}}}`, wantMissingMeta},
		{"no client capabilities",
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, wantMissingMeta},
		{"unsupported protocol version",
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2024-11-05","io.modelcontextprotocol/clientCapabilities":{}}}}`,
			&RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{
				Code:    env.CodeUnsupportedVersion,
				Message: "unsupported protocol version; use 2026-07-28",
				Data:    map[string]any{"supported": []string{"2026-07-28"}, "requested": "2024-11-05"},
			}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, test.givenBody, nil)
			var r Request
			require.Nil(t, r.Bind(c))

			got := r.BindParams(c)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBindHeaders(t *testing.T) {
	tests := []struct {
		name         string
		givenBody    string
		givenHeaders map[string]string
		want         *RPCError
	}{
		{"headers match without Mcp-Name", toolsList, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodToolsList),
		}, nil},
		{"Mcp-Name equals params.name", toolCall, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodToolsCall),
			string(env.HeaderName):            "add_context",
		}, nil},
		{"Mcp-Name equals params.uri", readSkill, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodResourcesRead),
			string(env.HeaderName):            "skill://precedent-recall",
		}, nil},
		{"skills/get without Mcp-Name", getSkill, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodSkillsGet),
		}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, test.givenBody, test.givenHeaders)
			var r Request
			require.Nil(t, r.Bind(c))

			got := r.BindHeaders(c)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBindHeadersErrors(t *testing.T) {
	tests := []struct {
		name         string
		givenBody    string
		givenHeaders map[string]string
		want         *RPCError
	}{
		{"no protocol version header", toolsList, map[string]string{
			string(env.HeaderMethod): string(env.MethodToolsList),
		}, wantHeaderMismatch},
		{"no method header", toolsList, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
		}, wantHeaderMismatch},
		{"protocol version header differs", toolsList, map[string]string{
			string(env.HeaderProtocolVersion): "2024-11-05",
			string(env.HeaderMethod):          string(env.MethodToolsList),
		}, wantHeaderMismatch},
		{"method header differs", toolsList, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodDiscover),
		}, wantHeaderMismatch},
		{"Mcp-Name differs from params.name", toolCall, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodToolsCall),
			string(env.HeaderName):            "review_decision",
		}, wantHeaderMismatch},
		{"Mcp-Name without name or uri", toolsList, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodToolsList),
			string(env.HeaderName):            "x",
		}, wantHeaderMismatch},
		{"tools/call without Mcp-Name", toolCall, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodToolsCall),
		}, wantHeaderMismatch},
		{"resources/read without Mcp-Name", readSkill, map[string]string{
			string(env.HeaderProtocolVersion): string(env.VersionProtocol),
			string(env.HeaderMethod):          string(env.MethodResourcesRead),
		}, wantHeaderMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, test.givenBody, test.givenHeaders)
			var r Request
			require.Nil(t, r.Bind(c))

			got := r.BindHeaders(c)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBindHost(t *testing.T) {
	tests := []struct {
		name           string
		givenLocalAddr net.Addr
		givenHost      string
		want           *RPCError
	}{
		{"localhost with port", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}, "localhost:8080", nil},
		{"localhost without port", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}, "localhost", nil},
		{"IPv4 loopback", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}, "127.0.0.1:8080", nil},
		{"IPv6 loopback", &net.TCPAddr{IP: net.IPv6loopback, Port: 8080}, "[::1]:8080", nil},
		{"LAN connection is outside the check", &net.TCPAddr{IP: net.IPv4(192, 168, 1, 5), Port: 8080}, "evil.com:8080", nil},
		{"no local address", nil, "evil.com:8080", nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, toolsList, nil)
			c.Request.Host = test.givenHost
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), http.LocalAddrContextKey, test.givenLocalAddr))
			var r Request

			got := r.BindHost(c)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestBindHostErrors(t *testing.T) {
	tests := []struct {
		name           string
		givenLocalAddr net.Addr
		givenHost      string
		want           *RPCError
	}{
		{"foreign host on loopback", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}, "evil.com:8080",
			&RPCError{HTTPStatus: http.StatusForbidden, Error: Error{Code: env.CodeInvalidRequest, Message: `invalid Host header "evil.com:8080"`}}},
		{"empty host on loopback", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}, "",
			&RPCError{HTTPStatus: http.StatusForbidden, Error: Error{Code: env.CodeInvalidRequest, Message: `invalid Host header ""`}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := ginContext(t, toolsList, nil)
			c.Request.Host = test.givenHost
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), http.LocalAddrContextKey, test.givenLocalAddr))
			var r Request

			got := r.BindHost(c)

			assert.Equal(t, test.want, got)
		})
	}
}
