// Test plan: example.go
//
// Logic flow
// | Function               | Step | Action                                   | Effect                                           |
// |------------------------|------|------------------------------------------|--------------------------------------------------|
// | ExampleStore.ListNames | 1    | Return the example names in sorted order | Same order on every call                         |
// | ExampleStore.Examples  | 1    | Accept only names of the examples map    | Other names, paths included → examples_not_found |
// | ExampleStore.Examples  | 2    | Read the examples file of the name       | Read failure → the file system error             |
// | ExampleStore.Examples  | 3    | Return the text unchanged                | An empty file is an empty text                   |
//
// Scenarios
// | Number | Function               | Result  | Value class    | Input                                  | Expected                                          |
// |--------|------------------------|---------|----------------|----------------------------------------|---------------------------------------------------|
// | 1      | ExampleStore.ListNames | Success | Middle value   | Names sample, empty-examples           | [empty-examples, sample] (step 1)                 |
// | 2      | ExampleStore.ListNames | Success | Boundary value | Empty examples map                     | No names (step 1)                                 |
// | 3      | ExampleStore.Examples  | Success | Middle value   | sample                                 | Text of its examples.md (steps 1 to 3)            |
// | 4      | ExampleStore.Examples  | Success | Boundary value | empty-examples; empty examples.md      | "" (step 3)                                       |
// | 5      | ExampleStore.Examples  | Failure | Middle value   | nope; not an example name              | examples_not_found for nope (step 1)              |
// | 6      | ExampleStore.Examples  | Failure | Boundary value | ../sample; a path, not an example name | examples_not_found for ../sample (step 1)         |
// | 7      | ExampleStore.Examples  | Failure | Boundary value | no-metadata; examples.md missing       | "open <path>: no such file or directory" (step 2) |
//
// Outside the unit: production assets/skills examples; e2e covers it.
//
// Branch coverage
// | Function               | Condition                | True | False |
// |------------------------|--------------------------|------|-------|
// | ExampleStore.ListNames | Examples map empty       | 2    | 1     |
// | ExampleStore.Examples  | Name in the examples map | 3, 4 | 5, 6  |
// | ExampleStore.Examples  | examples.md readable     | 3, 4 | 7     |
//
// Test mapping
// | Test                                               | Scenarios |
// |----------------------------------------------------|-----------|
// | TestExampleStoreListNames/sorted names             | 1         |
// | TestExampleStoreListNames/no names                 | 2         |
// | TestExampleStoreExamples/examples of a known name  | 3         |
// | TestExampleStoreExamples/empty examples.md         | 4         |
// | TestExampleStoreExamplesErrors/unknown name        | 5         |
// | TestExampleStoreExamplesErrors/path as name        | 6         |
// | TestExampleStoreExamplesErrors/examples.md missing | 7         |
// | No test                                            | None      |

package asset_test

import . "github.com/ngo/precedent_mcp/internal/asset"

import (
	"io/fs"
	"syscall"
	"testing"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleExamples = map[env.SkillName]env.AssetPath{
	"empty-examples": "skills/empty-examples/examples.md",
	"sample":         "skills/sample/examples.md",
}

func TestExampleStoreListNames(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenExamples  map[env.SkillName]env.AssetPath
		want           []env.SkillName
	}{
		{"sorted names", "testdata", sampleExamples, []env.SkillName{"empty-examples", "sample"}},
		{"no names", "testdata", map[env.SkillName]env.AssetPath{}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewExampleStore(test.givenDirectory, test.givenExamples).ListNames()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestExampleStoreExamples(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenExamples  map[env.SkillName]env.AssetPath
		givenName      string
		want           string
	}{
		{"examples of a known name", "testdata", sampleExamples, "sample", "# Examples for sample\n"},
		{"empty examples.md", "testdata", sampleExamples, "empty-examples", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewExampleStore(test.givenDirectory, test.givenExamples).Examples(test.givenName)

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestExampleStoreExamplesErrors(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenExamples  map[env.SkillName]env.AssetPath
		givenName      string
		want           string
		wantError      error
	}{
		{"unknown name", "testdata", sampleExamples, "nope", "", model.ErrExamplesNotFound("nope")},
		{"path as name", "testdata", sampleExamples, "../sample", "", model.ErrExamplesNotFound("../sample")},
		{"examples.md missing", "testdata", map[env.SkillName]env.AssetPath{"no-metadata": "skills/no-metadata/examples.md"},
			"no-metadata", "", &fs.PathError{Op: "open", Path: "testdata/skills/no-metadata/examples.md", Err: syscall.ENOENT}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewExampleStore(test.givenDirectory, test.givenExamples).Examples(test.givenName)

			assert.Equal(t, test.wantError, err)
			assert.Equal(t, test.want, got)
		})
	}
}
