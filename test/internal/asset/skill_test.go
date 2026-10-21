// Test plan: skill.go
//
// Logic flow
// | Function                | Step | Action                                                                   | Effect                                           |
// |-------------------------|------|--------------------------------------------------------------------------|--------------------------------------------------|
// | SkillStore.ListMetadata | 1    | Visit the skill names in sorted order                                    | Same order on every call                         |
// | SkillStore.ListMetadata | 2    | SkillStore.Metadata for each name                                        | Valid skills listed                              |
// | SkillStore.ListMetadata | 3    | Leave out a skill whose metadata fails; log "skip skill <name>: <cause>" | One broken skill never fails the list            |
// | SkillStore.Metadata     | 1    | Return the cached metadata when present                                  | No file read until DurationCache ends            |
// | SkillStore.Metadata     | 2    | Accept only names of the skill map                                       | Other names, paths included → skill_not_found    |
// | SkillStore.Metadata     | 3    | Read the metadata file of the skill                                      | Read failure → the file system error             |
// | SkillStore.Metadata     | 4    | Decode name, description, digest, size; other keys rejected              | Decode failure → invalid_metadata with the cause |
// | SkillStore.Metadata     | 5    | validator.MatchesDirectory, then validator.DescriptionLength             | First failing check → its error                  |
// | SkillStore.Metadata     | 6    | Cache and return the metadata                                            | A failure is never cached                        |
// | SkillStore.Skill        | 1    | Return the cached text when present                                      | No file read until DurationCache ends            |
// | SkillStore.Skill        | 2    | Accept only names of the skill map                                       | Other names, paths included → skill_not_found    |
// | SkillStore.Skill        | 3    | Read SKILL.md of the skill; cache it                                     | Read failure → the file system error             |
//
// Scenarios
// | Number | Function                | Result  | Value class    | Input                                                                 | Expected                                                              |
// |--------|-------------------------|---------|----------------|-----------------------------------------------------------------------|-----------------------------------------------------------------------|
// | 1      | SkillStore.ListMetadata | Success | Middle value   | sample plus long-description, name-mismatch, no-metadata, unknown-key | [sampleMetadata]; the four causes logged in name order (steps 1 to 3) |
// | 2      | SkillStore.ListMetadata | Success | Boundary value | Empty skill map                                                       | Empty list (step 1)                                                   |
// | 3      | SkillStore.Metadata     | Success | Middle value   | sample                                                                | sampleMetadata; digest and size of its SKILL.md (steps 2 to 6)        |
// | 4      | SkillStore.Metadata     | Success | Boundary value | Read; metadata.yaml removed; read again                               | Second read returns the first metadata (step 1)                       |
// | 5      | SkillStore.Metadata     | Success | Middle value   | First read fails: name mismatch; file fixed; read again               | Second read returns the fixed metadata (step 6)                       |
// | 6      | SkillStore.Metadata     | Failure | Middle value   | nope; not a skill name                                                | skill_not_found for nope (step 2)                                     |
// | 7      | SkillStore.Metadata     | Failure | Boundary value | ../sample; a path, not a skill name                                   | skill_not_found for ../sample (step 2)                                |
// | 8      | SkillStore.Metadata     | Failure | Boundary value | no-metadata; metadata.yaml missing                                    | "open <path>: no such file or directory" (step 3)                     |
// | 9      | SkillStore.Metadata     | Failure | Boundary value | Empty metadata.yaml                                                   | invalid_metadata "invalid metadata: EOF" (step 4)                     |
// | 10     | SkillStore.Metadata     | Failure | Middle value   | unknown-key; metadata.yaml holds license                              | invalid_metadata naming field license (step 4)                        |
// | 11     | SkillStore.Metadata     | Failure | Middle value   | name-mismatch; name other-name                                        | skill_name_mismatch for other-name in name-mismatch (step 5)          |
// | 12     | SkillStore.Metadata     | Failure | Middle value   | long-description; description of 57 characters                        | description_length 57 (step 5)                                        |
// | 13     | SkillStore.Skill        | Success | Middle value   | sample                                                                | Text of its SKILL.md (steps 2, 3)                                     |
// | 14     | SkillStore.Skill        | Success | Boundary value | Read; SKILL.md removed; read again                                    | Second read returns the first text (step 1)                           |
// | 15     | SkillStore.Skill        | Failure | Middle value   | nope; not a skill name                                                | skill_not_found for nope (step 2)                                     |
// | 16     | SkillStore.Skill        | Failure | Boundary value | ../sample; a path, not a skill name                                   | skill_not_found for ../sample (step 2)                                |
// | 17     | SkillStore.Skill        | Failure | Boundary value | long-description; SKILL.md missing                                    | "open <path>: no such file or directory" (step 3)                     |
//
// Owned elsewhere: name and description rules in validator/skill_test.go; error texts in model/error_test.go.
//
// Accepted: SkillStore.Skill serves SKILL.md even when the metadata is invalid.
//
// Outside the unit: production assets/skills content; e2e covers it.
//
// Branch coverage
// | Function                | Condition                  | True            | False      |
// |-------------------------|----------------------------|-----------------|------------|
// | SkillStore.ListMetadata | Metadata fails for a skill | 1 (four broken) | 1 (sample) |
// | SkillStore.ListMetadata | Skill map empty            | 2               | 1          |
// | SkillStore.Metadata     | Cache holds the metadata   | 4               | 3, 5       |
// | SkillStore.Metadata     | Name in the skill map      | 3               | 6, 7       |
// | SkillStore.Metadata     | metadata.yaml readable     | 3               | 8          |
// | SkillStore.Metadata     | metadata.yaml decodes      | 3               | 9, 10      |
// | SkillStore.Metadata     | Checks pass                | 3               | 11, 12     |
// | SkillStore.Skill        | Cache holds the text       | 14              | 13         |
// | SkillStore.Skill        | Name in the skill map      | 13              | 15, 16     |
// | SkillStore.Skill        | SKILL.md readable          | 13              | 17         |
//
// Test mapping
// | Test                                                               | Scenarios                                      |
// |--------------------------------------------------------------------|------------------------------------------------|
// | TestSkillStoreListMetadata/valid skills only; broken skills logged | 1                                              |
// | TestSkillStoreListMetadata/no skills                               | 2                                              |
// | TestSkillStoreMetadata/known skill                                 | 3                                              |
// | TestSkillStoreMetadataCache/second read uses the cache             | 4                                              |
// | TestSkillStoreMetadataFailureNotCached/file fixed after a failure  | 5                                              |
// | TestSkillStoreMetadataErrors/unknown skill                         | 6                                              |
// | TestSkillStoreMetadataErrors/path as name                          | 7                                              |
// | TestSkillStoreMetadataErrors/no metadata file                      | 8                                              |
// | TestSkillStoreMetadataErrors/empty metadata file                   | 9                                              |
// | TestSkillStoreMetadataErrors/unknown metadata key                  | 10                                             |
// | TestSkillStoreMetadataErrors/name differs from directory           | 11                                             |
// | TestSkillStoreMetadataErrors/description over the limit            | 12                                             |
// | TestSkillStoreSkill/known skill                                    | 13                                             |
// | TestSkillStoreSkillCache/second read uses the cache                | 14                                             |
// | TestSkillStoreSkillErrors/unknown skill                            | 15                                             |
// | TestSkillStoreSkillErrors/path as name                             | 16                                             |
// | TestSkillStoreSkillErrors/SKILL.md missing                         | 17                                             |
// | No test                                                            | Cache expiry: needs DurationCache of real time |

