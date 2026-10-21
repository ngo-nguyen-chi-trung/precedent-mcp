// Test plan: tool_context_response.go
//
// Logic flow
// | Function                                  | Step | Action                                                               | Effect                                           |
// |-------------------------------------------|------|----------------------------------------------------------------------|--------------------------------------------------|
// | RetrieveContextResult.FromContextSearch   | 1    | Start from an empty, non-nil list                                    | No hits give {"results":[]}                      |
// | RetrieveContextResult.FromContextSearch   | 2    | Copy thread_id, context, score of each hit in store order            | Best match stays first                           |
// | GetContextThreadResult.FromContextObjects | 1    | Set the requested thread_id; start from an empty, non-nil list       | Unknown thread gives entries []                  |
// | GetContextThreadResult.FromContextObjects | 2    | Copy thread_id, context, created_at of each entry in the given order | Ordering stays with ToolService.GetContextThread |
//
// Scenarios
// | Number | Function                                  | Result  | Value class    | Input                  | Expected                                                                        |
// |--------|-------------------------------------------|---------|----------------|------------------------|---------------------------------------------------------------------------------|
// | 1      | RetrieveContextResult.FromContextSearch   | Success | Middle value   | Two hits               | Results in order; JSON {"results":[{"thread_id", "context", "score"}]} (step 2) |
// | 2      | RetrieveContextResult.FromContextSearch   | Success | Boundary value | No hits                | {"results":[]} (step 1)                                                         |
// | 3      | GetContextThreadResult.FromContextObjects | Success | Middle value   | Thread and two entries | thread_id set; entries in the given order; created_at in RFC 3339 (steps 1, 2)  |
// | 4      | GetContextThreadResult.FromContextObjects | Success | Boundary value | Thread and no entries  | {"thread_id": the thread, "entries":[]} (step 1)                                |
//
// Branch coverage
// | Function                                  | Condition     | True | False |
// |-------------------------------------------|---------------|------|-------|
// | RetrieveContextResult.FromContextSearch   | Hits found    | 1    | 2     |
// | GetContextThreadResult.FromContextObjects | Entries found | 3    | 4     |
//
// Test mapping
// | Test                               | Scenarios                                                                                                                                                                                   |
// |------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | TestFromContextSearch/two hits     | 1                                                                                                                                                                                           |
// | TestFromContextSearch/no hits      | 2                                                                                                                                                                                           |
// | TestFromContextObjects/two entries | 3                                                                                                                                                                                           |
// | TestFromContextObjects/no entries  | 4                                                                                                                                                                                           |
// | No test                            | ContextToolCallEntryCountError: only reachable with entry counts the relay rejects; ContextToolCallInternalErrorResponse: proven by ToolService scenarios 8, 14, 18 in tool_context_test.go |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var contextCreatedAt = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func TestFromContextSearch(t *testing.T) {
	tests := []struct {
		name       string
		givenFound []dbmodel.ContextSearch
		want       RetrieveContextResult
		wantJSON   string
	}{
		{
			name: "two hits",
			givenFound: []dbmodel.ContextSearch{
				{ThreadID: "22222222-2222-2222-2222-222222222222", Content: "user prefers tabs", Metadata: dbmodel.ContextMetadata{Score: 0.9}},
				{ThreadID: "11111111-1111-1111-1111-111111111111", Content: "editor is vim", Metadata: dbmodel.ContextMetadata{Score: 0.4}},
			},
			want: RetrieveContextResult{Results: []RankedContext{
				{ThreadID: "22222222-2222-2222-2222-222222222222", Context: "user prefers tabs", Score: 0.9},
				{ThreadID: "11111111-1111-1111-1111-111111111111", Context: "editor is vim", Score: 0.4},
			}},
			wantJSON: `{"results":[` +
				`{"thread_id":"22222222-2222-2222-2222-222222222222","context":"user prefers tabs","score":0.9},` +
				`{"thread_id":"11111111-1111-1111-1111-111111111111","context":"editor is vim","score":0.4}]}`,
		},
		{
			name:       "no hits",
			givenFound: nil,
			want:       RetrieveContextResult{Results: []RankedContext{}},
			wantJSON:   `{"results":[]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got RetrieveContextResult

			got.FromContextSearch(test.givenFound)

			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.JSONEq(t, test.wantJSON, string(gotJSON))
		})
	}
}

func TestFromContextObjects(t *testing.T) {
	tests := []struct {
		name          string
		givenThreadID uuid.UUID
		givenFound    []dbmodel.ContextObject
		want          GetContextThreadResult
		wantJSON      string
	}{
		{
			name:          "two entries",
			givenThreadID: contextThreadA,
			givenFound: []dbmodel.ContextObject{
				{ThreadID: "22222222-2222-2222-2222-222222222222", Content: "later", CreatedAt: contextCreatedAt.Add(time.Second)},
				{ThreadID: "22222222-2222-2222-2222-222222222222", Content: "earlier", CreatedAt: contextCreatedAt},
			},
			want: GetContextThreadResult{ThreadID: contextThreadA, Entries: []ThreadContext{
				{ThreadID: "22222222-2222-2222-2222-222222222222", Context: "later", CreatedAt: contextCreatedAt.Add(time.Second)},
				{ThreadID: "22222222-2222-2222-2222-222222222222", Context: "earlier", CreatedAt: contextCreatedAt},
			}},
			wantJSON: `{"thread_id":"22222222-2222-2222-2222-222222222222","entries":[` +
				`{"thread_id":"22222222-2222-2222-2222-222222222222","context":"later","created_at":"2026-10-01T12:00:01Z"},` +
				`{"thread_id":"22222222-2222-2222-2222-222222222222","context":"earlier","created_at":"2026-10-01T12:00:00Z"}]}`,
		},
		{
			name:          "no entries",
			givenThreadID: contextThreadB,
			givenFound:    nil,
			want:          GetContextThreadResult{ThreadID: contextThreadB, Entries: []ThreadContext{}},
			wantJSON:      `{"thread_id":"11111111-1111-1111-1111-111111111111","entries":[]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got GetContextThreadResult

			got.FromContextObjects(test.givenThreadID, test.givenFound)

			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.JSONEq(t, test.wantJSON, string(gotJSON))
		})
	}
}
