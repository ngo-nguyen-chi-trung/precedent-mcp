package model

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Error struct {
	Code    env.RPCCode `json:"code"`
	Message string      `json:"message"`
	Data    any         `json:"data,omitempty"`
}

type RPCError struct {
	HTTPStatus int
	Error
}

func InvalidHost(host string) *RPCError {
	return newRPCError(http.StatusForbidden, env.CodeInvalidRequest, fmt.Sprintf("invalid Host header %q", host))
}

func ParseError() *RPCError {
	return newRPCError(http.StatusBadRequest, env.CodeParseError, "parse error")
}

func InvalidRequest() *RPCError {
	return newRPCError(http.StatusBadRequest, env.CodeInvalidRequest, "invalid request")
}

func MethodNotFound(method env.RPCMethod) *RPCError {
	return newRPCError(http.StatusNotFound, env.CodeMethodNotFound, fmt.Sprintf("method not found: %s", method))
}

func MissingMeta() *RPCError {
	return newRPCError(http.StatusBadRequest, env.CodeInvalidParams, "missing or invalid _meta")
}

func UnsupportedVersion(requested string) *RPCError {
	e := newRPCError(http.StatusBadRequest, env.CodeUnsupportedVersion, fmt.Sprintf("unsupported protocol version; use %s", env.VersionProtocol))
	e.Data = map[string]any{"supported": []string{string(env.VersionProtocol)}, "requested": requested}
	return e
}

func LegacyInitialize(requested string) *RPCError {
	e := UnsupportedVersion(requested)
	e.Message = fmt.Sprintf("%s is not supported; use protocol version %s", env.MethodInitialize, env.VersionProtocol)
	return e
}

func HeaderMismatch() *RPCError {
	return newRPCError(http.StatusBadRequest, env.CodeHeaderMismatch, "MCP headers missing or do not match the request body")
}

func InternalError() *RPCError {
	return newRPCError(http.StatusInternalServerError, env.CodeInternal, "internal error")
}

func FromStatus(err error) *RPCError {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if d, ok := detail.(*pb.RPCError); ok && st.Code() == codes.InvalidArgument {
			return FromInvalidArgument(d)
		}
	}
	log.Printf("internal error: %v", err)
	return InternalError()
}

func FromInvalidArgument(d *pb.RPCError) *RPCError {
	e := newRPCError(http.StatusBadRequest, env.RPCCode(d.GetCode()), d.GetMessage())
	if len(d.GetData()) > 0 {
		e.Data = json.RawMessage(d.GetData())
	}
	return e
}

func newRPCError(status int, code env.RPCCode, message string) *RPCError {
	return &RPCError{HTTPStatus: status, Error: Error{Code: code, Message: message}}
}