package asset_test

import . "github.com/ngo/precedent_mcp/internal/asset"

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleMetadata = model.Metadata{
	Name:        "sample",
	Description: "Sample skill for tests",
	Digest:      "sha256:ceeb3cadcf67e1bfefe503cf608cd3f8b9cf473ae05e490b250ec3f32d7eb137",
	Size:        67,
}

const sampleMetadataYAML = "name: sample\ndescription: Sample skill for tests\n" +
	"digest: sha256:ceeb3cadcf67e1bfefe503cf608cd3f8b9cf473ae05e490b250ec3f32d7eb137\nsize: 67\n"

const sampleSkillText = "---\nname: sample\ndescription: Sample skill for tests\n---\n\n# Sample\n"

var sampleSkills = map[env.SkillName]model.SkillPaths{
	"sample": {Skill: "skills/sample/SKILL.md", Metadata: "skills/sample/metadata.yaml"},
}

func TestSkillStoreListMetadata(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]model.SkillPaths
		want           []model.Metadata
		wantLog        string
	}{
		{
			name:           "valid skills only; broken skills logged",
			givenDirectory: "testdata",
			givenSkills: map[env.SkillName]model.SkillPaths{
				"long-description": {Skill: "skills/long-description/SKILL.md", Metadata: "skills/long-description/metadata.yaml"},
				"name-mismatch":    {Skill: "skills/name-mismatch/SKILL.md", Metadata: "skills/name-mismatch/metadata.yaml"},
				"no-metadata":      {Skill: "skills/no-metadata/SKILL.md", Metadata: "skills/no-metadata/metadata.yaml"},
				"sample":           {Skill: "skills/sample/SKILL.md", Metadata: "skills/sample/metadata.yaml"},
				"unknown-key":      {Skill: "skills/unknown-key/SKILL.md", Metadata: "skills/unknown-key/metadata.yaml"},
			},
			want: []model.Metadata{sampleMetadata},
			wantLog: "skip skill long-description: description must be 1 to 50 characters, got 57\n" +
				"skip skill name-mismatch: skill name \"other-name\" does not match directory \"name-mismatch\"\n" +
				"skip skill no-metadata: open testdata/skills/no-metadata/metadata.yaml: no such file or directory\n" +
				"skip skill unknown-key: invalid metadata: yaml: unmarshal errors:\n  line 3: field license not found in type model.Metadata\n",
		},
		{
			name:           "no skills",
			givenDirectory: "testdata",
			givenSkills:    map[env.SkillName]model.SkillPaths{},
			want:           []model.Metadata{},
			wantLog:        "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotLog bytes.Buffer
			previousWriter, previousFlags := log.Writer(), log.Flags()
			log.SetOutput(&gotLog)
			log.SetFlags(0)
			t.Cleanup(func() {
				log.SetOutput(previousWriter)
				log.SetFlags(previousFlags)
			})

			got := NewSkillStore(test.givenDirectory, test.givenSkills).ListMetadata()

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantLog, gotLog.String())
		})
	}
}

