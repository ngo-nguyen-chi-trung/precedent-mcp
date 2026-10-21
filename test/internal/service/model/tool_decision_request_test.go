// Test plan: tool_decision_request.go
//
// Logic flow
// | Function                               | Step | Action                                                          | Effect                                         |
// |----------------------------------------|------|-----------------------------------------------------------------|------------------------------------------------|
// | RecordDecisionRequest.ToDecisionObject | 1    | New random ID for each call                                     | Every recorded decision is a new decision      |
// | RecordDecisionRequest.ToDecisionObject | 2    | Copy title, status, tags, supersedes; Content = YAML of Details | Details survive the round trip through storage |
// | RecordDecisionRequest.ToDecisionObject | 3    | CreatedAt = now in UTC                                          | Creation time is stored in one time zone       |
// | ReviewDecisionRequest.ToDecisionObject | 1    | Review = YAML of score, reason, reviewed_at now in UTC          | The latest review only                         |
// | ReviewDecisionRequest.ToDecisionObject | 2    | Return an object holding only Review                            | A merge update touches no other field          |
//
// Scenarios
// | Number | Function                               | Result  | Value class    | Input                                                                 | Expected                                                                                                      |
// |--------|----------------------------------------|---------|----------------|-----------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------|
// | 1      | RecordDecisionRequest.ToDecisionObject | Success | Middle value   | Accepted; tags; supersedes; every Details field                       | Fields copied; Content decodes to the same Details; CreatedAt UTC inside the call window (steps 1 to 3)       |
// | 2      | RecordDecisionRequest.ToDecisionObject | Success | Boundary value | Proposed; only situation, decision, rationale; no tags; no supersedes | Content decodes to the three fields; Tags and Supersedes absent (step 2)                                      |
// | 3      | RecordDecisionRequest.ToDecisionObject | Success | Middle value   | Same request twice                                                    | Two different IDs (step 1)                                                                                    |
// | 4      | ReviewDecisionRequest.ToDecisionObject | Success | Boundary value | Score 1                                                               | Object holds only Review; decodes to score 1, the reason, reviewed_at UTC inside the call window (steps 1, 2) |
// | 5      | ReviewDecisionRequest.ToDecisionObject | Success | Middle value   | Score 3                                                               | Review decodes to score 3 (step 1)                                                                            |
// | 6      | ReviewDecisionRequest.ToDecisionObject | Success | Boundary value | Score 5                                                               | Review decodes to score 5 (step 1)                                                                            |
//
// Failure scenarios: none. yaml.Marshal of these plain structs cannot fail.
//
// Branch coverage
// | Function                               | Condition             | True | False |
// |----------------------------------------|-----------------------|------|-------|
// | RecordDecisionRequest.ToDecisionObject | Optional fields given | 1    | 2     |
// | ReviewDecisionRequest.ToDecisionObject | None                  | —    | —     |
//
// Test mapping
// | Test                                                      | Scenarios                                      |
// |-----------------------------------------------------------|------------------------------------------------|
// | TestRecordDecisionToDecisionObject/every field given      | 1                                              |
// | TestRecordDecisionToDecisionObject/required details only  | 2                                              |
// | TestRecordDecisionNewIdentifierPerCall/same request twice | 3                                              |
// | TestReviewDecisionToDecisionObject/minimum score          | 4                                              |
// | TestReviewDecisionToDecisionObject/middle score           | 5                                              |
// | TestReviewDecisionToDecisionObject/maximum score          | 6                                              |
// | No test                                                   | yaml.Marshal errors: plain structs cannot fail |

package model_test

import . "github.com/ngo/precedent_mcp/internal/service/model"

