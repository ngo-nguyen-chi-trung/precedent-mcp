// Test plan: error.go
//
// Logic flow
// | Function             | Step | Action                                                              | Effect                                                       |
// |----------------------|------|---------------------------------------------------------------------|--------------------------------------------------------------|
// | Error.Error          | 1    | Return Message                                                      | The text callers log or wrap                                 |
// | ErrSkillNotFound     | 1    | Error skill_not_found, data name                                    | "skill <name> not found"                                     |
// | ErrExamplesNotFound  | 1    | Error examples_not_found, data name                                 | "examples <name> not found"                                  |
// | ErrInvalidMetadata   | 1    | Error invalid_metadata, data name, decode cause in the message      | "invalid metadata: <cause>"                                  |
// | ErrSkillNameMismatch | 1    | Error skill_name_mismatch, data name is the directory               | "skill name "<name>" does not match directory "<directory>"" |
// | ErrDescriptionLength | 1    | Error description_length, data name, limit and count in the message | "description must be 1 to 50 characters, got <count>"        |
// | HasCode              | 1    | Find an asset Error in the error chain                              | None found (other error, nil) → false                        |
// | HasCode              | 2    | Compare its code                                                    | true only for the same code                                  |
//
// Scenarios
// | Number | Function             | Result  | Value class    | Input                                                        | Expected                                                                                     |
// |--------|----------------------|---------|----------------|--------------------------------------------------------------|----------------------------------------------------------------------------------------------|
// | 1      | ErrSkillNotFound     | Success | Middle value   | nope                                                         | skill_not_found; "skill nope not found"; data name nope; Error() is the message              |
// | 2      | ErrExamplesNotFound  | Success | Middle value   | nope                                                         | examples_not_found; "examples nope not found"; data name nope; Error() is the message        |
// | 3      | ErrInvalidMetadata   | Success | Middle value   | a; cause "bad"                                               | invalid_metadata; "invalid metadata: bad"; data name a; Error() is the message               |
// | 4      | ErrSkillNameMismatch | Success | Middle value   | name a; directory b                                          | skill_name_mismatch; data name b; Error() is the message                                     |
// | 5      | ErrDescriptionLength | Success | Middle value   | a; count 51                                                  | description_length; "description must be 1 to 50 characters, got 51"; Error() is the message |
// | 6      | HasCode              | Success | Middle value   | ErrSkillNotFound(nope); skill_not_found                      | true (steps 1, 2)                                                                            |
// | 7      | HasCode              | Success | Boundary value | ErrSkillNotFound(nope) wrapped once with %w; skill_not_found | true (step 1)                                                                                |
// | 8      | HasCode              | Failure | Middle value   | ErrExamplesNotFound(nope); skill_not_found                   | false (step 2)                                                                               |
// | 9      | HasCode              | Failure | Middle value   | errors.New("other"); skill_not_found                         | false (step 1)                                                                               |
// | 10     | HasCode              | Failure | Boundary value | nil; skill_not_found                                         | false (step 1)                                                                               |
//
// Owned elsewhere: count limits in validator/skill_test.go.
//
// Branch coverage
// | Function | Condition                | True    | False |
// |----------|--------------------------|---------|-------|
// | HasCode  | Asset Error in the chain | 6, 7, 8 | 9, 10 |
// | HasCode  | Same code                | 6, 7    | 8     |
//
// Test mapping
// | Test                                              | Scenarios |
// |---------------------------------------------------|-----------|
// | TestErrSkillNotFound/names the skill              | 1         |
// | TestErrExamplesNotFound/names the examples        | 2         |
// | TestErrInvalidMetadata/carries the decode cause   | 3         |
// | TestErrSkillNameMismatch/data names the directory | 4         |
// | TestErrDescriptionLength/names the count          | 5         |
// | TestHasCode/same code                             | 6         |
// | TestHasCode/wrapped error                         | 7         |
// | TestHasCode/other code                            | 8         |
// | TestHasCode/not an asset error                    | 9         |
// | TestHasCode/nil error                             | 10        |
// | No test                                           | None      |