func TestSkillStoreMetadata(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]model.SkillPaths
		givenSkill     string
		want           model.Metadata
	}{
		{"known skill", "testdata", sampleSkills, "sample", sampleMetadata},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewSkillStore(test.givenDirectory, test.givenSkills).Metadata(test.givenSkill)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkillStoreMetadataCache(t *testing.T) {
	tests := []struct {
		name          string
		givenSkills   map[env.SkillName]model.SkillPaths
		givenSkill    string
		givenMetadata string
		want          model.Metadata
	}{
		{"second read uses the cache", sampleSkills, "sample", sampleMetadataYAML, sampleMetadata},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, string(test.givenSkills[env.SkillName(test.givenSkill)].Metadata))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(test.givenMetadata), 0o644))
			store := NewSkillStore(directory, test.givenSkills)
			_, firstErr := store.Metadata(test.givenSkill)
			require.NoError(t, firstErr)
			require.NoError(t, os.Remove(path))

			got, err := store.Metadata(test.givenSkill)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkillStoreMetadataFailureNotCached(t *testing.T) {
	tests := []struct {
		name               string
		givenSkills        map[env.SkillName]model.SkillPaths
		givenSkill         string
		givenFirstMetadata string
		givenFixedMetadata string
		want               model.Metadata
	}{
		{"file fixed after a failure", sampleSkills, "sample",
			"name: other-name\ndescription: Sample skill for tests\n", sampleMetadataYAML, sampleMetadata},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, string(test.givenSkills[env.SkillName(test.givenSkill)].Metadata))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(test.givenFirstMetadata), 0o644))
			store := NewSkillStore(directory, test.givenSkills)
			_, firstErr := store.Metadata(test.givenSkill)
			require.Error(t, firstErr)
			require.NoError(t, os.WriteFile(path, []byte(test.givenFixedMetadata), 0o644))

			got, err := store.Metadata(test.givenSkill)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkillStoreMetadataErrors(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]model.SkillPaths
		givenSkill     string
		want           model.Metadata
		wantError      error
	}{
		{"unknown skill", "testdata", sampleSkills, "nope", model.Metadata{}, model.ErrSkillNotFound("nope")},
		{"path as name", "testdata", sampleSkills, "../sample", model.Metadata{}, model.ErrSkillNotFound("../sample")},
		{"no metadata file", "testdata",
			map[env.SkillName]model.SkillPaths{"no-metadata": {Skill: "skills/no-metadata/SKILL.md", Metadata: "skills/no-metadata/metadata.yaml"}},
			"no-metadata", model.Metadata{},
			&fs.PathError{Op: "open", Path: "testdata/skills/no-metadata/metadata.yaml", Err: syscall.ENOENT}},
		{"empty metadata file", "testdata",
			map[env.SkillName]model.SkillPaths{"empty-metadata": {Skill: "skills/empty-metadata/SKILL.md", Metadata: "skills/empty-metadata/metadata.yaml"}},
			"empty-metadata", model.Metadata{}, model.ErrInvalidMetadata("empty-metadata", io.EOF)},
		{"unknown metadata key", "testdata",
			map[env.SkillName]model.SkillPaths{"unknown-key": {Skill: "skills/unknown-key/SKILL.md", Metadata: "skills/unknown-key/metadata.yaml"}},
			"unknown-key", model.Metadata{},
			model.ErrInvalidMetadata("unknown-key", errors.New("yaml: unmarshal errors:\n  line 3: field license not found in type model.Metadata"))},
		{"name differs from directory", "testdata",
			map[env.SkillName]model.SkillPaths{"name-mismatch": {Skill: "skills/name-mismatch/SKILL.md", Metadata: "skills/name-mismatch/metadata.yaml"}},
			"name-mismatch", model.Metadata{}, model.ErrSkillNameMismatch("other-name", "name-mismatch")},
		{"description over the limit", "testdata",
			map[env.SkillName]model.SkillPaths{"long-description": {Skill: "skills/long-description/SKILL.md", Metadata: "skills/long-description/metadata.yaml"}},
			"long-description", model.Metadata{}, model.ErrDescriptionLength("long-description", 57)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewSkillStore(test.givenDirectory, test.givenSkills).Metadata(test.givenSkill)

			assert.Equal(t, test.wantError, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkillStoreSkill(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]model.SkillPaths
		givenSkill     string
		want           string
	}{
		{"known skill", "testdata", sampleSkills, "sample", sampleSkillText},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewSkillStore(test.givenDirectory, test.givenSkills).Skill(test.givenSkill)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkillStoreSkillCache(t *testing.T) {
	tests := []struct {
		name           string
		givenSkills    map[env.SkillName]model.SkillPaths
		givenSkill     string
		givenSkillText string
		want           string
	}{
		{"second read uses the cache", sampleSkills, "sample", sampleSkillText, sampleSkillText},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, string(test.givenSkills[env.SkillName(test.givenSkill)].Skill))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(test.givenSkillText), 0o644))
			store := NewSkillStore(directory, test.givenSkills)
			_, firstErr := store.Skill(test.givenSkill)
			require.NoError(t, firstErr)
			require.NoError(t, os.Remove(path))

			got, err := store.Skill(test.givenSkill)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSkillStoreSkillErrors(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenSkills    map[env.SkillName]model.SkillPaths
		givenSkill     string
		want           string
		wantError      error
	}{
		{"unknown skill", "testdata", sampleSkills, "nope", "", model.ErrSkillNotFound("nope")},
		{"path as name", "testdata", sampleSkills, "../sample", "", model.ErrSkillNotFound("../sample")},
		{"SKILL.md missing", "testdata",
			map[env.SkillName]model.SkillPaths{"long-description": {Skill: "skills/long-description/SKILL.md", Metadata: "skills/long-description/metadata.yaml"}},
			"long-description", "", &fs.PathError{Op: "open", Path: "testdata/skills/long-description/SKILL.md", Err: syscall.ENOENT}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewSkillStore(test.givenDirectory, test.givenSkills).Skill(test.givenSkill)

			assert.Equal(t, test.wantError, err)
			assert.Equal(t, test.want, got)
		})
	}
}
