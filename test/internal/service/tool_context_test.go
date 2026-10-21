// Test plan: tool_context.go
//
// Logic flow
// | Function                     | Step | Action                                                                                | Effect                                                            |
// |------------------------------|------|---------------------------------------------------------------------------------------|-------------------------------------------------------------------|
// | ToolService.AddContext       | 1    | AddContextRequest.ToContextObjects                                                    | One object per entry, in request order                            |
// | ToolService.AddContext       | 2    | Send each distinct entry once; same thread and same context are one entry             | A repeated entry is stored once                                   |
// | ToolService.AddContext       | 3    | ContextStore.Create(distinct entries)                                                 | Failure → is_error internal with the cause                        |
// | ToolService.AddContext       | 4    | Report every request position in succeeded or failed by the store result of its entry | A repeated entry reports the same result at each of its positions |
// | ToolService.AddContext       | 5    | Return {"succeeded": [...], "failed": [...]}                                          | Success, also when some entries failed                            |
// | ToolService.RetrieveContext  | 1    | limit = top_n; LimitTopN 20 when absent                                               | Never more than 20 results                                        |
// | ToolService.RetrieveContext  | 2    | ContextStore.Search(query, limit)                                                     | Failure → is_error internal with the cause                        |
// | ToolService.RetrieveContext  | 3    | Return the hits as results in store order                                             | Best match first                                                  |
// | ToolService.GetContextThread | 1    | ContextStore.Get(thread_id, 20)                                                       | The latest 20 entries; failure → is_error internal with the cause |
// | ToolService.GetContextThread | 2    | Order the entries oldest first                                                        | The thread reads in the order it was written                      |
// | ToolService.GetContextThread | 3    | Return {"thread_id", "entries"}                                                       | Unknown thread: entries []                                        |
//
// Scenarios
// | Number | Function                     | Result  | Value class    | Input                                                          | Expected                                                                                         |
// |--------|------------------------------|---------|----------------|----------------------------------------------------------------|--------------------------------------------------------------------------------------------------|
// | 1      | ToolService.AddContext       | Success | Middle value   | Three entries in two threads; store saves all                  | Create receives three objects in request order; succeeded 0, 1, 2; failed none (steps 1, 3 to 5) |
// | 2      | ToolService.AddContext       | Success | Middle value   | Three entries; store fails the second                          | succeeded 0, 2; failed 1 (step 4)                                                                |
// | 3      | ToolService.AddContext       | Success | Boundary value | Three entries; store fails every entry                         | succeeded none; failed 0, 1, 2 (step 4)                                                          |
// | 4      | ToolService.AddContext       | Success | Boundary value | One entry; minimum count                                       | Create receives one object; succeeded 0 (steps 1, 4)                                             |
// | 5      | ToolService.AddContext       | Success | Boundary value | Ten entries; maximum count                                     | Create receives ten objects; succeeded 0 to 9 (steps 1, 4)                                       |
// | 6      | ToolService.AddContext       | Success | Middle value   | Same entry at positions 0 and 2; another at 1; store saves all | Create receives two objects; succeeded 0, 1, 2 (steps 2, 4)                                      |
// | 7      | ToolService.AddContext       | Success | Middle value   | Same entry twice; store fails it                               | Create receives one object; failed 0, 1 (steps 2, 4)                                             |
// | 8      | ToolService.AddContext       | Failure | Middle value   | Create returns "weaviate down"                                 | is_error internal "weaviate down" (step 3)                                                       |
// | 9      | ToolService.RetrieveContext  | Success | Boundary value | top_n absent                                                   | Search limit 20 (step 1)                                                                         |
// | 10     | ToolService.RetrieveContext  | Success | Boundary value | top_n 1                                                        | Search limit 1 (step 1)                                                                          |
// | 11     | ToolService.RetrieveContext  | Success | Middle value   | top_n 5                                                        | Search limit 5 (step 1)                                                                          |
// | 12     | ToolService.RetrieveContext  | Success | Boundary value | top_n 20                                                       | Search limit 20 (step 1)                                                                         |
// | 13     | ToolService.RetrieveContext  | Success | Middle value   | Store returns two hits, best match first                       | Results in the same order with thread_id, context, score (step 3)                                |
// | 14     | ToolService.RetrieveContext  | Failure | Middle value   | Search returns "weaviate down"                                 | is_error internal "weaviate down" (step 2)                                                       |
// | 15     | ToolService.GetContextThread | Success | Middle value   | Store returns three entries, newest first                      | Get limit 20; entries oldest first; thread_id echoed (steps 1 to 3)                              |
// | 16     | ToolService.GetContextThread | Success | Boundary value | Store returns 20 entries, newest first                         | All 20 entries, oldest first (steps 1, 2)                                                        |
// | 17     | ToolService.GetContextThread | Success | Boundary value | Unknown thread; store returns nothing                          | thread_id echoed; entries [] (step 3)                                                            |
// | 18     | ToolService.GetContextThread | Failure | Middle value   | Get returns "weaviate down"                                    | is_error internal "weaviate down" (step 1)                                                       |
//
// Outside the handler contract: entry count 0 or above 10, empty context, thread_id not a UUID, top_n below 1 or above 20, and empty query; the relay Arguments step rejects them with tools.yaml and e2e covers that.
//
// Outside the handler: ContextStore.Get picks the latest 20 entries of the thread; e2e covers that.
//
// Branch coverage
// | Function                     | Condition                             | True       | False |
// |------------------------------|---------------------------------------|------------|-------|
// | ToolService.AddContext       | Entry repeated in the request         | 6, 7       | 1     |
// | ToolService.AddContext       | Store fails an entry                  | 2, 3, 7    | 1     |
// | ToolService.AddContext       | Create fails                          | 8          | 1     |
// | ToolService.RetrieveContext  | top_n given                           | 10, 11, 12 | 9     |
// | ToolService.RetrieveContext  | Search fails                          | 14         | 13    |
// | ToolService.GetContextThread | Store order differs from oldest first | 15, 16     | 17    |
// | ToolService.GetContextThread | Get fails                             | 18         | 15    |
//
// Test mapping
// | Test                                                  | Scenarios                                                                                                                                |
// |-------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
// | TestAddContext/entries in two threads                 | 1                                                                                                                                        |
// | TestAddContext/store fails one entry                  | 2                                                                                                                                        |
// | TestAddContext/store fails every entry                | 3                                                                                                                                        |
// | TestAddContext/minimum entry count                    | 4                                                                                                                                        |
// | TestAddContext/maximum entry count                    | 5                                                                                                                                        |
// | TestAddContext/repeated entry saved                   | 6                                                                                                                                        |
// | TestAddContext/repeated entry failed                  | 7                                                                                                                                        |
// | TestAddContextErrors/store create fails               | 8                                                                                                                                        |
// | TestRetrieveContextLimit/default when top_n is absent | 9                                                                                                                                        |
// | TestRetrieveContextLimit/minimum top_n                | 10                                                                                                                                       |
// | TestRetrieveContextLimit/top_n inside the range       | 11                                                                                                                                       |
// | TestRetrieveContextLimit/maximum top_n                | 12                                                                                                                                       |
// | TestRetrieveContext/hits keep the store order         | 13                                                                                                                                       |
// | TestRetrieveContextErrors/store search fails          | 14                                                                                                                                       |
// | TestGetContextThread/entries oldest first             | 15                                                                                                                                       |
// | TestGetContextThread/latest 20 entries                | 16                                                                                                                                       |
// | TestGetContextThread/unknown thread                   | 17                                                                                                                                       |
// | TestGetContextThreadErrors/store get fails            | 18                                                                                                                                       |
// | No test                                               | Entry count outside 1 to 10 (ContextToolCallEntryCountError): the relay rejects it first; json.Marshal errors: plain structs cannot fail |

