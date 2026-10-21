package validator

import (
	"unicode/utf8"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
)

func MatchesDirectory(directory string) Step[model.Metadata] {
	return func(md model.Metadata) error {
		if md.Name != directory {
			return model.ErrSkillNameMismatch(md.Name, directory)
		}
		return nil
	}
}

func DescriptionLength() Step[model.Metadata] {
	return func(md model.Metadata) error {
		if n := utf8.RuneCountInString(md.Description); n == 0 || n > int(env.LimitSkillDescription) {
			return model.ErrDescriptionLength(md.Name, n)
		}
		return nil
	}
}
