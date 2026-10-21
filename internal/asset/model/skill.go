package model

import "github.com/ngo/precedent_mcp/internal/env"

type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Digest      string `yaml:"digest"`
	Size        int64  `yaml:"size"`
}

type SkillPaths struct {
	Skill    env.AssetPath
	Metadata env.AssetPath
}
