// Test plan: error.go
//
// Logic flow
// | Function   | Step | Action                  | Effect                                                                |
// |------------|------|-------------------------|-----------------------------------------------------------------------|
// | Error.JSON | 1    | Encode code and message | The data of every is_error tool response, as tool_error in tools.yaml |
//
// Scenarios
// | Number | Function   | Result  | Value class  | Input                                  | Expected                                               |
// |--------|------------|---------|--------------|----------------------------------------|--------------------------------------------------------|
// | 1      | Error.JSON | Success | Middle value | Code internal, message "weaviate down" | {"code":"internal","message":"weaviate down"} (step 1) |
//
// Failure scenarios: none. Encoding two strings cannot fail.
//
// Branch coverage
// | Function   | Condition | True | False |
// |------------|-----------|------|-------|
// | Error.JSON | None      | —    | —     |
//
// Test mapping
// | Test                           | Scenarios |
// |--------------------------------|-----------|
// | TestErrorJSON/code and message | 1         |
// | No test                        | None      |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"testing"

	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
)

func TestErrorJSON(t *testing.T) {
	tests := []struct {
		name       string
		givenError Error
		want       string
	}{
		{"code and message", Error{Code: env.ServiceErrorInternal, Message: "weaviate down"}, `{"code":"internal","message":"weaviate down"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenError.JSON()

			assert.Equal(t, test.want, string(got))
		})
	}
}
