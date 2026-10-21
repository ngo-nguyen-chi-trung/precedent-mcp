package asset

import (
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
)

var Examples = map[env.SkillName]env.AssetPath{
	env.SkillPrecedentCapture: env.PathPrecedentCaptureExamples,
	env.SkillPrecedentDecide:  env.PathPrecedentDecideExamples,
	env.SkillPrecedentRecall:  env.PathPrecedentRecallExamples,
	env.SkillPrecedentRefine:  env.PathPrecedentRefineExamples,
	env.SkillPrecedentReview:  env.PathPrecedentReviewExamples,
}

type ExampleStore struct {
	dir      string
	examples map[env.SkillName]env.AssetPath
}

func NewExampleStore(directory string, examples map[env.SkillName]env.AssetPath) *ExampleStore {
	return &ExampleStore{dir: directory, examples: examples}
}

func (s *ExampleStore) ListNames() []env.SkillName {
	return slices.Sorted(maps.Keys(s.examples))
}

func (s *ExampleStore) Examples(name string) (string, error) {
	path, ok := s.examples[env.SkillName(name)]
	if !ok {
		return "", model.ErrExamplesNotFound(name)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, string(path)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
