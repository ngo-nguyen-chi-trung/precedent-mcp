// Test plan: response.go
//
// Logic flow
// | Function           | Step | Action                                                | Effect                                     |
// |--------------------|------|-------------------------------------------------------|--------------------------------------------|
// | NewResponse        | 1    | jsonrpc "2.0"; no id yet                              | id null until SetID                        |
// | Response.SetID     | 1    | Echo the request id                                   | String or number kept; nil written as null |
// | Response.FromError | 1    | Error object with code, message and data when present | No result key                              |
//
// Scenarios
// | Number | Function           | Result  | Value class    | Input                               | Expected                                                                    |
// |--------|--------------------|---------|----------------|-------------------------------------|-----------------------------------------------------------------------------|
// | 1      | Response.SetID     | Success | Middle value   | id "a"                              | {"jsonrpc":"2.0","id":"a"} (NewResponse step 1, SetID step 1)               |
// | 2      | Response.SetID     | Success | Middle value   | id 7                                | {"jsonrpc":"2.0","id":7}                                                    |
// | 3      | Response.SetID     | Success | Boundary value | id 0                                | {"jsonrpc":"2.0","id":0}; zero kept                                         |
// | 4      | Response.SetID     | Success | Boundary value | id nil (invalid request id)         | {"jsonrpc":"2.0","id":null}                                                 |
// | 5      | Response.FromError | Failure | Middle value   | ParseError; id nil                  | {"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}} |
// | 6      | Response.FromError | Failure | Middle value   | UnsupportedVersion 2024-11-05; id 1 | error with data supported and requested; no result key                      |
//
// Branch coverage
// | Function           | Condition    | True | False |
// |--------------------|--------------|------|-------|
// | Response.FromError | Data present | 6    | 5     |
//
// Test mapping
// | Test                                     | Scenarios                                                    |
// |------------------------------------------|--------------------------------------------------------------|
// | TestResponseSetID/string id              | 1                                                            |
// | TestResponseSetID/integer id             | 2                                                            |
// | TestResponseSetID/zero id                | 3                                                            |
// | TestResponseSetID/nil id                 | 4                                                            |
// | TestResponseFromError/error without data | 5                                                            |
// | TestResponseFromError/error with data    | 6                                                            |
// | No test                                  | complete and publicCache: proven by the From… response tests |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseSetID(t *testing.T) {
	tests := []struct {
		name    string
		givenID any
		want    string
	}{
		{"string id", "a", `{"jsonrpc":"2.0","id":"a"}`},
		{"integer id", float64(7), `{"jsonrpc":"2.0","id":7}`},
		{"zero id", float64(0), `{"jsonrpc":"2.0","id":0}`},
		{"nil id", nil, `{"jsonrpc":"2.0","id":null}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()

			res.SetID(test.givenID)

			got, err := json.Marshal(res)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}

func TestResponseFromError(t *testing.T) {
	tests := []struct {
		name       string
		givenID    any
		givenError *RPCError
		want       string
	}{
		{"error without data", nil, ParseError(),
			`{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`},
		{"error with data", float64(1), UnsupportedVersion("2024-11-05"),
			`{"jsonrpc":"2.0","id":1,"error":{"code":-32022,"message":"unsupported protocol version; use 2026-07-28",` +
				`"data":{"supported":["2026-07-28"],"requested":"2024-11-05"}}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := NewResponse()
			res.SetID(test.givenID)

			res.FromError(test.givenError)

			got, err := json.Marshal(res)
			require.NoError(t, err)
			assert.JSONEq(t, test.want, string(got))
		})
	}
}
