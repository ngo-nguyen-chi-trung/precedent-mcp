// Test plan: builder.go
//
// Logic flow
// | Function     | Step | Action                                       | Effect                                                     |
// |--------------|------|----------------------------------------------|------------------------------------------------------------|
// | Builder      | 1    | NewBuilder().Add(step)….Build()              | The pipeline holds the steps in the order added            |
// | Pipeline.Run | 1    | Run the steps in order with the same request | A step sees what earlier steps set on the request          |
// | Pipeline.Run | 2    | A step returns an error or a non-zero result | Stop; return that result and error; later steps do not run |
// | Pipeline.Run | 3    | Every step passes                            | Zero result and nil error: the service continues           |
//
// Scenarios
// | Number | Function     | Result  | Value class    | Input                                                 | Expected                                                                              |
// |--------|--------------|---------|----------------|-------------------------------------------------------|---------------------------------------------------------------------------------------|
// | 1      | Pipeline.Run | Success | Boundary value | Built without steps                                   | Zero result; nil error (step 3)                                                       |
// | 2      | Pipeline.Run | Success | Middle value   | Two passing steps; each records itself on the request | Zero result; nil error; request holds first, then second (builder step 1; steps 1, 3) |
// | 3      | Pipeline.Run | Failure | Middle value   | First step returns an is_error result                 | That result; nil error; second step not run (step 2)                                  |
// | 4      | Pipeline.Run | Failure | Middle value   | First step returns an error                           | Nil result; that error; second step not run (step 2)                                  |
// | 5      | Pipeline.Run | Failure | Boundary value | Second step returns an error after a pass             | Nil result; that error; both ran (step 2)                                             |
//
// Branch coverage
// | Function     | Condition                         | True    | False |
// |--------------|-----------------------------------|---------|-------|
// | Pipeline.Run | Steps given                       | 2 to 5  | 1     |
// | Pipeline.Run | Step returns a result or an error | 3, 4, 5 | 2     |
//
// Test mapping
// | Test                                         | Scenarios |
// |----------------------------------------------|-----------|
// | TestPipelineRun/no steps pass                | 1         |
// | TestPipelineRun/steps run in the order added | 2         |
// | TestPipelineRun/result stops the pipeline    | 3         |
// | TestPipelineRun/error stops the pipeline     | 4         |
// | TestPipelineRun/error after a pass           | 5         |
// | No test                                      | None      |

package validator_test

import . "github.com/ngo/precedent_mcp/internal/service/validator"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type pipelineResult struct {
	text string
}

var (
	pipelineFailedResult = &pipelineResult{text: "failed"}
	errPipelineStep      = errors.New("weaviate down")
)

func TestPipelineRun(t *testing.T) {
	tests := []struct {
		name         string
		givenResults []*pipelineResult
		givenErrors  []error
		want         *pipelineResult
		wantError    error
		wantRan      []int
	}{
		{"no steps pass", nil, nil, nil, nil, nil},
		{"steps run in the order added", []*pipelineResult{nil, nil}, []error{nil, nil}, nil, nil, []int{0, 1}},
		{"result stops the pipeline", []*pipelineResult{pipelineFailedResult, nil}, []error{nil, nil}, pipelineFailedResult, nil, []int{0}},
		{"error stops the pipeline", []*pipelineResult{nil, nil}, []error{errPipelineStep, nil}, nil, errPipelineStep, []int{0}},
		{"error after a pass", []*pipelineResult{nil, nil}, []error{nil, errPipelineStep}, nil, errPipelineStep, []int{0, 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := NewBuilder[*[]int, *pipelineResult]()
			for i := range test.givenResults {
				builder.Add(func(_ context.Context, ran *[]int) (*pipelineResult, error) {
					*ran = append(*ran, i)
					return test.givenResults[i], test.givenErrors[i]
				})
			}
			var gotRan []int

			got, err := builder.Build().Run(context.Background(), &gotRan)

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantError, err)
			assert.Equal(t, test.wantRan, gotRan)
		})
	}
}
