package model

import pb "github.com/ngo/precedent_mcp/gen/proto/v1"

type SkillFrontmatter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SkillResource struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type SkillEntry struct {
	URI         string           `json:"uri"`
	Frontmatter SkillFrontmatter `json:"frontmatter"`
	Resources   []SkillResource  `json:"resources"`
}

type SkillsListResult struct {
	ResultBase
	CacheHints
	Skills []SkillEntry `json:"skills"`
}

func (r *Response) FromSkillListResponse(res *pb.SkillListResponse) {
	skills := make([]SkillEntry, len(res.GetSkills()))
	for i, skill := range res.GetSkills() {
		skills[i].FromSkillEntry(skill)
	}
	r.Result = SkillsListResult{ResultBase: complete, CacheHints: publicCache, Skills: skills}
}

type SkillGetResult struct {
	ResultBase
	CacheHints
	Skill SkillEntry `json:"skill"`
}

func (r *Response) FromSkillGetResponse(res *pb.SkillGetResponse) {
	result := SkillGetResult{ResultBase: complete, CacheHints: publicCache}
	result.Skill.FromSkillEntry(res.GetSkill())
	r.Result = result
}

func (e *SkillEntry) FromSkillEntry(s *pb.SkillEntry) {
	e.URI = s.GetUri()
	e.Frontmatter = SkillFrontmatter{Name: s.GetFrontmatter().GetName(), Description: s.GetFrontmatter().GetDescription()}
	e.Resources = make([]SkillResource, 0, len(s.GetResources()))
	for _, resource := range s.GetResources() {
		e.Resources = append(e.Resources, SkillResource{URI: resource.GetUri(), Digest: resource.GetDigest(), Size: resource.GetSize()})
	}
}
