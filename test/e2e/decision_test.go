//go:build e2e

// Test plan: e2e precedent-decide, precedent-refine and precedent-review business flows
//
// A business client follows SKILL.md of each skill to record, replace and score architecture decisions.
//
// Logic flow
// | Step | Client call                   | Path                                                                                                                                            | Expected                                                    |
// |------|-------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------|
// | T    | Every tools/call              | As in step T of context_test.go                                                                                                                 | As in step T of context_test.go                             |
// | D1   | tools/call record_decision    | ToolService.RecordDecision: DecisionRepo.Create; with supersedes, DecisionRepo.Update of the old decision to superseded                         | {decision_id}                                               |
// | D2   | tools/call retrieve_decisions | ToolService.RetrieveDecisions: DecisionRepo.Search hybrid without superseded; match rank and review rank fused, unreviewed as 3; score 0 to 1   | {decisions}, highest score first; at most top_n, default 20 |
// | D3   | tools/call review_decision    | ToolService.ReviewDecision: DecisionRepo.Get; DecisionRepo.Update with the new review as the latest review                                      | {decision_id}; unknown decision: isError decision_not_found |
//
// Scenarios
// | Number | Step   | Result  | Value class    | Input                                                            | Expected                                                                             |
// |--------|--------|---------|----------------|------------------------------------------------------------------|--------------------------------------------------------------------------------------|
// | 1      | D1, D2 | Success | Middle value   | D1 accepted with tags and every Details field; D2 the topic      | The decision with every field; supersedes and review absent                          |
// | 2      | D1, D2 | Success | Boundary value | D1 proposed with title, status and the 3 required Details fields | The decision; tags, supersedes and review absent                                     |
// | 3      | D1, D2 | Success | Middle value   | D1 accepted A; D1 accepted B with supersedes A; D2               | B holds supersedes A; A absent                                                       |
// | 4      | D1, D2 | Success | Middle value   | D1 X and Y with the same text; D3 Y score 5; D2                  | Y before X; both scores within 0 to 1                                                |
// | 5      | D3, D2 | Success | Boundary value | D1 A; D3 score 5; D3 score 1; D2                                 | Both D3 return A; review holds score 1 and its reason only                           |
// | 6      | D2     | Success | Boundary value | D1 2 decisions; D2 top_n 1                                       | 1 decision; it is one of the 2                                                       |
// | 7      | D1     | Failure | Middle value   | D1 supersedes a new UUID                                         | isError; internal "decision <id> created, but supersede decision <uuid> failed: ..." |
// | 8      | D3     | Failure | Middle value   | D3 a new UUID                                                    | isError; decision_not_found "decision not found: <uuid>"                             |
// | 9      | D1     | Failure | Middle value   | D1 supersedes "abc"                                              | isError; invalid_argument 'abc' is not valid uuid                                    |
// | 10     | D1     | Failure | Boundary value | D1 title ""                                                      | isError; invalid_argument minLength: got 0, want 1                                   |
// | 11     | D1     | Failure | Middle value   | D1 status "draft"                                                | isError; invalid_argument value must be one of                                       |
// | 12     | D1     | Failure | Boundary value | D1 status "superseded"                                           | isError; invalid_argument value must be one of                                       |
// | 13     | D1     | Failure | Boundary value | D1 without Details                                               | isError; invalid_argument missing property 'Details'                                 |
// | 14     | D1     | Failure | Boundary value | D1 Details situation ""                                          | isError; invalid_argument minLength: got 0, want 1                                   |
// | 15     | D2     | Failure | Boundary value | D2 top_n 0                                                       | isError; invalid_argument minimum: got 0, want 1                                     |
// | 16     | D2     | Failure | Boundary value | D2 top_n 21                                                      | isError; invalid_argument maximum: got 21, want 20                                   |
// | 17     | D2     | Failure | Boundary value | D2 query ""                                                      | isError; invalid_argument minLength: got 0, want 1                                   |
// | 18     | D3     | Failure | Boundary value | D3 score 0                                                       | isError; invalid_argument minimum: got 0, want 1                                     |
// | 19     | D3     | Failure | Boundary value | D3 score 6                                                       | isError; invalid_argument maximum: got 6, want 5                                     |
// | 20     | D3     | Failure | Middle value   | D3 decision_id "abc"                                             | isError; invalid_argument 'abc' is not valid uuid                                    |
// | 21     | D3     | Failure | Boundary value | D3 reason ""                                                     | isError; invalid_argument minLength: got 0, want 1                                   |
//
// Branch coverage
// | Step | Condition                    | True   | False   |
// |------|------------------------------|--------|---------|
// | T    | Arguments match input_schema | 1 to 8 | 9 to 21 |
// | D1   | supersedes given             | 3, 7   | 1, 2    |
// | D1   | Superseded decision updated  | 3      | 7       |
// | D2   | Decision superseded          | 3      | 1       |
// | D2   | Decision reviewed            | 4      | 4       |
// | D2   | top_n given                  | 6      | 1       |
// | D3   | Decision exists              | 5      | 8       |
// | D3   | Earlier review replaced      | 5      | 5       |
//
// Test mapping
// | Test                                                                             | Scenarios |
// |----------------------------------------------------------------------------------|-----------|
// | TestRecordDecisionAllFields/accepted with tags and every Details field           | 1         |
// | TestRecordDecisionRequiredFields/proposed with the three required Details fields | 2         |
// | TestSupersedeDecision/B supersedes A                                             | 3         |
// | TestReviewedOutranksUnreviewed/reviewed Y ranks before unreviewed X              | 4         |
// | TestReviewDecision/latest review replaces the earlier one                        | 5         |
// | TestRetrieveDecisionsTopN/top_n 1 of two decisions                               | 6         |
// | TestSupersedeUnknownDecision/supersedes a new UUID                               | 7         |
// | TestReviewUnknownDecision/review of a new UUID                                   | 8         |
// | TestDecisionInvalidArguments/supersedes abc                                      | 9         |
// | TestDecisionInvalidArguments/empty title                                         | 10        |
// | TestDecisionInvalidArguments/status draft                                        | 11        |
// | TestDecisionInvalidArguments/status superseded                                   | 12        |
// | TestDecisionInvalidArguments/no Details                                          | 13        |
// | TestDecisionInvalidArguments/empty situation                                     | 14        |
// | TestDecisionInvalidArguments/top_n 0                                             | 15        |
// | TestDecisionInvalidArguments/top_n 21                                            | 16        |
// | TestDecisionInvalidArguments/empty query                                         | 17        |
// | TestDecisionInvalidArguments/score 0                                             | 18        |
// | TestDecisionInvalidArguments/score 6                                             | 19        |
// | TestDecisionInvalidArguments/decision_id abc                                     | 20        |
// | TestDecisionInvalidArguments/empty reason                                        | 21        |
// | No test                                                                          | None      |

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (s *E2ESuite) TestRecordDecisionAllFields() {
	details := map[string]any{
		"situation": s.topic + " agents need shared recall",
		"decision":  "Store memory in Weaviate",
		"rationale": "Hybrid search out of the box",
		"options":   []any{map[string]any{"name": "Weaviate", "pros": []any{"hybrid search"}, "cons": []any{"one more service"}, "verdict": "chosen"}},
		"tradeoffs": []any{"one more service to run"},
		"facts":     []any{map[string]any{"text": "Weaviate supports BM25", "source": "https://docs.weaviate.io"}},
		"practices": []any{map[string]any{"text": "keep schemas small"}},
	}
	tests := []struct {
		name               string
		givenRecordRequest map[string]any
		givenRecordHeader  http.Header
		givenSearchRequest map[string]any
		givenSearchHeader  http.Header
		wantStatus         int
		wantRecord         map[string]any
		wantDecision       map[string]any
	}{
		{
			name: "accepted with tags and every Details field",
			givenRecordRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{
					"title": s.topic + " use Weaviate", "status": "accepted", "tags": []any{"storage"}, "Details": details,
				},
			}},
			givenRecordHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			wantRecord: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"isError":    false,
			}},
			wantDecision: map[string]any{"title": s.topic + " use Weaviate", "status": "accepted", "tags": []any{"storage"}, "Details": details},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRecordRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotRecord map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotRecord))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			decisionID, _ := gotRecord["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)
			wantResult := test.wantRecord["result"].(map[string]any)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": `{"decision_id":"` + decisionID + `"}`}}
			wantResult["structuredContent"] = map[string]any{"decision_id": decisionID}
			test.wantDecision["decision_id"] = decisionID
			s.Require().Equal(test.wantRecord, gotRecord)
			s.NoError(s.schemas["record_decision"].Validate(gotRecord["result"].(map[string]any)["structuredContent"]))

			s.EventuallyWithT(func(c *assert.CollectT) {
				payload, err := json.Marshal(test.givenSearchRequest)
				if !assert.NoError(c, err) {
					return
				}
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				if !assert.NoError(c, err) {
					return
				}
				req.Header = test.givenSearchHeader
				res, err := s.client.Do(req)
				if !assert.NoError(c, err) {
					return
				}
				defer func() { assert.NoError(c, res.Body.Close()) }()
				var gotSearch map[string]any
				if !assert.NoError(c, json.NewDecoder(res.Body).Decode(&gotSearch)) {
					return
				}

				assert.Equal(c, test.wantStatus, res.StatusCode)
				gotResult, _ := gotSearch["result"].(map[string]any)
				assert.Equal(c, false, gotResult["isError"])
				gotStructured, _ := gotResult["structuredContent"].(map[string]any)
				assert.NoError(c, s.schemas["retrieve_decisions"].Validate(gotStructured))
				gotDecisions, _ := gotStructured["decisions"].([]any)
				var gotDecision map[string]any
				for _, decision := range gotDecisions {
					if item, _ := decision.(map[string]any); item["decision_id"] == decisionID {
						gotDecision = item
					}
				}
				if !assert.NotNil(c, gotDecision) {
					return
				}
				delete(gotDecision, "created_at")
				delete(gotDecision, "score")
				assert.Equal(c, test.wantDecision, gotDecision)
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestRecordDecisionRequiredFields() {
	tests := []struct {
		name               string
		givenRecordRequest map[string]any
		givenRecordHeader  http.Header
		givenSearchRequest map[string]any
		givenSearchHeader  http.Header
		wantStatus         int
		wantDecision       map[string]any
	}{
		{
			name: "proposed with the three required Details fields",
			givenRecordRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{
					"title": s.topic + " required only", "status": "proposed",
					"Details": map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"},
				},
			}},
			givenRecordHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			wantDecision: map[string]any{"title": s.topic + " required only", "status": "proposed",
				"Details": map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRecordRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotRecord map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotRecord))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(false, gotRecord["result"].(map[string]any)["isError"])
			decisionID, _ := gotRecord["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)
			test.wantDecision["decision_id"] = decisionID

			s.EventuallyWithT(func(c *assert.CollectT) {
				payload, err := json.Marshal(test.givenSearchRequest)
				if !assert.NoError(c, err) {
					return
				}
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				if !assert.NoError(c, err) {
					return
				}
				req.Header = test.givenSearchHeader
				res, err := s.client.Do(req)
				if !assert.NoError(c, err) {
					return
				}
				defer func() { assert.NoError(c, res.Body.Close()) }()
				var gotSearch map[string]any
				if !assert.NoError(c, json.NewDecoder(res.Body).Decode(&gotSearch)) {
					return
				}

				assert.Equal(c, test.wantStatus, res.StatusCode)
				gotStructured, _ := gotSearch["result"].(map[string]any)["structuredContent"].(map[string]any)
				assert.NoError(c, s.schemas["retrieve_decisions"].Validate(gotStructured))
				gotDecisions, _ := gotStructured["decisions"].([]any)
				var gotDecision map[string]any
				for _, decision := range gotDecisions {
					if item, _ := decision.(map[string]any); item["decision_id"] == decisionID {
						gotDecision = item
					}
				}
				if !assert.NotNil(c, gotDecision) {
					return
				}
				delete(gotDecision, "created_at")
				delete(gotDecision, "score")
				assert.Equal(c, test.wantDecision, gotDecision)
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestSupersedeDecision() {
	details := map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}
	tests := []struct {
		name               string
		givenFirstRequest  map[string]any
		givenSecondRequest map[string]any
		givenRecordHeader  http.Header
		givenSearchRequest map[string]any
		givenSearchHeader  http.Header
		wantStatus         int
		wantSecond         map[string]any
	}{
		{
			name: "B supersedes A",
			givenFirstRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " A", "status": "accepted", "Details": details},
			}},
			givenSecondRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " B", "status": "accepted", "Details": details},
			}},
			givenRecordHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			wantSecond: map[string]any{"title": s.topic + " B", "status": "accepted", "Details": details},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenFirstRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotFirst map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotFirst))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(false, gotFirst["result"].(map[string]any)["isError"])
			firstID, _ := gotFirst["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)
			test.givenSecondRequest["params"].(map[string]any)["arguments"].(map[string]any)["supersedes"] = firstID

			payload, err = json.Marshal(test.givenSecondRequest)
			s.Require().NoError(err)
			req, err = http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err = s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotSecond map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotSecond))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(false, gotSecond["result"].(map[string]any)["isError"])
			secondID, _ := gotSecond["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)
			test.wantSecond["decision_id"] = secondID
			test.wantSecond["supersedes"] = firstID

			s.EventuallyWithT(func(c *assert.CollectT) {
				payload, err := json.Marshal(test.givenSearchRequest)
				if !assert.NoError(c, err) {
					return
				}
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				if !assert.NoError(c, err) {
					return
				}
				req.Header = test.givenSearchHeader
				res, err := s.client.Do(req)
				if !assert.NoError(c, err) {
					return
				}
				defer func() { assert.NoError(c, res.Body.Close()) }()
				var gotSearch map[string]any
				if !assert.NoError(c, json.NewDecoder(res.Body).Decode(&gotSearch)) {
					return
				}

				assert.Equal(c, test.wantStatus, res.StatusCode)
				gotStructured, _ := gotSearch["result"].(map[string]any)["structuredContent"].(map[string]any)
				assert.NoError(c, s.schemas["retrieve_decisions"].Validate(gotStructured))
				gotDecisions, _ := gotStructured["decisions"].([]any)
				var gotDecision map[string]any
				for _, decision := range gotDecisions {
					item, _ := decision.(map[string]any)
					assert.NotEqual(c, firstID, item["decision_id"])
					if item["decision_id"] == secondID {
						gotDecision = item
					}
				}
				if !assert.NotNil(c, gotDecision) {
					return
				}
				delete(gotDecision, "created_at")
				delete(gotDecision, "score")
				assert.Equal(c, test.wantSecond, gotDecision)
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestReviewedOutranksUnreviewed() {
	details := map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}
	tests := []struct {
		name               string
		givenXRequest      map[string]any
		givenYRequest      map[string]any
		givenRecordHeader  http.Header
		givenReviewRequest map[string]any
		givenReviewHeader  http.Header
		givenSearchRequest map[string]any
		givenSearchHeader  http.Header
		wantStatus         int
		wantScoreMinimum   float64
		wantScoreMaximum   float64
	}{
		{
			name: "reviewed Y ranks before unreviewed X",
			givenXRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " option", "status": "accepted", "Details": details},
			}},
			givenYRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " option", "status": "accepted", "Details": details},
			}},
			givenRecordHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			givenReviewRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "review_decision", "_meta": meta, "arguments": map[string]any{"score": 5, "reason": "worked as decided"},
			}},
			givenReviewHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus:       http.StatusOK,
			wantScoreMinimum: 0,
			wantScoreMaximum: 1,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenXRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotX map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotX))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			xID, _ := gotX["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)

			payload, err = json.Marshal(test.givenYRequest)
			s.Require().NoError(err)
			req, err = http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err = s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotY map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotY))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			yID, _ := gotY["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)

			test.givenReviewRequest["params"].(map[string]any)["arguments"].(map[string]any)["decision_id"] = yID
			payload, err = json.Marshal(test.givenReviewRequest)
			s.Require().NoError(err)
			req, err = http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenReviewHeader
			res, err = s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotReview map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotReview))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(false, gotReview["result"].(map[string]any)["isError"])
			wantOrder := []any{yID, xID}

			s.EventuallyWithT(func(c *assert.CollectT) {
				payload, err := json.Marshal(test.givenSearchRequest)
				if !assert.NoError(c, err) {
					return
				}
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				if !assert.NoError(c, err) {
					return
				}
				req.Header = test.givenSearchHeader
				res, err := s.client.Do(req)
				if !assert.NoError(c, err) {
					return
				}
				defer func() { assert.NoError(c, res.Body.Close()) }()
				var gotSearch map[string]any
				if !assert.NoError(c, json.NewDecoder(res.Body).Decode(&gotSearch)) {
					return
				}

				assert.Equal(c, test.wantStatus, res.StatusCode)
				gotStructured, _ := gotSearch["result"].(map[string]any)["structuredContent"].(map[string]any)
				gotDecisions, _ := gotStructured["decisions"].([]any)
				assert.NoError(c, s.schemas["retrieve_decisions"].Validate(gotStructured))
				var gotOrder []any
				for _, decision := range gotDecisions {
					item, _ := decision.(map[string]any)
					if item["decision_id"] == xID || item["decision_id"] == yID {
						gotOrder = append(gotOrder, item["decision_id"])
						gotScore, _ := item["score"].(float64)
						assert.GreaterOrEqual(c, gotScore, test.wantScoreMinimum)
						assert.LessOrEqual(c, gotScore, test.wantScoreMaximum)
					}
				}
				assert.Equal(c, wantOrder, gotOrder)
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestReviewDecision() {
	tests := []struct {
		name                string
		givenRecordRequest  map[string]any
		givenRecordHeader   http.Header
		givenReviewRequests []map[string]any
		givenReviewHeader   http.Header
		givenSearchRequest  map[string]any
		givenSearchHeader   http.Header
		wantStatus          int
		wantReview          map[string]any
		wantLatestReview    map[string]any
	}{
		{
			name: "latest review replaces the earlier one",
			givenRecordRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " reviewed", "status": "accepted",
					"Details": map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}},
			}},
			givenRecordHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			givenReviewRequests: []map[string]any{
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "review_decision", "_meta": meta, "arguments": map[string]any{"score": 5, "reason": "worked as decided"}}},
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "review_decision", "_meta": meta, "arguments": map[string]any{"score": 1, "reason": "rejected later"}}},
			},
			givenReviewHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			wantReview: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"isError":    false,
			}},
			wantLatestReview: map[string]any{"score": float64(1), "reason": "rejected later"},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRecordRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenRecordHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotRecord map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotRecord))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(false, gotRecord["result"].(map[string]any)["isError"])
			decisionID, _ := gotRecord["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"].(string)
			wantResult := test.wantReview["result"].(map[string]any)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": `{"decision_id":"` + decisionID + `"}`}}
			wantResult["structuredContent"] = map[string]any{"decision_id": decisionID}

			for _, givenReviewRequest := range test.givenReviewRequests {
				givenReviewRequest["params"].(map[string]any)["arguments"].(map[string]any)["decision_id"] = decisionID
				payload, err := json.Marshal(givenReviewRequest)
				s.Require().NoError(err)
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				s.Require().NoError(err)
				req.Header = test.givenReviewHeader
				res, err := s.client.Do(req)
				s.Require().NoError(err)
				defer func() { s.NoError(res.Body.Close()) }()
				var gotReview map[string]any
				s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotReview))

				s.Equal(test.wantStatus, res.StatusCode)
				s.Require().Equal(test.wantReview, gotReview)
				s.NoError(s.schemas["review_decision"].Validate(gotReview["result"].(map[string]any)["structuredContent"]))
			}

			s.EventuallyWithT(func(c *assert.CollectT) {
				payload, err := json.Marshal(test.givenSearchRequest)
				if !assert.NoError(c, err) {
					return
				}
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				if !assert.NoError(c, err) {
					return
				}
				req.Header = test.givenSearchHeader
				res, err := s.client.Do(req)
				if !assert.NoError(c, err) {
					return
				}
				defer func() { assert.NoError(c, res.Body.Close()) }()
				var gotSearch map[string]any
				if !assert.NoError(c, json.NewDecoder(res.Body).Decode(&gotSearch)) {
					return
				}

				assert.Equal(c, test.wantStatus, res.StatusCode)
				gotStructured, _ := gotSearch["result"].(map[string]any)["structuredContent"].(map[string]any)
				assert.NoError(c, s.schemas["retrieve_decisions"].Validate(gotStructured))
				gotDecisions, _ := gotStructured["decisions"].([]any)
				var gotLatestReview map[string]any
				for _, decision := range gotDecisions {
					if item, _ := decision.(map[string]any); item["decision_id"] == decisionID {
						gotLatestReview, _ = item["review"].(map[string]any)
					}
				}
				delete(gotLatestReview, "reviewed_at")
				assert.Equal(c, test.wantLatestReview, gotLatestReview)
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestRetrieveDecisionsTopN() {
	details := map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}
	tests := []struct {
		name                string
		givenRecordRequests []map[string]any
		givenRecordHeader   http.Header
		givenSearchRequest  map[string]any
		givenSearchHeader   http.Header
		wantStatus          int
		wantCount           int
	}{
		{
			name: "top_n 1 of two decisions",
			givenRecordRequests: []map[string]any{
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " first", "status": "accepted", "Details": details}}},
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " second", "status": "accepted", "Details": details}}},
			},
			givenRecordHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 1},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			var recordedIDs []any
			for _, givenRecordRequest := range test.givenRecordRequests {
				payload, err := json.Marshal(givenRecordRequest)
				s.Require().NoError(err)
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				s.Require().NoError(err)
				req.Header = test.givenRecordHeader
				res, err := s.client.Do(req)
				s.Require().NoError(err)
				defer func() { s.NoError(res.Body.Close()) }()
				var gotRecord map[string]any
				s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotRecord))

				s.Require().Equal(test.wantStatus, res.StatusCode)
				s.Require().Equal(false, gotRecord["result"].(map[string]any)["isError"])
				recordedIDs = append(recordedIDs, gotRecord["result"].(map[string]any)["structuredContent"].(map[string]any)["decision_id"])
			}

			s.EventuallyWithT(func(c *assert.CollectT) {
				payload, err := json.Marshal(test.givenSearchRequest)
				if !assert.NoError(c, err) {
					return
				}
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				if !assert.NoError(c, err) {
					return
				}
				req.Header = test.givenSearchHeader
				res, err := s.client.Do(req)
				if !assert.NoError(c, err) {
					return
				}
				defer func() { assert.NoError(c, res.Body.Close()) }()
				var gotSearch map[string]any
				if !assert.NoError(c, json.NewDecoder(res.Body).Decode(&gotSearch)) {
					return
				}

				assert.Equal(c, test.wantStatus, res.StatusCode)
				gotStructured, _ := gotSearch["result"].(map[string]any)["structuredContent"].(map[string]any)
				assert.NoError(c, s.schemas["retrieve_decisions"].Validate(gotStructured))
				gotDecisions, _ := gotStructured["decisions"].([]any)
				if assert.Len(c, gotDecisions, test.wantCount) {
					item, _ := gotDecisions[0].(map[string]any)
					assert.Contains(c, recordedIDs, item["decision_id"])
				}
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestSupersedeUnknownDecision() {
	unknownID := uuid.NewString()
	tests := []struct {
		name              string
		givenRequest      map[string]any
		givenHeader       http.Header
		wantStatus        int
		wantMessagePrefix string
		wantMessagePart   string
		want              map[string]any
	}{
		{
			name: "supersedes a new UUID",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic + " orphan", "status": "accepted",
					"supersedes": unknownID, "Details": map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}},
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus:        http.StatusOK,
			wantMessagePrefix: "decision ",
			wantMessagePart:   " created, but supersede decision " + unknownID + " failed: ",
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"isError":    true,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			gotResult := got["result"].(map[string]any)
			message, _ := gotResult["structuredContent"].(map[string]any)["message"].(string)
			s.True(strings.HasPrefix(message, test.wantMessagePrefix), message)
			s.Contains(message, test.wantMessagePart)
			text, _ := gotResult["content"].([]any)[0].(map[string]any)["text"].(string)
			wantResult := test.want["result"].(map[string]any)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": text}}
			wantResult["structuredContent"] = map[string]any{"code": "internal", "message": message}
			var mirrored map[string]any
			s.Require().NoError(json.Unmarshal([]byte(text), &mirrored))
			s.Equal(wantResult["structuredContent"], mirrored)
			s.Equal(test.want, got)
			s.NoError(s.schemas["record_decision"].Validate(gotResult["structuredContent"]))
		})
	}
}

