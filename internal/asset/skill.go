package asset

import (
	"bytes"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/asset/validator"
	"github.com/ngo/precedent_mcp/internal/env"
	"gopkg.in/yaml.v3"
)

var Skills = map[env.SkillName]model.SkillPaths{
	env.SkillPrecedentCapture: {Skill: env.PathPrecedentCaptureSkill, Metadata: env.PathPrecedentCaptureMetadata},
	env.SkillPrecedentDecide:  {Skill: env.PathPrecedentDecideSkill, Metadata: env.PathPrecedentDecideMetadata},
	env.SkillPrecedentRecall:  {Skill: env.PathPrecedentRecallSkill, Metadata: env.PathPrecedentRecallMetadata},
	env.SkillPrecedentRefine:  {Skill: env.PathPrecedentRefineSkill, Metadata: env.PathPrecedentRefineMetadata},
	env.SkillPrecedentReview:  {Skill: env.PathPrecedentReviewSkill, Metadata: env.PathPrecedentReviewMetadata},
}

type SkillStore struct {
	dir        string
	skills     map[env.SkillName]model.SkillPaths
	metadata   *expirable.LRU[string, model.Metadata]
	skillTexts *expirable.LRU[string, string]
}

func NewSkillStore(directory string, skills map[env.SkillName]model.SkillPaths) *SkillStore {
	return &SkillStore{
		dir:        directory,
		skills:     skills,
		metadata:   expirable.NewLRU[string, model.Metadata](int(env.CacheSizeSkills), nil, env.DurationCache),
		skillTexts: expirable.NewLRU[string, string](int(env.CacheSizeSkills), nil, env.DurationCache),
	}
}

func (s *SkillStore) ListMetadata() []model.Metadata {
	names := slices.Sorted(maps.Keys(s.skills))
	list := make([]model.Metadata, 0, len(names))
	for _, name := range names {
		md, err := s.Metadata(string(name))
		if err != nil {
			log.Printf("skip skill %s: %v", name, err)
			continue
		}
		list = append(list, md)
	}
	return list
}

func (s *SkillStore) Skill(name string) (string, error) {
	if text, ok := s.skillTexts.Get(name); ok {
		return text, nil
	}
	paths, ok := s.skills[env.SkillName(name)]
	if !ok {
		return "", model.ErrSkillNotFound(name)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, string(paths.Skill)))
	if err != nil {
		return "", err
	}
	text := string(data)
	s.skillTexts.Add(name, text)
	return text, nil
}

func (s *SkillStore) Metadata(name string) (model.Metadata, error) {
	if md, ok := s.metadata.Get(name); ok {
		return md, nil
	}
	paths, ok := s.skills[env.SkillName(name)]
	if !ok {
		return model.Metadata{}, model.ErrSkillNotFound(name)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, string(paths.Metadata)))
	if err != nil {
		return model.Metadata{}, err
	}
	var md model.Metadata
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&md); err != nil {
		return model.Metadata{}, model.ErrInvalidMetadata(name, err)
	}
	pipe := validator.NewBuilder[model.Metadata]().
		Add(validator.MatchesDirectory(name)).
		Add(validator.DescriptionLength()).
		Build()
	if err := pipe.Run(md); err != nil {
		return model.Metadata{}, err
	}
	s.metadata.Add(name, md)
	return md, nil
}
