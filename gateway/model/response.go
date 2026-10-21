package model

import "github.com/ngo/precedent_mcp/internal/env"

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

func NewResponse() *Response {
	return &Response{JSONRPC: string(env.VersionJSONRPC)}
}

func (r *Response) SetID(id any) {
	r.ID = id
}

func (r *Response) FromError(e *RPCError) {
	r.Error = &e.Error
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ResultMeta struct {
	ServerInfo ServerInfo `json:"io.modelcontextprotocol/serverInfo"`
}

type ResultBase struct {
	ResultType env.ResultType `json:"resultType"`
	Meta       ResultMeta     `json:"_meta"`
}

type CacheHints struct {
	TTLMs      int64          `json:"ttlMs"`
	CacheScope env.CacheScope `json:"cacheScope"`
}

var complete = ResultBase{
	ResultType: env.ResultComplete,
	Meta:       ResultMeta{ServerInfo: ServerInfo{Name: string(env.ServerInfoName), Version: string(env.ServerInfoVersion)}},
}

var publicCache = CacheHints{TTLMs: env.DurationCache.Milliseconds(), CacheScope: env.CachePublic}
