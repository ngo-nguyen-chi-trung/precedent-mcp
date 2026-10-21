package model

import pb "github.com/ngo/precedent_mcp/gen/proto/v1"

func (r *Request) ToSkillGetRequest() *pb.SkillGetRequest {
	return &pb.SkillGetRequest{Uri: r.Params.URI}
}