func (s *E2ESuite) TestReviewUnknownDecision() {
	unknownID := uuid.NewString()
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name: "review of a new UUID",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "review_decision", "_meta": meta, "arguments": map[string]any{"decision_id": unknownID, "score": 5, "reason": "worked"},
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"code":"decision_not_found","message":"decision not found: ` + unknownID + `"}`}},
				"structuredContent": map[string]any{"code": "decision_not_found", "message": "decision not found: " + unknownID},
				"isError":           true,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.want, got)
			s.NoError(s.schemas["review_decision"].Validate(got["result"].(map[string]any)["structuredContent"]))
		})
	}
}

func (s *E2ESuite) TestDecisionInvalidArguments() {
	details := map[string]any{"situation": s.topic + " situation", "decision": "decision", "rationale": "rationale"}
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name: "supersedes abc",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic, "status": "accepted", "Details": details, "supersedes": "abc"}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/supersedes': 'abc' is not valid uuid: must have 5 elements"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/supersedes': 'abc' is not valid uuid: must have 5 elements"},
				"isError": true,
			}},
		},
		{
			name: "empty title",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": "", "status": "accepted", "Details": details}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/title': minLength: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/title': minLength: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "status draft",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic, "status": "draft", "Details": details}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/status': value must be one of 'proposed', 'accepted'"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/status': value must be one of 'proposed', 'accepted'"},
				"isError": true,
			}},
		},
		{
			name: "status superseded",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic, "status": "superseded", "Details": details}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/status': value must be one of 'proposed', 'accepted'"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/status': value must be one of 'proposed', 'accepted'"},
				"isError": true,
			}},
		},
		{
			name: "no Details",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic, "status": "accepted"}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '': missing property 'Details'"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '': missing property 'Details'"},
				"isError": true,
			}},
		},
		{
			name: "empty situation",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "record_decision", "_meta": meta, "arguments": map[string]any{"title": s.topic, "status": "accepted",
					"Details": map[string]any{"situation": "", "decision": "decision", "rationale": "rationale"}}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"record_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/Details/situation': minLength: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:record_decision#'\n- at '/Details/situation': minLength: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "top_n 0",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 0}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:retrieve_decisions#'\n- at '/top_n': minimum: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:retrieve_decisions#'\n- at '/top_n': minimum: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "top_n 21",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 21}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:retrieve_decisions#'\n- at '/top_n': maximum: got 21, want 20"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:retrieve_decisions#'\n- at '/top_n': maximum: got 21, want 20"},
				"isError": true,
			}},
		},
		{
			name: "empty query",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_decisions", "_meta": meta, "arguments": map[string]any{"query": ""}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_decisions"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:retrieve_decisions#'\n- at '/query': minLength: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:retrieve_decisions#'\n- at '/query': minLength: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "score 0",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "review_decision", "_meta": meta, "arguments": map[string]any{"decision_id": uuid.NewString(), "score": 0, "reason": "reason"}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/score': minimum: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/score': minimum: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "score 6",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "review_decision", "_meta": meta, "arguments": map[string]any{"decision_id": uuid.NewString(), "score": 6, "reason": "reason"}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/score': maximum: got 6, want 5"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/score': maximum: got 6, want 5"},
				"isError": true,
			}},
		},
		{
			name: "decision_id abc",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "review_decision", "_meta": meta, "arguments": map[string]any{"decision_id": "abc", "score": 5, "reason": "reason"}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/decision_id': 'abc' is not valid uuid: must have 5 elements"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/decision_id': 'abc' is not valid uuid: must have 5 elements"},
				"isError": true,
			}},
		},
		{
			name: "empty reason",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "review_decision", "_meta": meta, "arguments": map[string]any{"decision_id": uuid.NewString(), "score": 5, "reason": ""}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"review_decision"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/reason': minLength: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:review_decision#'\n- at '/reason': minLength: got 0, want 1"},
				"isError": true,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var got map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.want, got)
			s.NoError(s.schemas[test.givenRequest["params"].(map[string]any)["name"].(string)].Validate(got["result"].(map[string]any)["structuredContent"]))
		})
	}
}
