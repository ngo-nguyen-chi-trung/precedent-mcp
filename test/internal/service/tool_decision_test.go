// Test plan: tool_decision.go
//
// Logic flow
// | Function                      | Step | Action                                                                               | Effect                                                                                                                        |
// |-------------------------------|------|--------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------|
// | ToolService.RecordDecision    | 1    | RecordDecisionRequest.ToDecisionObject                                               | A new decision with a new ID                                                                                                  |
// | ToolService.RecordDecision    | 2    | DecisionStore.Create(new decision)                                                   | Failure → is_error internal "create decision <new>: <cause>"; old decision untouched                                          |
// | ToolService.RecordDecision    | 3    | supersedes given → DecisionStore.Update(old, status superseded, superseded_by new)   | Failure → new decision stays stored; is_error internal "decision <new> created, but supersede decision <old> failed: <cause>" |
// | ToolService.RecordDecision    | 4    | Return {"decision_id": new}                                                          | Success                                                                                                                       |
// | ToolService.RetrieveDecisions | 1    | limit = top_n; LimitTopN 20 when absent                                              | Never more than 20 decisions                                                                                                  |
// | ToolService.RetrieveDecisions | 2    | DecisionStore.Search(query, limit)                                                   | Failure → is_error internal with the cause                                                                                    |
// | ToolService.RetrieveDecisions | 3    | Fuse match rank (weight 2) and review rank (weight 3); unreviewed counts as review 3 | Review rank weighs more than match rank; score from 0 to 1, 1 means first by match and by review                              |
// | ToolService.RetrieveDecisions | 4    | Return decisions with the highest score first                                        | Score only compares decisions within this search                                                                              |
// | ToolService.ReviewDecision    | 1    | DecisionStore.Get(decision_id)                                                       | Failure → is_error internal                                                                                                   |
// | ToolService.ReviewDecision    | 2    | Decision missing                                                                     | is_error decision_not_found "decision not found: <id>"; nothing stored                                                        |
// | ToolService.ReviewDecision    | 3    | DecisionStore.Update(decision_id, object holding only the new review)                | The latest review replaces an earlier one; failure → is_error internal                                                        |
// | ToolService.ReviewDecision    | 4    | Return {"decision_id"}                                                               | Success                                                                                                                       |
//
// Scenarios
// | Number | Function                      | Result  | Value class    | Input                                           | Expected                                                                                |
// |--------|-------------------------------|---------|----------------|-------------------------------------------------|-----------------------------------------------------------------------------------------|
// | 1      | ToolService.RecordDecision    | Success | Middle value   | Accepted request without supersedes             | Create receives the converted decision; data holds its ID; no Update (steps 1, 2, 4)    |
// | 2      | ToolService.RecordDecision    | Success | Middle value   | Proposed request superseding an old decision    | Create, then Update(old, superseded, superseded_by new); data holds the new ID (step 3) |
// | 3      | ToolService.RecordDecision    | Failure | Middle value   | Create returns "weaviate down"                  | is_error internal "create decision <new>: weaviate down"; no Update (step 2)            |
// | 4      | ToolService.RecordDecision    | Failure | Middle value   | Update returns "weaviate down"                  | is_error internal naming new, old and "weaviate down"; Create already done (step 3)     |
// | 5      | ToolService.RetrieveDecisions | Success | Boundary value | top_n absent                                    | Search limit 20 (step 1)                                                                |
// | 6      | ToolService.RetrieveDecisions | Success | Boundary value | top_n 1                                         | Search limit 1 (step 1)                                                                 |
// | 7      | ToolService.RetrieveDecisions | Success | Middle value   | top_n 3                                         | Search limit 3 (step 1)                                                                 |
// | 8      | ToolService.RetrieveDecisions | Success | Boundary value | top_n 20                                        | Search limit 20 (step 1)                                                                |
// | 9      | ToolService.RetrieveDecisions | Success | Boundary value | One unreviewed hit                              | Score 1, the maximum; no review key (step 3)                                            |
// | 10     | ToolService.RetrieveDecisions | Success | Middle value   | Match order A, B, C; reviews 1, none, 5         | Order C, B, A; C 311/315, B 61/62, A 309/315 (steps 3, 4)                               |
// | 11     | ToolService.RetrieveDecisions | Success | Boundary value | Match order A, B; A review 2, B unreviewed      | Order B, A; B 308/310, A 307/310; a review below neutral loses (step 3)                 |
// | 12     | ToolService.RetrieveDecisions | Success | Boundary value | Match order A, B; A unreviewed, B review 3      | Order A, B; A 1, B 61/62; review 3 ranks like unreviewed (step 3)                       |
// | 13     | ToolService.RetrieveDecisions | Success | Boundary value | Match order A, B; A unreviewed, B review 4      | Order B, A; B 308/310, A 307/310; a review above neutral wins (step 3)                  |
// | 14     | ToolService.RetrieveDecisions | Failure | Middle value   | Search returns "weaviate down"                  | is_error internal "weaviate down" (step 2)                                              |
// | 15     | ToolService.ReviewDecision    | Success | Middle value   | Decision without a review                       | Update holds only the converted review; data holds decision_id (steps 3, 4)             |
// | 16     | ToolService.ReviewDecision    | Success | Middle value   | Decision with an earlier review 2; new review 4 | Update holds only review 4; the earlier review is not carried over (step 3)             |
// | 17     | ToolService.ReviewDecision    | Failure | Middle value   | Unknown decision_id                             | is_error decision_not_found "decision not found: <id>"; no Update (step 2)              |
// | 18     | ToolService.ReviewDecision    | Failure | Middle value   | Get returns "weaviate down"                     | is_error internal "weaviate down"; no Update (step 1)                                   |
// | 19     | ToolService.ReviewDecision    | Failure | Middle value   | Update returns "weaviate down"                  | is_error internal "weaviate down" (step 3)                                              |
//
// Outside the handler contract: top_n below 1 or above 20, empty query, title or reason, score outside 1 to 5, and status other than proposed or accepted; the relay Arguments step rejects them with tools.yaml and e2e covers that.
//
// Outside the handler: DecisionStore.Search excludes superseded decisions; e2e covers that.
//
// Branch coverage
// | Function                      | Condition                             | True       | False   |
// |-------------------------------|---------------------------------------|------------|---------|
// | ToolService.RecordDecision    | supersedes given                      | 2, 4       | 1, 3    |
// | ToolService.RecordDecision    | Create fails                          | 3          | 1, 2, 4 |
// | ToolService.RecordDecision    | Update fails                          | 4          | 2       |
// | ToolService.RetrieveDecisions | top_n given                           | 6, 7, 8    | 5       |
// | ToolService.RetrieveDecisions | Search fails                          | 14         | 9 to 13 |
// | ToolService.RetrieveDecisions | Review order differs from match order | 10, 11, 13 | 12      |
// | ToolService.ReviewDecision    | Get fails                             | 18         | 15      |
// | ToolService.ReviewDecision    | Decision missing                      | 17         | 15      |
// | ToolService.ReviewDecision    | Earlier review exists                 | 16         | 15      |
// | ToolService.ReviewDecision    | Update fails                          | 19         | 15, 16  |
//
// Test mapping
// | Test                                                               | Scenarios                                                                                                                              |
// |--------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
// | TestRecordDecision/accepted decision without supersedes            | 1                                                                                                                                      |
// | TestRecordDecision/proposed decision supersedes an old one         | 2                                                                                                                                      |
// | TestRecordDecisionErrors/store create fails                        | 3                                                                                                                                      |
// | TestRecordDecisionErrors/supersede fails after create              | 4                                                                                                                                      |
// | TestRetrieveDecisionsLimit/default when top_n is absent            | 5                                                                                                                                      |
// | TestRetrieveDecisionsLimit/minimum top_n                           | 6                                                                                                                                      |
// | TestRetrieveDecisionsLimit/top_n inside the range                  | 7                                                                                                                                      |
// | TestRetrieveDecisionsLimit/maximum top_n                           | 8                                                                                                                                      |
// | TestRetrieveDecisionsRanking/unreviewed hit ranks as review 3      | 9                                                                                                                                      |
// | TestRetrieveDecisionsRanking/review order decides over match order | 10                                                                                                                                     |
// | TestRetrieveDecisionsRanking/review 2 ranks below unreviewed       | 11                                                                                                                                     |
// | TestRetrieveDecisionsRanking/review 3 ranks like unreviewed        | 12                                                                                                                                     |
// | TestRetrieveDecisionsRanking/review 4 ranks above unreviewed       | 13                                                                                                                                     |
// | TestRetrieveDecisionsErrors/store search fails                     | 14                                                                                                                                     |
// | TestReviewDecision/first review                                    | 15                                                                                                                                     |
// | TestReviewDecision/new review replaces the earlier one             | 16                                                                                                                                     |
// | TestReviewDecisionErrors/unknown decision                          | 17                                                                                                                                     |
// | TestReviewDecisionErrors/store get fails                           | 18                                                                                                                                     |
// | TestReviewDecisionErrors/store update fails                        | 19                                                                                                                                     |
// | No test                                                            | Stored Content or Review not YAML: only data written outside the server; conversion and json.Marshal errors: plain structs cannot fail |