package model_test

import . "github.com/ngo/precedent_mcp/internal/asset/model"

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
)

func TestErrSkillNotFound(t *testing.T) {
	tests := []struct {
		name        string
		givenName   string
		want        *Error
		wantMessage string
	}{
		{"names the skill", "nope",
			&Error{Code: env.AssetErrorSkillNotFound, Message: "skill nope not found", Data: SkillErrorData{Name: "nope"}},
			"skill nope not found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ErrSkillNotFound(test.givenName)

			assert.Equal(t, test.want, got)
			assert.EqualError(t, got, test.wantMessage)
		})
	}
}

func TestErrExamplesNotFound(t *testing.T) {
	tests := []struct {
		name        string
		givenName   string
		want        *Error
		wantMessage string
	}{
		{"names the examples", "nope",
			&Error{Code: env.AssetErrorExamplesNotFound, Message: "examples nope not found", Data: ExamplesErrorData{Name: "nope"}},
			"examples nope not found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ErrExamplesNotFound(test.givenName)

			assert.Equal(t, test.want, got)
			assert.EqualError(t, got, test.wantMessage)
		})
	}
}

func TestErrInvalidMetadata(t *testing.T) {
	tests := []struct {
		name        string
		givenName   string
		givenError  error
		want        *Error
		wantMessage string
	}{
		{"carries the decode cause", "a", errors.New("bad"),
			&Error{Code: env.AssetErrorInvalidMetadata, Message: "invalid metadata: bad", Data: SkillErrorData{Name: "a"}},
			"invalid metadata: bad"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ErrInvalidMetadata(test.givenName, test.givenError)

			assert.Equal(t, test.want, got)
			assert.EqualError(t, got, test.wantMessage)
		})
	}
}

func TestErrSkillNameMismatch(t *testing.T) {
	tests := []struct {
		name           string
		givenName      string
		givenDirectory string
		want           *Error
		wantMessage    string
	}{
		{"data names the directory", "a", "b",
			&Error{Code: env.AssetErrorSkillNameMismatch, Message: `skill name "a" does not match directory "b"`, Data: SkillErrorData{Name: "b"}},
			`skill name "a" does not match directory "b"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ErrSkillNameMismatch(test.givenName, test.givenDirectory)

			assert.Equal(t, test.want, got)
			assert.EqualError(t, got, test.wantMessage)
		})
	}
}

func TestErrDescriptionLength(t *testing.T) {
	tests := []struct {
		name        string
		givenName   string
		givenCount  int
		want        *Error
		wantMessage string
	}{
		{"names the count", "a", 51,
			&Error{Code: env.AssetErrorDescriptionLength, Message: "description must be 1 to 50 characters, got 51", Data: SkillErrorData{Name: "a"}},
			"description must be 1 to 50 characters, got 51"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ErrDescriptionLength(test.givenName, test.givenCount)

			assert.Equal(t, test.want, got)
			assert.EqualError(t, got, test.wantMessage)
		})
	}
}

func TestHasCode(t *testing.T) {
	tests := []struct {
		name       string
		givenError error
		givenCode  env.AssetErrorCode
		want       bool
	}{
		{"same code", ErrSkillNotFound("nope"), env.AssetErrorSkillNotFound, true},
		{"wrapped error", fmt.Errorf("wrapped: %w", ErrSkillNotFound("nope")), env.AssetErrorSkillNotFound, true},
		{"other code", ErrExamplesNotFound("nope"), env.AssetErrorSkillNotFound, false},
		{"not an asset error", errors.New("other"), env.AssetErrorSkillNotFound, false},
		{"nil error", nil, env.AssetErrorSkillNotFound, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := HasCode(test.givenError, test.givenCode)

			assert.Equal(t, test.want, got)
		})
	}
}
