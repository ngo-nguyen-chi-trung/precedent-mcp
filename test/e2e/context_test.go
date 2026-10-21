//go:build e2e

// Test plan: e2e precedent-capture and precedent-recall business flows
//
// A business client follows SKILL.md of precedent-capture to save context, and of precedent-recall to load it.
//
// Logic flow
// | Step | Client call                   | Path                                                                                                 | Expected                                                                                           |
// |------|-------------------------------|------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------|
// | T    | Every tools/call              | Gateway pipe; ToolService.CallTool: KnownTool, Arguments against input_schema with format assertion  | HTTP 200; one text block mirroring structuredContent; structuredContent valid against outputSchema |
// | T    | Every tools/call              | Arguments outside input_schema in assets/tools.yaml                                                  | isError; invalid_argument "invalid args: ..." naming the path                                      |
// | C1   | tools/call add_context        | ToolService.AddContext: ID from thread_id and context; one write per ID; ContextRepo.Create          | {succeeded, failed}: positions in entries, from 0                                                  |
// | C2   | tools/call get_context_thread | ToolService.GetContextThread: ContextRepo.Get latest 20; sort oldest first                           | {thread_id, entries}: latest 20 entries of the thread, oldest first                                |
// | C3   | tools/call retrieve_context   | ToolService.RetrieveContext: ContextRepo.Search hybrid without score cutoff; limit top_n, default 20 | {results}: best match first, at most top_n                                                         |
//
// Scenarios
// | Number | Step       | Result  | Value class    | Input                                                  | Expected                                                                     |
// |--------|------------|---------|----------------|--------------------------------------------------------|------------------------------------------------------------------------------|
// | 1      | C1, C2, C3 | Success | Middle value   | C1 3 entries in thread T; C2 T; C3 the topic           | succeeded [0, 1, 2]; C2 returns the 3 texts in order; C3 holds the 3 entries |
// | 2      | C1, C2     | Success | Middle value   | C1 entries for threads A, B, A in one call; C2 A and B | A returns its 2 texts in order; B returns its 1 text                         |
// | 3      | C1         | Success | Boundary value | C1 1 entry                                             | succeeded [0]; failed []                                                     |
// | 4      | C1         | Success | Boundary value | C1 10 entries                                          | succeeded [0 to 9]; failed []                                                |
// | 5      | C1, C2     | Success | Middle value   | C1 twice with the same 2 entries                       | Both calls succeeded [0, 1]; C2 returns 2 texts                              |
// | 6      | C1, C2     | Success | Boundary value | C1 one entry at positions 0 and 1                      | succeeded [0, 1]; C2 returns 1 text                                          |
// | 7      | C1, C2     | Success | Boundary value | C1 25 entries in calls of 10, 10, 5; C2                | C2 returns the latest 20 texts in order                                      |
// | 8      | C2         | Success | Boundary value | C2 a new UUID                                          | entries []                                                                   |
// | 9      | C3         | Success | Boundary value | 2 entries on the topic; C3 top_n 1                     | 1 result; it is one of the 2 entries                                         |
// | 10     | C3         | Success | Boundary value | 2 entries on the topic; C3 top_n 20                    | 2 to 20 results; the 2 entries first                                         |
// | 11     | C1         | Failure | Boundary value | C1 11 entries                                          | isError; invalid_argument maxItems: got 11, want 10                          |
// | 12     | C1         | Failure | Boundary value | C1 entries []                                          | isError; invalid_argument minItems: got 0, want 1                            |
// | 13     | C1         | Failure | Middle value   | C1 thread_id "not-a-uuid"                              | isError; invalid_argument 'not-a-uuid' is not valid uuid                     |
// | 14     | C1         | Failure | Boundary value | C1 context ""                                          | isError; invalid_argument minLength: got 0, want 1                           |
// | 15     | C1         | Failure | Middle value   | C1 entry with the extra property note                  | isError; invalid_argument additional properties 'note' not allowed           |
// | 16     | T          | Failure | Boundary value | C1 arguments null                                      | isError; invalid_argument got null, want object                              |
// | 17     | C3         | Failure | Boundary value | C3 top_n 21                                            | isError; invalid_argument maximum: got 21, want 20                           |
// | 18     | C3         | Failure | Boundary value | C3 top_n 0                                             | isError; invalid_argument minimum: got 0, want 1                             |
// | 19     | C3         | Failure | Boundary value | C3 query ""                                            | isError; invalid_argument minLength: got 0, want 1                           |
// | 20     | C2         | Failure | Middle value   | C2 thread_id "abc"                                     | isError; invalid_argument 'abc' is not valid uuid                            |
//
// Branch coverage
// | Step | Condition                                      | True    | False                                     |
// |------|------------------------------------------------|---------|-------------------------------------------|
// | T    | Arguments match input_schema                   | 1 to 10 | 11 to 20                                  |
// | C1   | Same thread_id and context earlier in the call | 6       | 1                                         |
// | C1   | Same thread_id and context already stored      | 5       | 1                                         |
// | C1   | Store saves every entry                        | 1       | Unreachable in e2e; needs a store failure |
// | C2   | Thread holds more than 20 entries              | 7       | 1                                         |
// | C2   | Thread holds entries                           | 1       | 8                                         |
// | C3   | top_n given                                    | 9, 10   | 1                                         |
//
// Test mapping
// | Test                                                                   | Scenarios |
// |------------------------------------------------------------------------|-----------|
// | TestContextFlow/add, load the thread, find it by topic                 | 1         |
// | TestContextOneCallManyThreads/threads A, B, A in one call              | 2         |
// | TestAddContextEntryCount/one entry                                     | 3         |
// | TestAddContextEntryCount/ten entries                                   | 4         |
// | TestContextResendCreatesNoDuplicates/same two entries sent twice       | 5         |
// | TestContextDuplicateInOneCall/one entry at positions 0 and 1           | 6         |
// | TestContextThreadReturnsLatestEntries/25 entries in calls of 10, 10, 5 | 7         |
// | TestContextThreadUnknown/new thread id                                 | 8         |
// | TestRetrieveContextTopN/top_n 1 and 20                                 | 9, 10     |
// | TestContextInvalidArguments/eleven entries                             | 11        |
// | TestContextInvalidArguments/no entries                                 | 12        |
// | TestContextInvalidArguments/thread_id not a uuid                       | 13        |
// | TestContextInvalidArguments/empty context                              | 14        |
// | TestContextInvalidArguments/extra property                             | 15        |
// | TestContextInvalidArguments/null arguments                             | 16        |
// | TestContextInvalidArguments/top_n 21                                   | 17        |
// | TestContextInvalidArguments/top_n 0                                    | 18        |
// | TestContextInvalidArguments/empty query                                | 19        |
// | TestContextInvalidArguments/thread_id abc                              | 20        |
// | No test                                                                | None      |

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (s *E2ESuite) TestContextFlow() {
	thread := uuid.NewString()
	tests := []struct {
		name               string
		givenAddRequest    map[string]any
		givenAddHeader     http.Header
		givenThreadRequest map[string]any
		givenThreadHeader  http.Header
		givenSearchRequest map[string]any
		givenSearchHeader  http.Header
		wantStatus         int
		wantAdd            map[string]any
		wantThread         map[string]any
		wantSearchResults  []any
	}{
		{
			name: "add, load the thread, find it by topic",
			givenAddRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " first"},
					map[string]any{"thread_id": thread, "context": s.topic + " second"},
					map[string]any{"thread_id": thread, "context": s.topic + " third"},
				}},
			}},
			givenAddHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			givenThreadRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": thread},
			}},
			givenThreadHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			givenSearchRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_context", "_meta": meta, "arguments": map[string]any{"query": s.topic},
			}},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_context"}},
			wantStatus: http.StatusOK,
			wantAdd: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"succeeded":[0,1,2],"failed":[]}`}},
				"structuredContent": map[string]any{"succeeded": []any{float64(0), float64(1), float64(2)}, "failed": []any{}},
				"isError":           false,
			}},
			wantThread: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"structuredContent": map[string]any{"thread_id": thread, "entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " first"},
					map[string]any{"thread_id": thread, "context": s.topic + " second"},
					map[string]any{"thread_id": thread, "context": s.topic + " third"},
				}},
				"isError": false,
			}},
			wantSearchResults: []any{
				map[string]any{"thread_id": thread, "context": s.topic + " first"},
				map[string]any{"thread_id": thread, "context": s.topic + " second"},
				map[string]any{"thread_id": thread, "context": s.topic + " third"},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenAddRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenAddHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotAdd map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotAdd))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.wantAdd, gotAdd)
			s.NoError(s.schemas["add_context"].Validate(gotAdd["result"].(map[string]any)["structuredContent"]))

			payload, err = json.Marshal(test.givenThreadRequest)
			s.Require().NoError(err)
			req, err = http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenThreadHeader
			res, err = s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotThread map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotThread))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			gotResult := gotThread["result"].(map[string]any)
			gotEntries, _ := gotResult["structuredContent"].(map[string]any)["entries"].([]any)
			wantResult := test.wantThread["result"].(map[string]any)
			wantEntries := wantResult["structuredContent"].(map[string]any)["entries"].([]any)
			s.Require().Len(gotEntries, len(wantEntries))
			for i, entry := range wantEntries {
				entry.(map[string]any)["created_at"] = gotEntries[i].(map[string]any)["created_at"]
			}
			text, _ := gotResult["content"].([]any)[0].(map[string]any)["text"].(string)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": text}}
			var mirrored map[string]any
			s.Require().NoError(json.Unmarshal([]byte(text), &mirrored))
			s.Equal(wantResult["structuredContent"], mirrored)
			s.Equal(test.wantThread, gotThread)
			s.NoError(s.schemas["get_context_thread"].Validate(gotResult["structuredContent"]))

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
				assert.NoError(c, s.schemas["retrieve_context"].Validate(gotStructured))
				gotResults, _ := gotStructured["results"].([]any)
				var gotThreadResults []any
				for _, result := range gotResults {
					item, _ := result.(map[string]any)
					if item["thread_id"] == thread {
						delete(item, "score")
						gotThreadResults = append(gotThreadResults, item)
					}
				}
				assert.ElementsMatch(c, test.wantSearchResults, gotThreadResults)
			}, searchTimeout, searchTick)
		})
	}
}

func (s *E2ESuite) TestContextOneCallManyThreads() {
	threadA, threadB := uuid.NewString(), uuid.NewString()
	tests := []struct {
		name                string
		givenAddRequest     map[string]any
		givenAddHeader      http.Header
		givenThreadRequests []map[string]any
		givenThreadHeader   http.Header
		wantStatus          int
		wantAdd             map[string]any
		wantThreads         []map[string]any
	}{
		{
			name: "threads A, B, A in one call",
			givenAddRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": threadA, "context": s.topic + " thread a first"},
					map[string]any{"thread_id": threadB, "context": s.topic + " thread b only"},
					map[string]any{"thread_id": threadA, "context": s.topic + " thread a second"},
				}},
			}},
			givenAddHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			givenThreadRequests: []map[string]any{
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": threadA}}},
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": threadB}}},
			},
			givenThreadHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			wantAdd: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"succeeded":[0,1,2],"failed":[]}`}},
				"structuredContent": map[string]any{"succeeded": []any{float64(0), float64(1), float64(2)}, "failed": []any{}},
				"isError":           false,
			}},
			wantThreads: []map[string]any{
				{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
					"resultType": "complete",
					"_meta":      serverMeta,
					"structuredContent": map[string]any{"thread_id": threadA, "entries": []any{
						map[string]any{"thread_id": threadA, "context": s.topic + " thread a first"},
						map[string]any{"thread_id": threadA, "context": s.topic + " thread a second"},
					}},
					"isError": false,
				}},
				{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
					"resultType": "complete",
					"_meta":      serverMeta,
					"structuredContent": map[string]any{"thread_id": threadB, "entries": []any{
						map[string]any{"thread_id": threadB, "context": s.topic + " thread b only"},
					}},
					"isError": false,
				}},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenAddRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenAddHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotAdd map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotAdd))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.wantAdd, gotAdd)

			for i, givenThreadRequest := range test.givenThreadRequests {
				payload, err := json.Marshal(givenThreadRequest)
				s.Require().NoError(err)
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				s.Require().NoError(err)
				req.Header = test.givenThreadHeader
				res, err := s.client.Do(req)
				s.Require().NoError(err)
				defer func() { s.NoError(res.Body.Close()) }()
				var gotThread map[string]any
				s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotThread))

				s.Require().Equal(test.wantStatus, res.StatusCode)
				gotResult := gotThread["result"].(map[string]any)
				gotEntries, _ := gotResult["structuredContent"].(map[string]any)["entries"].([]any)
				wantResult := test.wantThreads[i]["result"].(map[string]any)
				wantEntries := wantResult["structuredContent"].(map[string]any)["entries"].([]any)
				s.Require().Len(gotEntries, len(wantEntries))
				for j, entry := range wantEntries {
					entry.(map[string]any)["created_at"] = gotEntries[j].(map[string]any)["created_at"]
				}
				text, _ := gotResult["content"].([]any)[0].(map[string]any)["text"].(string)
				wantResult["content"] = []any{map[string]any{"type": "text", "text": text}}
				var mirrored map[string]any
				s.Require().NoError(json.Unmarshal([]byte(text), &mirrored))
				s.Equal(wantResult["structuredContent"], mirrored)
				s.Equal(test.wantThreads[i], gotThread)
				s.NoError(s.schemas["get_context_thread"].Validate(gotResult["structuredContent"]))
			}
		})
	}
}