package service_test

import . "github.com/ngo/precedent_mcp/internal/service"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

var (
	decisionA     = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	decisionB     = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002")
	decisionC     = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000003")
	oldDecisionID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

var requiredDetails = model.DecisionDetails{
	Situation: "Agents need shared recall",
	Decision:  "Store memory in Weaviate",
	Rationale: "Hybrid search out of the box",
}

const requiredDetailsYAML = "situation: Agents need shared recall\ndecision: Store memory in Weaviate\nrationale: Hybrid search out of the box\n"

func TestRecordDecision(t *testing.T) {
	tests := []struct {
		name           string
		givenRequest   model.RecordDecisionRequest
		wantObject     dbmodel.DecisionObject
		wantSupersedes *uuid.UUID
		wantUpdate     dbmodel.DecisionObject
		want           model.RecordDecisionResult
	}{
		{
			name: "accepted decision without supersedes",
			givenRequest: model.RecordDecisionRequest{
				Title:   "Use Weaviate for memory",
				Status:  env.StatusAccepted,
				Tags:    []string{"storage"},
				Details: requiredDetails,
			},
			wantObject: dbmodel.DecisionObject{
				Title:   "Use Weaviate for memory",
				Status:  "accepted",
				Tags:    []string{"storage"},
				Content: requiredDetailsYAML,
			},
		},
		{
			name: "proposed decision supersedes an old one",
			givenRequest: model.RecordDecisionRequest{
				Title:      "Use Weaviate for memory",
				Status:     env.StatusProposed,
				Supersedes: &oldDecisionID,
				Details:    requiredDetails,
			},
			wantObject: dbmodel.DecisionObject{
				Title:      "Use Weaviate for memory",
				Status:     "proposed",
				Supersedes: &oldDecisionID,
				Content:    requiredDetailsYAML,
			},
			wantSupersedes: &oldDecisionID,
			wantUpdate:     dbmodel.DecisionObject{Status: "superseded"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			var gotObject dbmodel.DecisionObject
			create := store.EXPECT().Create(mock.Anything, mock.Anything).
				Run(func(_ context.Context, d dbmodel.DecisionObject) { gotObject = d }).
				Return(nil)
			var gotUpdate dbmodel.DecisionObject
			if test.wantSupersedes != nil {
				store.EXPECT().Update(mock.Anything, *test.wantSupersedes, mock.Anything).
					Run(func(_ context.Context, _ uuid.UUID, d dbmodel.DecisionObject) { gotUpdate = d }).
					Return(nil).
					NotBefore(create.Call)
			}
			s := NewToolService(nil, nil, store)

			res := s.RecordDecision(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.RecordDecisionResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			test.want.DecisionID = gotObject.ID
			test.wantObject.ID, test.wantObject.CreatedAt = gotObject.ID, gotObject.CreatedAt
			if test.wantSupersedes != nil {
				test.wantUpdate.SupersededBy = &gotObject.ID
			}
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantObject, gotObject)
			assert.Equal(t, test.wantUpdate, gotUpdate)
		})
	}
}

func TestRecordDecisionErrors(t *testing.T) {
	tests := []struct {
		name             string
		givenRequest     model.RecordDecisionRequest
		givenCreateError error
		givenUpdateError error
		want             model.Error
	}{
		{
			name: "store create fails",
			givenRequest: model.RecordDecisionRequest{
				Title:      "Use Weaviate for memory",
				Status:     env.StatusProposed,
				Supersedes: &oldDecisionID,
				Details:    requiredDetails,
			},
			givenCreateError: errors.New("weaviate down"),
			want:             model.Error{Code: env.ServiceErrorInternal, Message: "create decision %s: weaviate down"},
		},
		{
			name: "supersede fails after create",
			givenRequest: model.RecordDecisionRequest{
				Title:      "Use Weaviate for memory",
				Status:     env.StatusProposed,
				Supersedes: &oldDecisionID,
				Details:    requiredDetails,
			},
			givenUpdateError: errors.New("weaviate down"),
			want: model.Error{
				Code:    env.ServiceErrorInternal,
				Message: "decision %s created, but supersede decision 33333333-3333-3333-3333-333333333333 failed: weaviate down",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			var gotObject dbmodel.DecisionObject
			create := store.EXPECT().Create(mock.Anything, mock.Anything).
				Run(func(_ context.Context, d dbmodel.DecisionObject) { gotObject = d }).
				Return(test.givenCreateError)
			if test.givenUpdateError != nil {
				store.EXPECT().Update(mock.Anything, *test.givenRequest.Supersedes, mock.Anything).
					Return(test.givenUpdateError).
					NotBefore(create.Call)
			}
			s := NewToolService(nil, nil, store)

			res := s.RecordDecision(context.Background(), test.givenRequest)

			require.True(t, res.GetIsError())
			var got model.Error
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			test.want.Message = fmt.Sprintf(test.want.Message, gotObject.ID)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRetrieveDecisionsLimit(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest model.RetrieveDecisionsRequest
		wantLimit    int
		want         model.RetrieveDecisionsResult
	}{
		{"default when top_n is absent", model.RetrieveDecisionsRequest{Query: "storage"}, 20,
			model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{}}},
		{"minimum top_n", model.RetrieveDecisionsRequest{Query: "storage", TopN: new(1)}, 1,
			model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{}}},
		{"top_n inside the range", model.RetrieveDecisionsRequest{Query: "storage", TopN: new(3)}, 3,
			model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{}}},
		{"maximum top_n", model.RetrieveDecisionsRequest{Query: "storage", TopN: new(20)}, 20,
			model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			store.EXPECT().Search(mock.Anything, test.givenRequest.Query, test.wantLimit).Return(nil, nil)
			s := NewToolService(nil, nil, store)

			res := s.RetrieveDecisions(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.RetrieveDecisionsResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRetrieveDecisionsRanking(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest model.RetrieveDecisionsRequest
		givenFound   []dbmodel.DecisionSearch
		want         model.RetrieveDecisionsResult
	}{
		{
			name:         "unreviewed hit ranks as review 3",
			givenRequest: model.RetrieveDecisionsRequest{Query: "storage"},
			givenFound: []dbmodel.DecisionSearch{
				{Title: "A", Status: "accepted", Metadata: dbmodel.DecisionMetadata{ID: decisionA, Score: 0.9}},
			},
			want: model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{
				{DecisionID: decisionA, Title: "A", Status: env.StatusAccepted, Score: 1},
			}},
		},
		{
			name:         "review order decides over match order",
			givenRequest: model.RetrieveDecisionsRequest{Query: "storage"},
			givenFound: []dbmodel.DecisionSearch{
				{Title: "A", Status: "accepted", Review: "score: 1\nreason: r\nreviewed_at: 2026-10-01T12:00:00Z\n",
					Metadata: dbmodel.DecisionMetadata{ID: decisionA, Score: 0.9}},
				{Title: "B", Status: "accepted", Metadata: dbmodel.DecisionMetadata{ID: decisionB, Score: 0.6}},
				{Title: "C", Status: "accepted", Review: "score: 5\nreason: r\nreviewed_at: 2026-10-01T12:00:00Z\n",
					Metadata: dbmodel.DecisionMetadata{ID: decisionC, Score: 0.3}},
			},
			want: model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{
				{DecisionID: decisionC, Title: "C", Status: env.StatusAccepted,
					Review: &model.DecisionReview{Score: 5, Reason: "r", ReviewedAt: fixedTime}, Score: 311.0 / 315},
				{DecisionID: decisionB, Title: "B", Status: env.StatusAccepted, Score: 61.0 / 62},
				{DecisionID: decisionA, Title: "A", Status: env.StatusAccepted,
					Review: &model.DecisionReview{Score: 1, Reason: "r", ReviewedAt: fixedTime}, Score: 309.0 / 315},
			}},
		},
		{
			name:         "review 2 ranks below unreviewed",
			givenRequest: model.RetrieveDecisionsRequest{Query: "storage"},
			givenFound: []dbmodel.DecisionSearch{
				{Title: "A", Status: "accepted", Review: "score: 2\nreason: r\nreviewed_at: 2026-10-01T12:00:00Z\n",
					Metadata: dbmodel.DecisionMetadata{ID: decisionA, Score: 0.9}},
				{Title: "B", Status: "accepted", Metadata: dbmodel.DecisionMetadata{ID: decisionB, Score: 0.6}},
			},
			want: model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{
				{DecisionID: decisionB, Title: "B", Status: env.StatusAccepted, Score: 308.0 / 310},
				{DecisionID: decisionA, Title: "A", Status: env.StatusAccepted,
					Review: &model.DecisionReview{Score: 2, Reason: "r", ReviewedAt: fixedTime}, Score: 307.0 / 310},
			}},
		},
		{
			name:         "review 3 ranks like unreviewed",
			givenRequest: model.RetrieveDecisionsRequest{Query: "storage"},
			givenFound: []dbmodel.DecisionSearch{
				{Title: "A", Status: "accepted", Metadata: dbmodel.DecisionMetadata{ID: decisionA, Score: 0.9}},
				{Title: "B", Status: "accepted", Review: "score: 3\nreason: r\nreviewed_at: 2026-10-01T12:00:00Z\n",
					Metadata: dbmodel.DecisionMetadata{ID: decisionB, Score: 0.6}},
			},
			want: model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{
				{DecisionID: decisionA, Title: "A", Status: env.StatusAccepted, Score: 1},
				{DecisionID: decisionB, Title: "B", Status: env.StatusAccepted,
					Review: &model.DecisionReview{Score: 3, Reason: "r", ReviewedAt: fixedTime}, Score: 61.0 / 62},
			}},
		},
		{
			name:         "review 4 ranks above unreviewed",
			givenRequest: model.RetrieveDecisionsRequest{Query: "storage"},
			givenFound: []dbmodel.DecisionSearch{
				{Title: "A", Status: "accepted", Metadata: dbmodel.DecisionMetadata{ID: decisionA, Score: 0.9}},
				{Title: "B", Status: "accepted", Review: "score: 4\nreason: r\nreviewed_at: 2026-10-01T12:00:00Z\n",
					Metadata: dbmodel.DecisionMetadata{ID: decisionB, Score: 0.6}},
			},
			want: model.RetrieveDecisionsResult{Decisions: []model.RankedDecision{
				{DecisionID: decisionB, Title: "B", Status: env.StatusAccepted,
					Review: &model.DecisionReview{Score: 4, Reason: "r", ReviewedAt: fixedTime}, Score: 308.0 / 310},
				{DecisionID: decisionA, Title: "A", Status: env.StatusAccepted, Score: 307.0 / 310},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			store.EXPECT().Search(mock.Anything, test.givenRequest.Query, mock.Anything).Return(test.givenFound, nil)
			s := NewToolService(nil, nil, store)

			res := s.RetrieveDecisions(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.RetrieveDecisionsResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			require.Len(t, got.Decisions, len(test.want.Decisions))
			for i := range got.Decisions {
				assert.InDelta(t, test.want.Decisions[i].Score, got.Decisions[i].Score, 1e-12)
				test.want.Decisions[i].Score = got.Decisions[i].Score
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRetrieveDecisionsErrors(t *testing.T) {
	tests := []struct {
		name             string
		givenRequest     model.RetrieveDecisionsRequest
		givenSearchError error
		want             model.Error
	}{
		{"store search fails", model.RetrieveDecisionsRequest{Query: "storage"}, errors.New("weaviate down"),
			model.Error{Code: env.ServiceErrorInternal, Message: "weaviate down"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			store.EXPECT().Search(mock.Anything, test.givenRequest.Query, mock.Anything).Return(nil, test.givenSearchError)
			s := NewToolService(nil, nil, store)

			res := s.RetrieveDecisions(context.Background(), test.givenRequest)

			require.True(t, res.GetIsError())
			var got model.Error
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestReviewDecision(t *testing.T) {
	tests := []struct {
		name         string
		givenRequest model.ReviewDecisionRequest
		givenStored  *dbmodel.DecisionObject
		wantUpdate   dbmodel.DecisionObject
		wantReview   model.DecisionReview
		want         model.ReviewDecisionResult
	}{
		{
			name:         "first review",
			givenRequest: model.ReviewDecisionRequest{DecisionID: decisionA, Score: 3, Reason: "held up in prod"},
			givenStored:  &dbmodel.DecisionObject{Title: "A", Status: "accepted", Content: requiredDetailsYAML},
			wantUpdate:   dbmodel.DecisionObject{},
			wantReview:   model.DecisionReview{Score: 3, Reason: "held up in prod"},
			want:         model.ReviewDecisionResult{DecisionID: decisionA},
		},
		{
			name:         "new review replaces the earlier one",
			givenRequest: model.ReviewDecisionRequest{DecisionID: decisionA, Score: 4, Reason: "worked after tuning"},
			givenStored: &dbmodel.DecisionObject{Title: "A", Status: "accepted", Content: requiredDetailsYAML,
				Review: "score: 2\nreason: slow\nreviewed_at: 2026-10-01T12:00:00Z\n"},
			wantUpdate: dbmodel.DecisionObject{},
			wantReview: model.DecisionReview{Score: 4, Reason: "worked after tuning"},
			want:       model.ReviewDecisionResult{DecisionID: decisionA},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			store.EXPECT().Get(mock.Anything, test.givenRequest.DecisionID).Return(test.givenStored, nil)
			var gotUpdate dbmodel.DecisionObject
			store.EXPECT().Update(mock.Anything, test.givenRequest.DecisionID, mock.Anything).
				Run(func(_ context.Context, _ uuid.UUID, d dbmodel.DecisionObject) { gotUpdate = d }).
				Return(nil)
			s := NewToolService(nil, nil, store)

			res := s.ReviewDecision(context.Background(), test.givenRequest)

			require.False(t, res.GetIsError(), string(res.GetData()))
			var got model.ReviewDecisionResult
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			var gotReview model.DecisionReview
			require.NoError(t, yaml.Unmarshal([]byte(gotUpdate.Review), &gotReview))
			test.wantUpdate.Review = gotUpdate.Review
			test.wantReview.ReviewedAt = gotReview.ReviewedAt
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantUpdate, gotUpdate)
			assert.Equal(t, test.wantReview, gotReview)
		})
	}
}

func TestReviewDecisionErrors(t *testing.T) {
	tests := []struct {
		name             string
		givenRequest     model.ReviewDecisionRequest
		givenStored      *dbmodel.DecisionObject
		givenGetError    error
		givenUpdateError error
		want             model.Error
	}{
		{
			name:         "unknown decision",
			givenRequest: model.ReviewDecisionRequest{DecisionID: decisionA, Score: 4, Reason: "worked"},
			want: model.Error{
				Code:    env.ServiceErrorDecisionNotFound,
				Message: "decision not found: aaaaaaaa-0000-0000-0000-000000000001",
			},
		},
		{
			name:          "store get fails",
			givenRequest:  model.ReviewDecisionRequest{DecisionID: decisionA, Score: 4, Reason: "worked"},
			givenGetError: errors.New("weaviate down"),
			want:          model.Error{Code: env.ServiceErrorInternal, Message: "weaviate down"},
		},
		{
			name:             "store update fails",
			givenRequest:     model.ReviewDecisionRequest{DecisionID: decisionA, Score: 4, Reason: "worked"},
			givenStored:      &dbmodel.DecisionObject{Title: "A", Status: "accepted", Content: requiredDetailsYAML},
			givenUpdateError: errors.New("weaviate down"),
			want:             model.Error{Code: env.ServiceErrorInternal, Message: "weaviate down"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := NewMockDecisionStore(t)
			store.EXPECT().Get(mock.Anything, test.givenRequest.DecisionID).Return(test.givenStored, test.givenGetError)
			if test.givenUpdateError != nil {
				store.EXPECT().Update(mock.Anything, test.givenRequest.DecisionID, mock.Anything).Return(test.givenUpdateError)
			}
			s := NewToolService(nil, nil, store)

			res := s.ReviewDecision(context.Background(), test.givenRequest)

			require.True(t, res.GetIsError())
			var got model.Error
			require.NoError(t, json.Unmarshal(res.GetData(), &got))
			assert.Equal(t, test.want, got)
		})
	}
}
