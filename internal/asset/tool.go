package asset

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/asset/validator"
	"github.com/ngo/precedent_mcp/internal/env"
	"gopkg.in/yaml.v3"
)

type ToolStore struct {
	dir   string
	pipe  validator.Pipeline[[]model.ToolEntry]
	tools *expirable.LRU[string, []model.ToolEntry]
}

func NewToolStore(directory string) *ToolStore {
	return &ToolStore{
		dir:   directory,
		pipe:  validator.NewBuilder[[]model.ToolEntry]().Add(validator.InputSchemas()).Build(),
		tools: expirable.NewLRU[string, []model.ToolEntry](int(env.CacheSizeTools), nil, env.DurationCache),
	}
}

func (s *ToolStore) Tools() ([]model.ToolEntry, error) {
	if list, ok := s.tools.Get(string(env.PathToolList)); ok {
		return list, nil
	}
	list, err := loadToolList(filepath.Join(s.dir, string(env.PathToolList)))
	if err != nil {
		return nil, err
	}
	if err := s.pipe.Run(list); err != nil {
		return nil, err
	}
	s.tools.Add(string(env.PathToolList), list)
	return list, nil
}

func loadToolList(path string) ([]model.ToolEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file model.ToolList
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.Tools, nil
}
