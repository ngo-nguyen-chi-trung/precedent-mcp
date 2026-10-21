package model

import (
	"cmp"
	"encoding/json"
	"errors"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
)

type Request struct {
	JSONRPC    string        `json:"jsonrpc" binding:"required"`
	ID         any           `json:"id"`
	Method     env.RPCMethod `json:"method" binding:"required"`
	ParamsBody `binding:"-"`
	Headers    Headers `json:"-" binding:"-"`
}

type ParamsBody struct {
	Params Params `json:"params"`
}

type Params struct {
	Name            string          `json:"name"`
	Arguments       json.RawMessage `json:"arguments"`
	URI             string          `json:"uri"`
	ProtocolVersion string          `json:"protocolVersion"`
	Meta            Meta            `json:"_meta"`
}

type Meta struct {
	ProtocolVersion    string         `json:"io.modelcontextprotocol/protocolVersion" binding:"required"`
	ClientCapabilities map[string]any `json:"io.modelcontextprotocol/clientCapabilities" binding:"required"`
}

type Headers struct {
	ProtocolVersion string        `header:"MCP-Protocol-Version" binding:"required"`
	Method          env.RPCMethod `header:"Mcp-Method" binding:"required"`
	Name            string        `header:"Mcp-Name"`
}

func (r *Request) ToEmpty() *pb.Empty {
	return &pb.Empty{}
}

func (r *Request) BindHost(c *gin.Context) *RPCError {
	local, ok := c.Request.Context().Value(http.LocalAddrContextKey).(*net.TCPAddr)
	if !ok || !local.IP.IsLoopback() {
		return nil
	}
	host := (&url.URL{Host: c.Request.Host}).Hostname()
	ip, err := netip.ParseAddr(host)
	if host == string(env.HostLoopback) || (err == nil && ip.IsLoopback()) {
		return nil
	}
	return InvalidHost(c.Request.Host)
}

func (r *Request) Bind(c *gin.Context) *RPCError {
	err := c.ShouldBindBodyWithJSON(r)
	var typeErr *json.UnmarshalTypeError
	var fieldErrs validator.ValidationErrors
	if err != nil && !errors.As(err, &typeErr) && !errors.As(err, &fieldErrs) {
		return ParseError()
	}
	_, isString := r.ID.(string)
	number, isNumber := r.ID.(float64)
	if !isString && !(isNumber && number == math.Trunc(number)) {
		r.ID = nil
		return InvalidRequest()
	}
	if err != nil || r.JSONRPC != string(env.VersionJSONRPC) {
		return InvalidRequest()
	}
	return nil
}

func (r *Request) BindLegacy(_ *gin.Context) *RPCError {
	if r.Method == env.MethodInitialize {
		return LegacyInitialize(r.Params.ProtocolVersion)
	}
	return nil
}

func (r *Request) BindParams(c *gin.Context) *RPCError {
	if err := c.ShouldBindBodyWithJSON(&r.ParamsBody); err != nil {
		return MissingMeta()
	}
	if r.Params.Meta.ProtocolVersion != string(env.VersionProtocol) {
		return UnsupportedVersion(r.Params.Meta.ProtocolVersion)
	}
	return nil
}

func (r *Request) BindHeaders(c *gin.Context) *RPCError {
	if err := c.ShouldBindHeader(&r.Headers); err != nil {
		return HeaderMismatch()
	}
	if r.Headers.ProtocolVersion != r.Params.Meta.ProtocolVersion || r.Headers.Method != r.Method {
		return HeaderMismatch()
	}
	if r.Headers.Name == "" && (r.Method == env.MethodToolsCall || r.Method == env.MethodResourcesRead) {
		return HeaderMismatch()
	}
	if r.Headers.Name != "" && r.Headers.Name != cmp.Or(r.Params.Name, r.Params.URI) {
		return HeaderMismatch()
	}
	return nil
}
