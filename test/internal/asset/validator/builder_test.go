// Test plan: builder.go
//
// Logic flow
// | Function                               | Step | Action                                                        | Effect                                       |
// |----------------------------------------|------|---------------------------------------------------------------|----------------------------------------------|
// | Pipeline.Run                           | 1    | Run the steps in the order they were added, on the same value | Each check sees the value                    |
// | Pipeline.Run                           | 2    | A step fails                                                  | Stop; return that error; later steps not run |
// | Pipeline.Run                           | 3    | Every step passes                                             | nil                                          |
// | NewBuilder, Builder.Add, Builder.Build | 1    | Collect the steps in call order                               | Build gives the pipeline                     |
//
// Scenarios
// | Number | Function     | Result  | Value class    | Input                          | Expected                                  |
// |--------|--------------|---------|----------------|--------------------------------|-------------------------------------------|
// | 1      | Pipeline.Run | Success | Boundary value | No steps                       | nil (step 3)                              |
// | 2      | Pipeline.Run | Success | Middle value   | Two passing steps              | nil; both ran in order (steps 1, 3)       |
// | 3      | Pipeline.Run | Failure | Middle value   | First step fails               | First error; second step not run (step 2) |
// | 4      | Pipeline.Run | Failure | Boundary value | Second step fails after a pass | Second error; first step ran (steps 1, 2) |
//
// Branch coverage
// | Function     | Condition    | True    | False |
// |--------------|--------------|---------|-------|
// | Pipeline.Run | A step fails | 3, 4    | 1, 2  |
// | Pipeline.Run | Steps added  | 2, 3, 4 | 1     |
//
// Test mapping
// | Test                                     | Scenarios |
// |------------------------------------------|-----------|
// | TestPipelineRun/no steps pass            | 1         |
// | TestPipelineRun/all steps pass in order  | 2         |
// | TestPipelineRun/first failure stops      | 3         |
// | TestPipelineRun/last failure is returned | 4         |
// | No test                                  | None      |

package validator_test

import . "github.com/ngo/precedent_mcp/internal/asset/validator"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errStep = errors.New("boom")

func TestPipelineRun(t *testing.T) {
	tests := []struct {
		name        string
		givenErrors []error
		givenValue  string
		want        error
		wantRan     []int
	}{
		{"no steps pass", nil, "value", nil, nil},
		{"all steps pass in order", []error{nil, nil}, "value", nil, []int{0, 1}},
		{"first failure stops", []error{errStep, nil}, "value", errStep, []int{0}},
		{"last failure is returned", []error{nil, errStep}, "value", errStep, []int{0, 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotRan []int
			builder := NewBuilder[string]()
			for i, stepError := range test.givenErrors {
				builder.Add(func(string) error {
					gotRan = append(gotRan, i)
					return stepError
				})
			}

			got := builder.Build().Run(test.givenValue)

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantRan, gotRan)
		})
	}
}
