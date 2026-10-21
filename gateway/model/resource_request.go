package model

import pb "github.com/ngo/precedent_mcp/gen/proto/v1"

func (r *Request) ToResourceReadRequest() *pb.ResourceReadRequest {
	return &pb.ResourceReadRequest{Uri: r.Params.URI}
}