func (s *E2ESuite) TestAddContextEntryCount() {
	thread := uuid.NewString()
	tenEntries := make([]any, 10)
	for i := range tenEntries {
		tenEntries[i] = map[string]any{"thread_id": thread, "context": fmt.Sprintf("%s entry %d", s.topic, i)}
	}
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name: "one entry",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " only entry"},
				}},
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"succeeded":[0],"failed":[]}`}},
				"structuredContent": map[string]any{"succeeded": []any{float64(0)}, "failed": []any{}},
				"isError":           false,
			}},
		},
		{
			name: "ten entries",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": tenEntries},
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content":    []any{map[string]any{"type": "text", "text": `{"succeeded":[0,1,2,3,4,5,6,7,8,9],"failed":[]}`}},
				"structuredContent": map[string]any{"failed": []any{}, "succeeded": []any{
					float64(0), float64(1), float64(2), float64(3), float64(4), float64(5), float64(6), float64(7), float64(8), float64(9),
				}},
				"isError": false,
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
			s.NoError(s.schemas["add_context"].Validate(got["result"].(map[string]any)["structuredContent"]))
		})
	}
}

func (s *E2ESuite) TestContextResendCreatesNoDuplicates() {
	thread := uuid.NewString()
	tests := []struct {
		name               string
		givenAddRequest    map[string]any
		givenAddHeader     http.Header
		givenThreadRequest map[string]any
		givenThreadHeader  http.Header
		wantStatus         int
		wantAdd            map[string]any
		wantThread         map[string]any
	}{
		{
			name: "same two entries sent twice",
			givenAddRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " resend first"},
					map[string]any{"thread_id": thread, "context": s.topic + " resend second"},
				}},
			}},
			givenAddHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			givenThreadRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": thread},
			}},
			givenThreadHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			wantAdd: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"succeeded":[0,1],"failed":[]}`}},
				"structuredContent": map[string]any{"succeeded": []any{float64(0), float64(1)}, "failed": []any{}},
				"isError":           false,
			}},
			wantThread: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"structuredContent": map[string]any{"thread_id": thread, "entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " resend first"},
					map[string]any{"thread_id": thread, "context": s.topic + " resend second"},
				}},
				"isError": false,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			for range 2 {
				payload, err := json.Marshal(test.givenAddRequest)
				s.Require().NoError(err)
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				s.Require().NoError(err)
				req.Header = test.givenAddHeader
				res, err := s.client.Do(req)
				s.Require().NoError(err)
				defer func() { s.NoError(res.Body.Close()) }()
				var gotAdd map[string]any
				s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotAdd))

				s.Equal(test.wantStatus, res.StatusCode)
				s.Require().Equal(test.wantAdd, gotAdd)
			}

			payload, err := json.Marshal(test.givenThreadRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenThreadHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotThread map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotThread))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			gotResult := gotThread["result"].(map[string]any)
			gotEntries, _ := gotResult["structuredContent"].(map[string]any)["entries"].([]any)
			wantResult := test.wantThread["result"].(map[string]any)
			wantEntries := wantResult["structuredContent"].(map[string]any)["entries"].([]any)
			s.Require().Len(gotEntries, len(wantEntries))
			for i, entry := range wantEntries {
				entry.(map[string]any)["created_at"] = gotEntries[i].(map[string]any)["created_at"]
			}
			text, _ := gotResult["content"].([]any)[0].(map[string]any)["text"].(string)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": text}}
			s.Equal(test.wantThread, gotThread)
		})
	}
}