package service_test

import . "github.com/ngo/precedent_mcp/internal/service"

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fixedTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

var (
	contextThreadA = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	contextThreadB = uuid.MustParse("11111111-1111-1111-1111-111111111111")
)

var contextTenEntries = []model.ContextEntry{
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
}

var contextTenObjects = []dbmodel.ContextObject{
	{ThreadID: contextThreadA.String(), Content: "entry 0"},
	{ThreadID: contextThreadA.String(), Content: "entry 1"},
	{ThreadID: contextThreadA.String(), Content: "entry 2"},
	{ThreadID: contextThreadA.String(), Content: "entry 3"},
	{ThreadID: contextThreadA.String(), Content: "entry 4"},
	{ThreadID: contextThreadA.String(), Content: "entry 5"},
	{ThreadID: contextThreadA.String(), Content: "entry 6"},
	{ThreadID: contextThreadA.String(), Content: "entry 7"},
	{ThreadID: contextThreadA.String(), Content: "entry 8"},
	{ThreadID: contextThreadA.String(), Content: "entry 9"},
}

var threadTwentyNewestFirst = []dbmodel.ContextObject{
	{ThreadID: contextThreadA.String(), Content: "entry 19", CreatedAt: fixedTime.Add(19 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 18", CreatedAt: fixedTime.Add(18 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 17", CreatedAt: fixedTime.Add(17 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 16", CreatedAt: fixedTime.Add(16 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 15", CreatedAt: fixedTime.Add(15 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 14", CreatedAt: fixedTime.Add(14 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 13", CreatedAt: fixedTime.Add(13 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 12", CreatedAt: fixedTime.Add(12 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 11", CreatedAt: fixedTime.Add(11 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 10", CreatedAt: fixedTime.Add(10 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 9", CreatedAt: fixedTime.Add(9 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 8", CreatedAt: fixedTime.Add(8 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 7", CreatedAt: fixedTime.Add(7 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 6", CreatedAt: fixedTime.Add(6 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 5", CreatedAt: fixedTime.Add(5 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 4", CreatedAt: fixedTime.Add(4 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 3", CreatedAt: fixedTime.Add(3 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 2", CreatedAt: fixedTime.Add(2 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 1", CreatedAt: fixedTime.Add(1 * time.Minute)},
	{ThreadID: contextThreadA.String(), Content: "entry 0", CreatedAt: fixedTime},
}

var threadTwentyOldestFirst = []model.ThreadContext{
	{ThreadID: contextThreadA.String(), Context: "entry 0", CreatedAt: fixedTime},
	{ThreadID: contextThreadA.String(), Context: "entry 1", CreatedAt: fixedTime.Add(1 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 2", CreatedAt: fixedTime.Add(2 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 3", CreatedAt: fixedTime.Add(3 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 4", CreatedAt: fixedTime.Add(4 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 5", CreatedAt: fixedTime.Add(5 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 6", CreatedAt: fixedTime.Add(6 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 7", CreatedAt: fixedTime.Add(7 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 8", CreatedAt: fixedTime.Add(8 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 9", CreatedAt: fixedTime.Add(9 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 10", CreatedAt: fixedTime.Add(10 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 11", CreatedAt: fixedTime.Add(11 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 12", CreatedAt: fixedTime.Add(12 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 13", CreatedAt: fixedTime.Add(13 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 14", CreatedAt: fixedTime.Add(14 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 15", CreatedAt: fixedTime.Add(15 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 16", CreatedAt: fixedTime.Add(16 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 17", CreatedAt: fixedTime.Add(17 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 18", CreatedAt: fixedTime.Add(18 * time.Minute)},
	{ThreadID: contextThreadA.String(), Context: "entry 19", CreatedAt: fixedTime.Add(19 * time.Minute)},
}

func TestAddContext(t *testing.T) {
	tests := []struct {
		name                string
		givenRequest        model.AddContextRequest
		givenFailedContexts []string
		wantObjects         []dbmodel.ContextObject
		want                model.AddContextResult
	}{
		{
			name: "entries in two threads",
			givenRequest: model.AddContextRequest{Entries: []model.ContextEntry{
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
				{ThreadID: contextThreadB, Context: "Constraint: EU region"},
				{ThreadID: contextThreadA, Context: "Finding: lint passes"},
			}},
			wantObjects: []dbmodel.ContextObject{
				{ThreadID: contextThreadA.String(), Content: "Preference: tabs"},
				{ThreadID: contextThreadB.String(), Content: "Constraint: EU region"},
				{ThreadID: contextThreadA.String(), Content: "Finding: lint passes"},
			},
			want: model.AddContextResult{Succeeded: []int{0, 1, 2}, Failed: []int{}},
		},
		{
			name: "store fails one entry",
			givenRequest: model.AddContextRequest{Entries: []model.ContextEntry{
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
				{ThreadID: contextThreadB, Context: "Constraint: EU region"},
				{ThreadID: contextThreadA, Context: "Finding: lint passes"},
			}},
			givenFailedContexts: []string{"Constraint: EU region"},
			wantObjects: []dbmodel.ContextObject{
				{ThreadID: contextThreadA.String(), Content: "Preference: tabs"},
				{ThreadID: contextThreadB.String(), Content: "Constraint: EU region"},
				{ThreadID: contextThreadA.String(), Content: "Finding: lint passes"},
			},
			want: model.AddContextResult{Succeeded: []int{0, 2}, Failed: []int{1}},
		},
		{
			name: "store fails every entry",
			givenRequest: model.AddContextRequest{Entries: []model.ContextEntry{
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
				{ThreadID: contextThreadB, Context: "Constraint: EU region"},
				{ThreadID: contextThreadA, Context: "Finding: lint passes"},
			}},
			givenFailedContexts: []string{"Preference: tabs", "Constraint: EU region", "Finding: lint passes"},
			wantObjects: []dbmodel.ContextObject{
				{ThreadID: contextThreadA.String(), Content: "Preference: tabs"},
				{ThreadID: contextThreadB.String(), Content: "Constraint: EU region"},
				{ThreadID: contextThreadA.String(), Content: "Finding: lint passes"},
			},
			want: model.AddContextResult{Succeeded: []int{}, Failed: []int{0, 1, 2}},
		},
		{
			name:         "minimum entry count",
			givenRequest: model.AddContextRequest{Entries: []model.ContextEntry{{ThreadID: contextThreadA, Context: "Preference: tabs"}}},
			wantObjects:  []dbmodel.ContextObject{{ThreadID: contextThreadA.String(), Content: "Preference: tabs"}},
			want:         model.AddContextResult{Succeeded: []int{0}, Failed: []int{}},
		},
		{
			name:         "maximum entry count",
			givenRequest: model.AddContextRequest{Entries: contextTenEntries},
			wantObjects:  contextTenObjects,
			want:         model.AddContextResult{Succeeded: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, Failed: []int{}},
		},
		{
			name: "repeated entry saved",
			givenRequest: model.AddContextRequest{Entries: []model.ContextEntry{
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
				{ThreadID: contextThreadA, Context: "Finding: lint passes"},
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
			}},
			wantObjects: []dbmodel.ContextObject{
				{ThreadID: contextThreadA.String(), Content: "Preference: tabs"},
				{ThreadID: contextThreadA.String(), Content: "Finding: lint passes"},
			},
			want: model.AddContextResult{Succeeded: []int{0, 1, 2}, Failed: []int{}},
		},
		{
			name: "repeated entry failed",
			givenRequest: model.AddContextRequest{Entries: []model.ContextEntry{
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
				{ThreadID: contextThreadA, Context: "Preference: tabs"},
			}},
			givenFailedContexts: []string{"Preference: tabs"},
			wantObjects:         []dbmodel.ContextObject{{ThreadID: contextThreadA.String(), Content: "Preference: tabs"}},
			want:                model.AddContextResult{Succeeded: []int{}, Failed: []int{0, 1}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			var gotObjects []dbmodel.ContextObject
			store.EXPECT().Create(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, objects []dbmodel.ContextObject) ([]uuid.UUID, []uuid.UUID, error) {
					gotObjects = objects
					var succeeded, failed []uuid.UUID
					for _, o := range objects {
						if slices.Contains(test.givenFailedContexts, o.Content) {
							failed = append(failed, o.ID)
						} else {
							succeeded = append(succeeded, o.ID)
						}
					}
					return succeeded, failed, nil
				})
			s := NewToolService(nil, store, nil)

			res := s.AddContext(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.AddContextResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			require.Len(t, gotObjects, len(test.wantObjects))
			wantObjects := slices.Clone(test.wantObjects)
			for i := range wantObjects {
				wantObjects[i].ID, wantObjects[i].CreatedAt = gotObjects[i].ID, gotObjects[i].CreatedAt
			}
			assert.Equal(t, wantObjects, gotObjects)
			assert.NotNil(t, got.Succeeded)
			assert.NotNil(t, got.Failed)
			assert.ElementsMatch(t, test.want.Succeeded, got.Succeeded)
			assert.ElementsMatch(t, test.want.Failed, got.Failed)
		})
	}
}

func TestAddContextErrors(t *testing.T) {
	tests := []struct {
		name             string
		givenRequest     model.AddContextRequest
		givenCreateError error
		want             model.Error
	}{
		{"store create fails", model.AddContextRequest{Entries: []model.ContextEntry{{ThreadID: contextThreadA, Context: "Preference: tabs"}}},
			errors.New("weaviate down"), model.Error{Code: env.ServiceErrorInternal, Message: "weaviate down"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			store.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, nil, test.givenCreateError)
			s := NewToolService(nil, store, nil)

			res := s.AddContext(context.Background(), test.givenRequest)

			require.True(t, res.GetIsError())
			var got model.Error
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRetrieveContextLimit(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest model.RetrieveContextRequest
		wantLimit    int
		want         model.RetrieveContextResult
	}{
		{"default when top_n is absent", model.RetrieveContextRequest{Query: "tabs"}, 20,
			model.RetrieveContextResult{Results: []model.RankedContext{}}},
		{"minimum top_n", model.RetrieveContextRequest{Query: "tabs", TopN: new(1)}, 1,
			model.RetrieveContextResult{Results: []model.RankedContext{}}},
		{"top_n inside the range", model.RetrieveContextRequest{Query: "tabs", TopN: new(5)}, 5,
			model.RetrieveContextResult{Results: []model.RankedContext{}}},
		{"maximum top_n", model.RetrieveContextRequest{Query: "tabs", TopN: new(20)}, 20,
			model.RetrieveContextResult{Results: []model.RankedContext{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			store.EXPECT().Search(mock.Anything, test.givenRequest.Query, test.wantLimit).Return(nil, nil)
			s := NewToolService(nil, store, nil)

			res := s.RetrieveContext(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.RetrieveContextResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRetrieveContext(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest model.RetrieveContextRequest
		givenFound   []dbmodel.ContextSearch
		want         model.RetrieveContextResult
	}{
		{
			name:         "hits keep the store order",
			givenRequest: model.RetrieveContextRequest{Query: "tabs"},
			givenFound: []dbmodel.ContextSearch{
				{ThreadID: contextThreadA.String(), Content: "user prefers tabs", Metadata: dbmodel.ContextMetadata{Score: 0.9}},
				{ThreadID: contextThreadB.String(), Content: "editor is vim", Metadata: dbmodel.ContextMetadata{Score: 0.4}},
			},
			want: model.RetrieveContextResult{Results: []model.RankedContext{
				{ThreadID: contextThreadA.String(), Context: "user prefers tabs", Score: 0.9},
				{ThreadID: contextThreadB.String(), Context: "editor is vim", Score: 0.4},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			store.EXPECT().Search(mock.Anything, test.givenRequest.Query, mock.Anything).Return(test.givenFound, nil)
			s := NewToolService(nil, store, nil)

			res := s.RetrieveContext(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.RetrieveContextResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRetrieveContextErrors(t *testing.T) {
	tests := []struct {
		name             string
		givenRequest     model.RetrieveContextRequest
		givenSearchError error
		want             model.Error
	}{
		{"store search fails", model.RetrieveContextRequest{Query: "tabs"}, errors.New("weaviate down"),
			model.Error{Code: env.ServiceErrorInternal, Message: "weaviate down"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			store.EXPECT().Search(mock.Anything, test.givenRequest.Query, mock.Anything).Return(nil, test.givenSearchError)
			s := NewToolService(nil, store, nil)

			res := s.RetrieveContext(context.Background(), test.givenRequest)

			require.True(t, res.GetIsError())
			var got model.Error
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGetContextThread(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest model.GetContextThreadRequest
		givenFound   []dbmodel.ContextObject
		wantLimit    int
		want         model.GetContextThreadResult
	}{
		{
			name:         "entries oldest first",
			givenRequest: model.GetContextThreadRequest{ThreadID: contextThreadA},
			givenFound: []dbmodel.ContextObject{
				{ThreadID: contextThreadA.String(), Content: "third", CreatedAt: fixedTime.Add(2 * time.Minute)},
				{ThreadID: contextThreadA.String(), Content: "second", CreatedAt: fixedTime.Add(time.Minute)},
				{ThreadID: contextThreadA.String(), Content: "first", CreatedAt: fixedTime},
			},
			wantLimit: 20,
			want: model.GetContextThreadResult{ThreadID: contextThreadA, Entries: []model.ThreadContext{
				{ThreadID: contextThreadA.String(), Context: "first", CreatedAt: fixedTime},
				{ThreadID: contextThreadA.String(), Context: "second", CreatedAt: fixedTime.Add(time.Minute)},
				{ThreadID: contextThreadA.String(), Context: "third", CreatedAt: fixedTime.Add(2 * time.Minute)},
			}},
		},
		{
			name:         "latest 20 entries",
			givenRequest: model.GetContextThreadRequest{ThreadID: contextThreadA},
			givenFound:   threadTwentyNewestFirst,
			wantLimit:    20,
			want:         model.GetContextThreadResult{ThreadID: contextThreadA, Entries: threadTwentyOldestFirst},
		},
		{
			name:         "unknown thread",
			givenRequest: model.GetContextThreadRequest{ThreadID: contextThreadB},
			givenFound:   nil,
			wantLimit:    20,
			want:         model.GetContextThreadResult{ThreadID: contextThreadB, Entries: []model.ThreadContext{}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			store.EXPECT().Get(mock.Anything, test.givenRequest.ThreadID.String(), test.wantLimit).Return(test.givenFound, nil)
			s := NewToolService(nil, store, nil)

			res := s.GetContextThread(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.GetContextThreadResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGetContextThreadErrors(t *testing.T) {
	tests := []struct {
		name          string
		givenRequest  model.GetContextThreadRequest
		givenGetError error
		want          model.Error
	}{
		{"store get fails", model.GetContextThreadRequest{ThreadID: contextThreadA}, errors.New("weaviate down"),
			model.Error{Code: env.ServiceErrorInternal, Message: "weaviate down"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockContextStore(t)
			store.EXPECT().Get(mock.Anything, test.givenRequest.ThreadID.String(), mock.Anything).Return(nil, test.givenGetError)
			s := NewToolService(nil, store, nil)

			res := s.GetContextThread(context.Background(), test.givenRequest)

			require.True(t, res.GetIsError())
			var got model.Error
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}
