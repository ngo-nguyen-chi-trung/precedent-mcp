package model

import (
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type ToolCallRequest struct {
	*pb.ToolCallRequest
	Schema *jsonschema.Schema
}