func (s *E2ESuite) TestContextDuplicateInOneCall() {
	thread := uuid.NewString()
	tests := []struct {
		name               string
		givenAddRequest    map[string]any
		givenAddHeader     http.Header
		givenThreadRequest map[string]any
		givenThreadHeader  http.Header
		wantStatus         int
		wantAdd            map[string]any
		wantThread         map[string]any
	}{
		{
			name: "one entry at positions 0 and 1",
			givenAddRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " twice in one call"},
					map[string]any{"thread_id": thread, "context": s.topic + " twice in one call"},
				}},
			}},
			givenAddHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			givenThreadRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": thread},
			}},
			givenThreadHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			wantAdd: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"succeeded":[0,1],"failed":[]}`}},
				"structuredContent": map[string]any{"succeeded": []any{float64(0), float64(1)}, "failed": []any{}},
				"isError":           false,
			}},
			wantThread: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"structuredContent": map[string]any{"thread_id": thread, "entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " twice in one call"},
				}},
				"isError": false,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenAddRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenAddHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotAdd map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotAdd))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.wantAdd, gotAdd)

			payload, err = json.Marshal(test.givenThreadRequest)
			s.Require().NoError(err)
			req, err = http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenThreadHeader
			res, err = s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotThread map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotThread))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			gotResult := gotThread["result"].(map[string]any)
			gotEntries, _ := gotResult["structuredContent"].(map[string]any)["entries"].([]any)
			wantResult := test.wantThread["result"].(map[string]any)
			wantEntries := wantResult["structuredContent"].(map[string]any)["entries"].([]any)
			s.Require().Len(gotEntries, len(wantEntries))
			for i, entry := range wantEntries {
				entry.(map[string]any)["created_at"] = gotEntries[i].(map[string]any)["created_at"]
			}
			text, _ := gotResult["content"].([]any)[0].(map[string]any)["text"].(string)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": text}}
			s.Equal(test.wantThread, gotThread)
		})
	}
}

func (s *E2ESuite) TestContextThreadReturnsLatestEntries() {
	thread := uuid.NewString()
	entries := make([]any, 25)
	for i := range entries {
		entries[i] = map[string]any{"thread_id": thread, "context": fmt.Sprintf("%s long thread entry %02d", s.topic, i)}
	}
	latest := make([]any, 20)
	for i := range latest {
		latest[i] = map[string]any{"thread_id": thread, "context": fmt.Sprintf("%s long thread entry %02d", s.topic, i+5)}
	}
	tests := []struct {
		name               string
		givenAddRequests   []map[string]any
		givenAddHeader     http.Header
		givenThreadRequest map[string]any
		givenThreadHeader  http.Header
		wantStatus         int
		wantThread         map[string]any
	}{
		{
			name: "25 entries in calls of 10, 10, 5",
			givenAddRequests: []map[string]any{
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": entries[0:10]}}},
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": entries[10:20]}}},
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": entries[20:25]}}},
			},
			givenAddHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			givenThreadRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": thread},
			}},
			givenThreadHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			wantThread: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"structuredContent": map[string]any{"thread_id": thread, "entries": latest},
				"isError":           false,
			}},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			for _, givenAddRequest := range test.givenAddRequests {
				payload, err := json.Marshal(givenAddRequest)
				s.Require().NoError(err)
				req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
				s.Require().NoError(err)
				req.Header = test.givenAddHeader
				res, err := s.client.Do(req)
				s.Require().NoError(err)
				defer func() { s.NoError(res.Body.Close()) }()
				var gotAdd map[string]any
				s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotAdd))

				s.Require().Equal(test.wantStatus, res.StatusCode)
				s.Require().Equal(false, gotAdd["result"].(map[string]any)["isError"])
			}

			payload, err := json.Marshal(test.givenThreadRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenThreadHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotThread map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotThread))

			s.Require().Equal(test.wantStatus, res.StatusCode)
			gotResult := gotThread["result"].(map[string]any)
			gotEntries, _ := gotResult["structuredContent"].(map[string]any)["entries"].([]any)
			wantResult := test.wantThread["result"].(map[string]any)
			wantEntries := wantResult["structuredContent"].(map[string]any)["entries"].([]any)
			s.Require().Len(gotEntries, len(wantEntries))
			for i, entry := range wantEntries {
				entry.(map[string]any)["created_at"] = gotEntries[i].(map[string]any)["created_at"]
			}
			text, _ := gotResult["content"].([]any)[0].(map[string]any)["text"].(string)
			wantResult["content"] = []any{map[string]any{"type": "text", "text": text}}
			var mirrored map[string]any
			s.Require().NoError(json.Unmarshal([]byte(text), &mirrored))
			s.Equal(wantResult["structuredContent"], mirrored)
			s.Equal(test.wantThread, gotThread)
			s.NoError(s.schemas["get_context_thread"].Validate(gotResult["structuredContent"]))
		})
	}
}

func (s *E2ESuite) TestContextThreadUnknown() {
	thread := uuid.NewString()
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name: "new thread id",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": thread},
			}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"thread_id":"` + thread + `","entries":[]}`}},
				"structuredContent": map[string]any{"thread_id": thread, "entries": []any{}},
				"isError":           false,
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
			s.NoError(s.schemas["get_context_thread"].Validate(got["result"].(map[string]any)["structuredContent"]))
		})
	}
}

