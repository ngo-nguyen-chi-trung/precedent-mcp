package model

import pb "github.com/ngo/precedent_mcp/gen/proto/v1"

func (r *Request) ToToolCallRequest() *pb.ToolCallRequest {
	return &pb.ToolCallRequest{Name: r.Params.Name, Arguments: r.Params.Arguments}
}
