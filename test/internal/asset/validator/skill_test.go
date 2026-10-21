// Test plan: skill.go
//
// Logic flow
// | Function          | Step | Action                                                                             | Effect                                      |
// |-------------------|------|------------------------------------------------------------------------------------|---------------------------------------------|
// | MatchesDirectory  | 1    | Metadata name equals the skill directory exactly                                   | Otherwise skill_name_mismatch               |
// | DescriptionLength | 1    | Description holds 1 to LimitSkillDescription (50) characters, counted as runes     | Otherwise description_length with the count |
//
// Scenarios
// | Number | Function          | Result  | Value class    | Input                              | Expected                                |
// |--------|-------------------|---------|----------------|------------------------------------|-----------------------------------------|
// | 1      | MatchesDirectory  | Success | Middle value   | Name sample in directory sample    | nil                                     |
// | 2      | MatchesDirectory  | Failure | Middle value   | Name other in directory sample     | skill_name_mismatch for other in sample |
// | 3      | MatchesDirectory  | Failure | Boundary value | Name Sample in directory sample    | skill_name_mismatch; case matters       |
// | 4      | MatchesDirectory  | Failure | Boundary value | No name in directory sample        | skill_name_mismatch for "" in sample    |
// | 5      | DescriptionLength | Failure | Boundary value | Empty description                  | description_length 0                    |
// | 6      | DescriptionLength | Success | Boundary value | 1 character                        | nil                                     |
// | 7      | DescriptionLength | Success | Middle value   | 25 characters                      | nil                                     |
// | 8      | DescriptionLength | Success | Boundary value | 50 characters                      | nil                                     |
// | 9      | DescriptionLength | Success | Boundary value | 50 two-byte characters (100 bytes) | nil; characters counted, not bytes      |
// | 10     | DescriptionLength | Failure | Boundary value | 51 characters                      | description_length 51                   |
//
// Branch coverage
// | Function          | Condition                       | True    | False  |
// |-------------------|---------------------------------|---------|--------|
// | MatchesDirectory  | Name differs from the directory | 2, 3, 4 | 1      |
// | DescriptionLength | Count outside 1 to 50           | 5, 10   | 6 to 9 |
//
// Test mapping
// | Test                                            | Scenarios |
// |-------------------------------------------------|-----------|
// | TestMatchesDirectory/same name                  | 1         |
// | TestMatchesDirectory/other name                 | 2         |
// | TestMatchesDirectory/other letter case          | 3         |
// | TestMatchesDirectory/no name                    | 4         |
// | TestDescriptionLength/empty description         | 5         |
// | TestDescriptionLength/one character             | 6         |
// | TestDescriptionLength/twenty-five characters    | 7         |
// | TestDescriptionLength/fifty characters          | 8         |
// | TestDescriptionLength/fifty two-byte characters | 9         |
// | TestDescriptionLength/fifty-one characters      | 10        |
// | No test                                         | None      |

package validator_test

import . "github.com/ngo/precedent_mcp/internal/asset/validator"

import (
	"strings"
	"testing"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/stretchr/testify/assert"
)

func TestMatchesDirectory(t *testing.T) {
	tests := []struct {
		name           string
		givenDirectory string
		givenMetadata  model.Metadata
		want           error
	}{
		{"same name", "sample", model.Metadata{Name: "sample"}, nil},
		{"other name", "sample", model.Metadata{Name: "other"}, model.ErrSkillNameMismatch("other", "sample")},
		{"other letter case", "sample", model.Metadata{Name: "Sample"}, model.ErrSkillNameMismatch("Sample", "sample")},
		{"no name", "sample", model.Metadata{}, model.ErrSkillNameMismatch("", "sample")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MatchesDirectory(test.givenDirectory)(test.givenMetadata)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestDescriptionLength(t *testing.T) {
	tests := []struct {
		name          string
		givenMetadata model.Metadata
		want          error
	}{
		{"empty description", model.Metadata{Name: "sample"}, model.ErrDescriptionLength("sample", 0)},
		{"one character", model.Metadata{Name: "sample", Description: "x"}, nil},
		{"twenty-five characters", model.Metadata{Name: "sample", Description: strings.Repeat("x", 25)}, nil},
		{"fifty characters", model.Metadata{Name: "sample", Description: strings.Repeat("x", 50)}, nil},
		{"fifty two-byte characters", model.Metadata{Name: "sample", Description: strings.Repeat("é", 50)}, nil},
		{"fifty-one characters", model.Metadata{Name: "sample", Description: strings.Repeat("x", 51)}, model.ErrDescriptionLength("sample", 51)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DescriptionLength()(test.givenMetadata)

			assert.Equal(t, test.want, got)
		})
	}
}