func (s *E2ESuite) TestRetrieveContextTopN() {
	thread := uuid.NewString()
	tests := []struct {
		name                string
		givenAddRequest     map[string]any
		givenAddHeader      http.Header
		givenSearchRequests []map[string]any
		givenSearchHeader   http.Header
		wantStatus          int
		wantAdd             map[string]any
		wantTopN            []int
		wantRanked          []any
	}{
		{
			name: "top_n 1 and 20",
			givenAddRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic + " ranked first"},
					map[string]any{"thread_id": thread, "context": s.topic + " ranked second"},
				}},
			}},
			givenAddHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			givenSearchRequests: []map[string]any{
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "retrieve_context", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 1}}},
				{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
					"name": "retrieve_context", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 20}}},
			},
			givenSearchHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_context"}},
			wantStatus: http.StatusOK,
			wantAdd: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType":        "complete",
				"_meta":             serverMeta,
				"content":           []any{map[string]any{"type": "text", "text": `{"succeeded":[0,1],"failed":[]}`}},
				"structuredContent": map[string]any{"succeeded": []any{float64(0), float64(1)}, "failed": []any{}},
				"isError":           false,
			}},
			wantTopN: []int{1, 20},
			wantRanked: []any{
				map[string]any{"thread_id": thread, "context": s.topic + " ranked first"},
				map[string]any{"thread_id": thread, "context": s.topic + " ranked second"},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			payload, err := json.Marshal(test.givenAddRequest)
			s.Require().NoError(err)
			req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
			s.Require().NoError(err)
			req.Header = test.givenAddHeader
			res, err := s.client.Do(req)
			s.Require().NoError(err)
			defer func() { s.NoError(res.Body.Close()) }()
			var gotAdd map[string]any
			s.Require().NoError(json.NewDecoder(res.Body).Decode(&gotAdd))

			s.Equal(test.wantStatus, res.StatusCode)
			s.Require().Equal(test.wantAdd, gotAdd)

			for i, givenSearchRequest := range test.givenSearchRequests {
				ranked := min(test.wantTopN[i], len(test.wantRanked))
				s.EventuallyWithT(func(c *assert.CollectT) {
					payload, err := json.Marshal(givenSearchRequest)
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
					assert.NoError(c, s.schemas["retrieve_context"].Validate(gotStructured))
					gotResults, _ := gotStructured["results"].([]any)
					if !assert.GreaterOrEqual(c, len(gotResults), ranked) {
						return
					}
					assert.LessOrEqual(c, len(gotResults), test.wantTopN[i])
					var gotRanked []any
					for _, result := range gotResults[:ranked] {
						item, _ := result.(map[string]any)
						delete(item, "score")
						gotRanked = append(gotRanked, item)
					}
					assert.Subset(c, test.wantRanked, gotRanked)
				}, searchTimeout, searchTick)
			}
		})
	}
}