import (
	"testing"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

var (
	decisionID      = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherDecisionID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
)

var fullDetails = DecisionDetails{
	Situation: "Agents need shared recall",
	Decision:  "Store memory in Weaviate",
	Rationale: "Hybrid search out of the box",
	Options:   []DecisionOption{{Name: "Weaviate", Pros: []string{"hybrid search"}, Cons: []string{"one more service"}, Verdict: "chosen"}},
	Tradeoffs: []string{"one more service to run"},
	Facts:     []SourcedNote{{Text: "supports BM25", Source: "docs"}},
	Practices: []SourcedNote{{Text: "keep schemas small"}},
}

var requiredDetails = DecisionDetails{
	Situation: "Agents need shared recall",
	Decision:  "Store memory in Weaviate",
	Rationale: "Hybrid search out of the box",
}

func TestRecordDecisionToDecisionObject(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest RecordDecisionRequest
		want         dbmodel.DecisionObject
		wantDetails  DecisionDetails
		wantLocation *time.Location
	}{
		{
			name: "every field given",
			givenRequest: RecordDecisionRequest{
				Title:      "Use Weaviate for memory",
				Status:     env.StatusAccepted,
				Tags:       []string{"storage", "search"},
				Supersedes: &otherDecisionID,
				Details:    fullDetails,
			},
			want: dbmodel.DecisionObject{
				Title:      "Use Weaviate for memory",
				Status:     "accepted",
				Tags:       []string{"storage", "search"},
				Supersedes: &otherDecisionID,
			},
			wantDetails:  fullDetails,
			wantLocation: time.UTC,
		},
		{
			name: "required details only",
			givenRequest: RecordDecisionRequest{
				Title:   "Use Weaviate for memory",
				Status:  env.StatusProposed,
				Details: requiredDetails,
			},
			want: dbmodel.DecisionObject{
				Title:  "Use Weaviate for memory",
				Status: "proposed",
			},
			wantDetails:  requiredDetails,
			wantLocation: time.UTC,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := time.Now().UTC()

			got, err := test.givenRequest.ToDecisionObject()

			after := time.Now().UTC()
			require.NoError(t, err)
			var gotDetails DecisionDetails
			require.NoError(t, yaml.Unmarshal([]byte(got.Content), &gotDetails))
			assert.WithinRange(t, got.CreatedAt, before, after)
			assert.Equal(t, test.wantLocation, got.CreatedAt.Location())
			test.want.ID, test.want.Content, test.want.CreatedAt = got.ID, got.Content, got.CreatedAt
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantDetails, gotDetails)
		})
	}
}

func TestRecordDecisionNewIdentifierPerCall(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest RecordDecisionRequest
	}{
		{"same request twice", RecordDecisionRequest{Title: "Use Weaviate for memory", Status: env.StatusProposed, Details: requiredDetails}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first, firstErr := test.givenRequest.ToDecisionObject()
			second, secondErr := test.givenRequest.ToDecisionObject()

			require.NoError(t, firstErr)
			require.NoError(t, secondErr)
			assert.NotEqual(t, first.ID, second.ID)
		})
	}
}

func TestReviewDecisionToDecisionObject(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest ReviewDecisionRequest
		want         dbmodel.DecisionObject
		wantReview   DecisionReview
		wantLocation *time.Location
	}{
		{"minimum score", ReviewDecisionRequest{DecisionID: decisionID, Score: 1, Reason: "rejected by the user"},
			dbmodel.DecisionObject{}, DecisionReview{Score: 1, Reason: "rejected by the user"}, time.UTC},
		{"middle score", ReviewDecisionRequest{DecisionID: decisionID, Score: 3, Reason: "adjusted by the user"},
			dbmodel.DecisionObject{}, DecisionReview{Score: 3, Reason: "adjusted by the user"}, time.UTC},
		{"maximum score", ReviewDecisionRequest{DecisionID: decisionID, Score: 5, Reason: "worked as decided"},
			dbmodel.DecisionObject{}, DecisionReview{Score: 5, Reason: "worked as decided"}, time.UTC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := time.Now().UTC()

			got, err := test.givenRequest.ToDecisionObject()

			after := time.Now().UTC()
			require.NoError(t, err)
			var gotReview DecisionReview
			require.NoError(t, yaml.Unmarshal([]byte(got.Review), &gotReview))
			assert.WithinRange(t, gotReview.ReviewedAt, before, after)
			assert.Equal(t, test.wantLocation, gotReview.ReviewedAt.Location())
			test.want.Review = got.Review
			test.wantReview.ReviewedAt = gotReview.ReviewedAt
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantReview, gotReview)
		})
	}
}
