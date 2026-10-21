// Test plan: tool_decision_response.go
//
// Logic flow
// | Function                                   | Step | Action                                                        | Effect                              |
// |--------------------------------------------|------|---------------------------------------------------------------|-------------------------------------|
// | RetrieveDecisionsResult.FromDecisionSearch | 1    | Start from an empty, non-nil list                             | No hits give {"decisions":[]}       |
// | RetrieveDecisionsResult.FromDecisionSearch | 2    | Copy every stored field of each hit in store order            | Score is left for ranking           |
// | RetrieveDecisionsResult.FromDecisionSearch | 3    | Decode Content into Details and Review into the latest review | An unreviewed hit has no review key |
//
// Scenarios
// | Number | Function                                   | Result  | Value class    | Input                                         | Expected                                                                                                  |
// |--------|--------------------------------------------|---------|----------------|-----------------------------------------------|-----------------------------------------------------------------------------------------------------------|
// | 1      | RetrieveDecisionsResult.FromDecisionSearch | Success | Middle value   | Reviewed hit with every field; unreviewed hit | Store order; every field copied; review only on the first; JSON matches the tools.yaml shape (steps 2, 3) |
// | 2      | RetrieveDecisionsResult.FromDecisionSearch | Success | Boundary value | No hits                                       | {"decisions":[]} (step 1)                                                                                 |
//
// Branch coverage
// | Function                                   | Condition        | True      | False      |
// |--------------------------------------------|------------------|-----------|------------|
// | RetrieveDecisionsResult.FromDecisionSearch | Hits found       | 1         | 2          |
// | RetrieveDecisionsResult.FromDecisionSearch | Hit has a review | 1 (first) | 1 (second) |
//
// Test mapping
// | Test                                                | Scenarios                                                                                                                                                                                                                                                  |
// |-----------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | TestFromDecisionSearch/reviewed and unreviewed hits | 1                                                                                                                                                                                                                                                          |
// | TestFromDecisionSearch/no hits                      | 2                                                                                                                                                                                                                                                          |
// | No test                                             | Error constructors: proven by ToolService scenarios 3, 4, 14, 17 to 19 in tool_decision_test.go; DecisionToolCallInvalidArgumentErrorResponse: only reachable through a conversion that cannot fail; unreadable YAML: only data written outside the server |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	supersededDecisionID = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	hitCreatedAt         = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	hitReviewedAt        = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
)

func TestFromDecisionSearch(t *testing.T) {
	tests := []struct {
		name       string
		givenFound []dbmodel.DecisionSearch
		want       RetrieveDecisionsResult
		wantJSON   string
	}{
		{
			name: "reviewed and unreviewed hits",
			givenFound: []dbmodel.DecisionSearch{
				{
					Title:      "Use Weaviate",
					Status:     "accepted",
					Tags:       []string{"storage"},
					Supersedes: &supersededDecisionID,
					Content:    "situation: s\ndecision: d\nrationale: r\n",
					Review:     "score: 4\nreason: worked\nreviewed_at: 2026-10-02T12:00:00Z\n",
					CreatedAt:  hitCreatedAt,
					Metadata:   dbmodel.DecisionMetadata{ID: decisionID, Score: 0.8},
				},
				{
					Title:     "Use Postgres",
					Status:    "proposed",
					Content:   "situation: s\ndecision: d\nrationale: r\n",
					CreatedAt: hitCreatedAt,
					Metadata:  dbmodel.DecisionMetadata{ID: otherDecisionID, Score: 0.5},
				},
			},
			want: RetrieveDecisionsResult{Decisions: []RankedDecision{
				{
					DecisionID: decisionID,
					Title:      "Use Weaviate",
					Status:     env.StatusAccepted,
					Tags:       []string{"storage"},
					Supersedes: &supersededDecisionID,
					CreatedAt:  hitCreatedAt,
					Details:    DecisionDetails{Situation: "s", Decision: "d", Rationale: "r"},
					Review:     &DecisionReview{Score: 4, Reason: "worked", ReviewedAt: hitReviewedAt},
				},
				{
					DecisionID: otherDecisionID,
					Title:      "Use Postgres",
					Status:     env.StatusProposed,
					CreatedAt:  hitCreatedAt,
					Details:    DecisionDetails{Situation: "s", Decision: "d", Rationale: "r"},
				},
			}},
			wantJSON: `{"decisions":[` +
				`{"decision_id":"33333333-3333-3333-3333-333333333333","title":"Use Weaviate","status":"accepted","tags":["storage"],` +
				`"supersedes":"55555555-5555-5555-5555-555555555555","created_at":"2026-10-01T12:00:00Z",` +
				`"Details":{"situation":"s","decision":"d","rationale":"r"},` +
				`"review":{"score":4,"reason":"worked","reviewed_at":"2026-10-02T12:00:00Z"},"score":0},` +
				`{"decision_id":"44444444-4444-4444-4444-444444444444","title":"Use Postgres","status":"proposed",` +
				`"created_at":"2026-10-01T12:00:00Z","Details":{"situation":"s","decision":"d","rationale":"r"},"score":0}]}`,
		},
		{
			name:       "no hits",
			givenFound: nil,
			want:       RetrieveDecisionsResult{Decisions: []RankedDecision{}},
			wantJSON:   `{"decisions":[]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got RetrieveDecisionsResult

			err := got.FromDecisionSearch(test.givenFound)

			require.NoError(t, err)
			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.JSONEq(t, test.wantJSON, string(gotJSON))
		})
	}
}