func (s *E2ESuite) TestContextInvalidArguments() {
	thread := uuid.NewString()
	elevenEntries := make([]any, 11)
	for i := range elevenEntries {
		elevenEntries[i] = map[string]any{"thread_id": thread, "context": fmt.Sprintf("%s entry %d", s.topic, i)}
	}
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name: "eleven entries",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": elevenEntries}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries': maxItems: got 11, want 10"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries': maxItems: got 11, want 10"},
				"isError": true,
			}},
		},
		{
			name: "no entries",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{}}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries': minItems: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries': minItems: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "thread_id not a uuid",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": "not-a-uuid", "context": s.topic}}}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries/0/thread_id': 'not-a-uuid' is not valid uuid: must have 5 elements"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries/0/thread_id': 'not-a-uuid' is not valid uuid: must have 5 elements"},
				"isError": true,
			}},
		},
		{
			name: "empty context",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": ""}}}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries/0/context': minLength: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries/0/context': minLength: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "extra property",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": map[string]any{"entries": []any{
					map[string]any{"thread_id": thread, "context": s.topic, "note": "x"}}}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries/0': additional properties 'note' not allowed"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '/entries/0': additional properties 'note' not allowed"},
				"isError": true,
			}},
		},
		{
			name: "null arguments",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "add_context", "_meta": meta, "arguments": nil}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"add_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '': got null, want object"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:add_context#'\n- at '': got null, want object"},
				"isError": true,
			}},
		},
		{
			name: "top_n 21",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_context", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 21}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:retrieve_context#'\n- at '/top_n': maximum: got 21, want 20"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:retrieve_context#'\n- at '/top_n': maximum: got 21, want 20"},
				"isError": true,
			}},
		},
		{
			name: "top_n 0",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_context", "_meta": meta, "arguments": map[string]any{"query": s.topic, "top_n": 0}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:retrieve_context#'\n- at '/top_n': minimum: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:retrieve_context#'\n- at '/top_n': minimum: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "empty query",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "retrieve_context", "_meta": meta, "arguments": map[string]any{"query": ""}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"retrieve_context"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:retrieve_context#'\n- at '/query': minLength: got 0, want 1"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:retrieve_context#'\n- at '/query': minLength: got 0, want 1"},
				"isError": true,
			}},
		},
		{
			name: "thread_id abc",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
				"name": "get_context_thread", "_meta": meta, "arguments": map[string]any{"thread_id": "abc"}}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"},
				"Mcp-Name": {"get_context_thread"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"content": []any{map[string]any{"type": "text",
					"text": `{"code":"invalid_argument","message":"invalid args: jsonschema validation failed with 'tool:get_context_thread#'\n- at '/thread_id': 'abc' is not valid uuid: must have 5 elements"}`}},
				"structuredContent": map[string]any{"code": "invalid_argument",
					"message": "invalid args: jsonschema validation failed with 'tool:get_context_thread#'\n- at '/thread_id': 'abc' is not valid uuid: must have 5 elements"},
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
