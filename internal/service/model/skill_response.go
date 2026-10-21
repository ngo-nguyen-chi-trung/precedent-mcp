package model

import (
	"net/url"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
)

type SkillListResponse struct {
	*pb.SkillListResponse
}

func (r *SkillListResponse) FromMetadata(mds []assetmodel.Metadata) {
	r.SkillListResponse = &pb.SkillListResponse{Skills: make([]*pb.SkillEntry, 0, len(mds))}
	for _, md := range mds {
		var entry SkillEntry
		entry.FromMetadata(md)
		r.Skills = append(r.Skills, entry.SkillEntry)
	}
}

type SkillGetResponse struct {
	*pb.SkillGetResponse
}

func (r *SkillGetResponse) FromMetadata(md assetmodel.Metadata) {
	var entry SkillEntry
	entry.FromMetadata(md)
	r.SkillGetResponse = &pb.SkillGetResponse{Skill: entry.SkillEntry}
}

type SkillEntry struct {
	*pb.SkillEntry
}

func (e *SkillEntry) FromMetadata(md assetmodel.Metadata) {
	uri := (&url.URL{Scheme: string(env.SchemeSkill), Host: md.Name}).String()
	e.SkillEntry = &pb.SkillEntry{
		Uri:         uri,
		Frontmatter: &pb.SkillFrontmatter{Name: md.Name, Description: md.Description},
		Resources:   []*pb.SkillResource{{Uri: uri, Digest: md.Digest, Size: md.Size}},
	}
}
