// Test plan: error.go
//
// Logic flow
// | Function            | Step | Action                                                        | Effect                                                                           |
// |---------------------|------|---------------------------------------------------------------|----------------------------------------------------------------------------------|
// | FromStatus          | 1    | Read the gRPC status of the relay error                       | A plain error counts as Unknown                                                  |
// | FromStatus          | 2    | InvalidArgument with an RPCError detail → FromInvalidArgument | HTTP 400 with the relay's code, message and data                                 |
// | FromStatus          | 3    | Any other error                                               | Log "internal error: <cause>"; InternalError -32603, HTTP 500; cause not exposed |
// | FromInvalidArgument | 1    | Copy code and message; data as raw JSON when present          | Empty data: no data key                                                          |
//
// Scenarios
// | Number | Function   | Result  | Value class    | Input                                                                              | Expected                                                        |
// |--------|------------|---------|----------------|------------------------------------------------------------------------------------|-----------------------------------------------------------------|
// | 1      | FromStatus | Failure | Middle value   | InvalidArgument; RPCError -32602 "resource not found", data {"uri":"skill://nope"} | 400; -32602; message; data as raw JSON; nothing logged (step 2) |
// | 2      | FromStatus | Failure | Boundary value | InvalidArgument; RPCError -32602 "unknown tool: nope" without data                 | 400; -32602; no data (step 2)                                   |
// | 3      | FromStatus | Failure | Middle value   | InvalidArgument "bad" without detail                                               | 500; -32603 "internal error"; cause logged (step 3)             |
// | 4      | FromStatus | Failure | Middle value   | Unavailable "conn refused" with an RPCError detail                                 | 500; -32603; the code decides, not the detail (step 3)          |
// | 5      | FromStatus | Failure | Middle value   | Unknown "open tools.yaml: no such file or directory"                               | 500; -32603; cause logged, not exposed (step 3)                 |
// | 6      | FromStatus | Failure | Boundary value | errors.New("boom"); not a gRPC status                                              | 500; -32603; log holds boom (steps 1, 3)                        |
//
// Branch coverage
// | Function            | Condition                               | True | False  |
// |---------------------|-----------------------------------------|------|--------|
// | FromStatus          | InvalidArgument with an RPCError detail | 1, 2 | 3 to 6 |
// | FromInvalidArgument | Data present                            | 1    | 2      |
//
// Test mapping
// | Test                                           | Scenarios                                                                                                                                                                                                                                               |
// |------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | TestFromStatus/invalid argument with data      | 1                                                                                                                                                                                                                                                       |
// | TestFromStatus/invalid argument without data   | 2                                                                                                                                                                                                                                                       |
// | TestFromStatus/invalid argument without detail | 3                                                                                                                                                                                                                                                       |
// | TestFromStatus/unavailable with detail         | 4                                                                                                                                                                                                                                                       |
// | TestFromStatus/relay store error               | 5                                                                                                                                                                                                                                                       |
// | TestFromStatusPlainError/plain error           | 6                                                                                                                                                                                                                                                       |
// | No test                                        | Error constructors InvalidHost, ParseError, InvalidRequest, MethodNotFound, MissingMeta, UnsupportedVersion, LegacyInitialize, HeaderMismatch, InternalError: proven by request_test.go and rest_test.go; FromInvalidArgument: proven by scenarios 1, 2 |

package model_test

import . "github.com/ngo/precedent_mcp/gateway/model"

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"testing"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
)

var wantInternalError = &RPCError{HTTPStatus: http.StatusInternalServerError, Error: Error{Code: env.CodeInternal, Message: "internal error"}}

func TestFromStatus(t *testing.T) {
	tests := []struct {
		name         string
		givenCode    codes.Code
		givenMessage string
		givenDetails []protoadapt.MessageV1
		want         *RPCError
		wantLog      string
	}{
		{
			name:         "invalid argument with data",
			givenCode:    codes.InvalidArgument,
			givenMessage: "resource not found",
			givenDetails: []protoadapt.MessageV1{&pb.RPCError{
				Code:    int32(env.CodeInvalidParams),
				Message: "resource not found",
				Data:    []byte(`{"uri":"skill://nope"}`),
			}},
			want: &RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{
				Code:    env.CodeInvalidParams,
				Message: "resource not found",
				Data:    json.RawMessage(`{"uri":"skill://nope"}`),
			}},
			wantLog: "",
		},
		{
			name:         "invalid argument without data",
			givenCode:    codes.InvalidArgument,
			givenMessage: "unknown tool: nope",
			givenDetails: []protoadapt.MessageV1{&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: nope"}},
			want:         &RPCError{HTTPStatus: http.StatusBadRequest, Error: Error{Code: env.CodeInvalidParams, Message: "unknown tool: nope"}},
			wantLog:      "",
		},
		{
			name:         "invalid argument without detail",
			givenCode:    codes.InvalidArgument,
			givenMessage: "bad",
			want:         wantInternalError,
			wantLog:      "internal error: rpc error: code = InvalidArgument desc = bad\n",
		},
		{
			name:         "unavailable with detail",
			givenCode:    codes.Unavailable,
			givenMessage: "conn refused",
			givenDetails: []protoadapt.MessageV1{&pb.RPCError{Code: int32(env.CodeInvalidParams), Message: "unknown tool: nope"}},
			want:         wantInternalError,
			wantLog:      "internal error: rpc error: code = Unavailable desc = conn refused\n",
		},
		{
			name:         "relay store error",
			givenCode:    codes.Unknown,
			givenMessage: "open tools.yaml: no such file or directory",
			want:         wantInternalError,
			wantLog:      "internal error: rpc error: code = Unknown desc = open tools.yaml: no such file or directory\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logged bytes.Buffer
			previousWriter, previousFlags := log.Writer(), log.Flags()
			log.SetOutput(&logged)
			log.SetFlags(0)
			t.Cleanup(func() {
				log.SetOutput(previousWriter)
				log.SetFlags(previousFlags)
			})
			st, err := status.New(test.givenCode, test.givenMessage).WithDetails(test.givenDetails...)
			require.NoError(t, err)

			got := FromStatus(st.Err())

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantLog, logged.String())
		})
	}
}

func TestFromStatusPlainError(t *testing.T) {
	tests := []struct {
		name       string
		givenError error
		want       *RPCError
		wantLog    string
	}{
		{"plain error", errors.New("boom"), wantInternalError, "internal error: boom\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logged bytes.Buffer
			previousWriter, previousFlags := log.Writer(), log.Flags()
			log.SetOutput(&logged)
			log.SetFlags(0)
			t.Cleanup(func() {
				log.SetOutput(previousWriter)
				log.SetFlags(previousFlags)
			})

			got := FromStatus(test.givenError)

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantLog, logged.String())
		})
	}
}
