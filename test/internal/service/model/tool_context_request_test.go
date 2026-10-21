// Test plan: tool_context_request.go
//
// Logic flow
// | Function                           | Step | Action                                                                                | Effect                                                                     |
// |------------------------------------|------|---------------------------------------------------------------------------------------|----------------------------------------------------------------------------|
// | AddContextRequest.ToContextObjects | 1    | Read the current time in UTC once                                                     | One base time for the call                                                 |
// | AddContextRequest.ToContextObjects | 2    | One object per entry in request order; ThreadID = thread UUID text; Content = context | Stored fields mirror the request                                           |
// | AddContextRequest.ToContextObjects | 3    | Identity from thread and context                                                      | Same thread and same context are one entry; resending it overwrites it     |
// | AddContextRequest.ToContextObjects | 4    | CreatedAt = base time; each entry later than the one before                           | Entries arrive oldest first; a resent entry moves to the end of its thread |
//
// Scenarios
// | Number | Function                           | Result  | Value class    | Input                              | Expected                                                                                      |
// |--------|------------------------------------|---------|----------------|------------------------------------|-----------------------------------------------------------------------------------------------|
// | 1      | AddContextRequest.ToContextObjects | Success | Middle value   | Two entries in two threads         | Two objects in request order; ThreadID and Content copied (step 2)                            |
// | 2      | AddContextRequest.ToContextObjects | Success | Middle value   | Same thread and same context twice | Same ID (step 3)                                                                              |
// | 3      | AddContextRequest.ToContextObjects | Success | Middle value   | Same context in two threads        | Different IDs (step 3)                                                                        |
// | 4      | AddContextRequest.ToContextObjects | Success | Middle value   | Two contexts in one thread         | Different IDs (step 3)                                                                        |
// | 5      | AddContextRequest.ToContextObjects | Success | Boundary value | Ten entries; maximum batch         | First CreatedAt UTC inside the call window; each entry later than the one before (steps 1, 4) |
//
// Failure scenarios: none. ToContextObjects has no error path.
//
// Branch coverage
// | Function                           | Condition | True | False |
// |------------------------------------|-----------|------|-------|
// | AddContextRequest.ToContextObjects | None      | —    | —     |
//
// Test mapping
// | Test                                                      | Scenarios                                                 |
// |-----------------------------------------------------------|-----------------------------------------------------------|
// | TestToContextObjects/entries in two threads               | 1                                                         |
// | TestToContextObjectsIdentity/same thread and same context | 2                                                         |
// | TestToContextObjectsIdentity/same context in two threads  | 3                                                         |
// | TestToContextObjectsIdentity/two contexts in one thread   | 4                                                         |
// | TestToContextObjectsCreatedAt/ten entries oldest first    | 5                                                         |
// | No test                                                   | No entries: the relay rejects an empty entries list first |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	contextThreadA = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	contextThreadB = uuid.MustParse("11111111-1111-1111-1111-111111111111")
)

func TestToContextObjects(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest AddContextRequest
		want         []dbmodel.ContextObject
	}{
		{
			name: "entries in two threads",
			givenRequest: AddContextRequest{Entries: []ContextEntry{
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
				{ThreadID: contextThreadB, Context: "Constraint: EU region"},
			}},
			want: []dbmodel.ContextObject{
				{ThreadID: "22222222-2222-2222-2222-222222222222", Content: "Preference: tabs"},
				{ThreadID: "11111111-1111-1111-1111-111111111111", Content: "Constraint: EU region"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenRequest.ToContextObjects()

			require.Len(t, got, len(test.want))
			want := slices.Clone(test.want)
			for i := range want {
				want[i].ID, want[i].CreatedAt = got[i].ID, got[i].CreatedAt
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestToContextObjectsIdentity(t *testing.T) {
	tests := []struct {
		name               string
		givenRequest       AddContextRequest
		wantSameIdentifier bool
	}{
		{"same thread and same context", AddContextRequest{Entries: []ContextEntry{
			{ThreadID: contextThreadA, Context: "Preference: tabs"},
			{ThreadID: contextThreadA, Context: "Preference: tabs"},
		}}, true},
		{"same context in two threads", AddContextRequest{Entries: []ContextEntry{
			{ThreadID: contextThreadA, Context: "Preference: tabs"},
			{ThreadID: contextThreadB, Context: "Preference: tabs"},
		}}, false},
		{"two contexts in one thread", AddContextRequest{Entries: []ContextEntry{
			{ThreadID: contextThreadA, Context: "Preference: tabs"},
			{ThreadID: contextThreadA, Context: "Preference: spaces"},
		}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenRequest.ToContextObjects()

			require.Len(t, got, 2)
			assert.Equal(t, test.wantSameIdentifier, got[0].ID == got[1].ID)
		})
	}
}

func TestToContextObjectsCreatedAt(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest AddContextRequest
		wantLocation *time.Location
	}{
		{"ten entries oldest first", AddContextRequest{Entries: []ContextEntry{
			{ThreadID: contextThreadA, Context: "entry 0"},
			{ThreadID: contextThreadA, Context: "entry 1"},
			{ThreadID: contextThreadA, Context: "entry 2"},
			{ThreadID: contextThreadA, Context: "entry 3"},
			{ThreadID: contextThreadA, Context: "entry 4"},
			{ThreadID: contextThreadA, Context: "entry 5"},
			{ThreadID: contextThreadA, Context: "entry 6"},
			{ThreadID: contextThreadA, Context: "entry 7"},
			{ThreadID: contextThreadA, Context: "entry 8"},
			{ThreadID: contextThreadA, Context: "entry 9"},
		}}, time.UTC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := time.Now().UTC()

			got := test.givenRequest.ToContextObjects()

			after := time.Now().UTC()
			require.Len(t, got, len(test.givenRequest.Entries))
			assert.WithinRange(t, got[0].CreatedAt, before, after)
			for i := range got {
				assert.Equal(t, test.wantLocation, got[i].CreatedAt.Location())
			}
			for i := 1; i < len(got); i++ {
				assert.True(t, got[i].CreatedAt.After(got[i-1].CreatedAt), "entry %d is not later than entry %d", i, i-1)
			}
		})
	}
}
