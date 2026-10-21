//go:build e2e

// Test plan: e2e skill loading business flow
//
// A business client lists the skills, then loads the instructions and examples of the skill it follows.
//
// Logic flow
// | Step | Client call     | Path                                                                           | Expected                                                                                     |
// |------|-----------------|--------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------|
// | K1   | skills/list     | SkillService.ListSkills; SkillStore.ListMetadata of asset.Skills               | HTTP 200; one entry per skill; one skill:// resource with digest and size from metadata.yaml |
// | K2   | skills/get      | SkillService.GetSkill: SkillURI step; SkillStore.Metadata                      | HTTP 200; entry equal to the K1 entry; public cache                                          |
// | K3   | resources/read  | ResourceService.ReadResource: ResourceURI step; SkillStore.Skill               | HTTP 200; text/markdown; sha256 digest and length equal to the K1 digest and size            |
// | K4   | resources/read  | ResourceService.ReadResource: ResourceURI step; ExampleStore.Examples          | HTTP 200; text/markdown; text not empty                                                      |
// | K5   | resources/list  | ResourceService.ListResources; SkillStore.ListMetadata; ExampleStore.ListNames | HTTP 200; skill resources, then examples resources with a description and audience [user]    |
// | K6   | Any of K2 to K4 | Name not in the skill or examples map                                          | HTTP 400; -32602 "resource not found"; data.uri equal to the uri                             |
// | K7   | Any of K2 to K4 | Scheme not served by the method                                                | HTTP 400; -32602 "unsupported resource scheme"; data.uri equal to the uri                    |
// | K8   | Any of K2 to K4 | Empty name                                                                     | HTTP 400; -32602 "resource URI has no name"; data.uri equal to the uri                       |
// | K9   | Any of K2 to K4 | URI with a path                                                                | HTTP 400; -32602 "resource URI must not have a path"; data.uri equal to the uri              |
//
// Scenarios
// | Number | Step | Result  | Value class    | Input                                                            | Expected                      |
// |--------|------|---------|----------------|------------------------------------------------------------------|-------------------------------|
// | 1      | K1   | Success | Middle value   | skills/list                                                      | As in K1 for the 5 skills     |
// | 2      | K2   | Success | Middle value   | skills/get for each of the 5 skills                              | As in K2                      |
// | 3      | K3   | Success | Middle value   | resources/read skill:// of each skill, Mcp-Name equal            | As in K3                      |
// | 4      | K4   | Success | Middle value   | resources/read examples:// of each skill, Mcp-Name equal         | As in K4                      |
// | 5      | K5   | Success | Middle value   | resources/list                                                   | As in K5 for the 10 resources |
// | 6      | K2   | Failure | Middle value   | skill://nope                                                     | As in K6                      |
// | 7      | K2   | Failure | Middle value   | examples://precedent-recall                                      | As in K7                      |
// | 8      | K3   | Failure | Middle value   | skill://nope                                                     | As in K6                      |
// | 9      | K4   | Failure | Middle value   | examples://nope                                                  | As in K6                      |
// | 10     | K3   | Failure | Boundary value | skill://; empty skill name                                       | As in K8                      |
// | 11     | K3   | Failure | Boundary value | skill://precedent-recall/SKILL.md                                | As in K9                      |
// | 12     | K4   | Failure | Boundary value | examples://precedent-recall/examples.md                          | As in K9                      |
// | 13     | K3   | Failure | Middle value   | file:///precedent-recall/SKILL.md                                | As in K7                      |
// | 14     | K3   | Failure | Middle value   | skill://precedent-recall with Mcp-Name skill://precedent-capture | HTTP 400; -32020              |
//
// Branch coverage
// | Step       | Condition                       | True    | False   |
// |------------|---------------------------------|---------|---------|
// | K2         | URI scheme is skill             | 2       | 7       |
// | K3, K4     | URI scheme is skill or examples | 3, 4    | 13      |
// | K3, K4     | Name not empty                  | 3, 4    | 10      |
// | K3, K4     | Path empty                      | 3, 4    | 11, 12  |
// | K3, K4     | URI scheme is examples          | 4       | 3       |
// | K2, K3, K4 | Name is a map key               | 2, 3, 4 | 6, 8, 9 |
// | K5         | Resource is examples            | 5       | 5       |
// | K3         | Mcp-Name sent and mirrors uri   | 3       | 14      |
//
// Test mapping
// | Test                                                                    | Scenarios |
// |-------------------------------------------------------------------------|-----------|
// | TestSkillsList/five skills                                              | 1         |
// | TestSkillsGet/precedent-capture to TestSkillsGet/precedent-review       | 2         |
// | TestReadSkill/precedent-capture to TestReadSkill/precedent-review       | 3         |
// | TestReadExamples/precedent-capture to TestReadExamples/precedent-review | 4         |
// | TestResourcesList/skill and examples resources                          | 5         |
// | TestResourceURIRejected/skills get unknown skill                        | 6         |
// | TestResourceURIRejected/skills get examples uri                         | 7         |
// | TestResourceURIRejected/read unknown skill                              | 8         |
// | TestResourceURIRejected/read unknown examples                           | 9         |
// | TestResourceURIRejected/read empty skill name                           | 10        |
// | TestResourceURIRejected/read skill file path                            | 11        |
// | TestResourceURIRejected/read examples file path                         | 12        |
// | TestResourceURIRejected/read file scheme                                | 13        |
// | TestResourcesReadMcpNameMismatch/Mcp-Name names another skill           | 14        |
// | No test                                                                 | None      |

package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

var skillEntries = []any{
	map[string]any{
		"uri":         "skill://precedent-capture",
		"frontmatter": map[string]any{"name": "precedent-capture", "description": "Save durable knowledge from the conversation"},
		"resources": []any{map[string]any{"uri": "skill://precedent-capture",
			"digest": "sha256:99add05c9551d7354437de589b7872effed4f40447d64a70aa0ba540f7a48bcb", "size": float64(2943)}},
	},
	map[string]any{
		"uri":         "skill://precedent-decide",
		"frontmatter": map[string]any{"name": "precedent-decide", "description": "Draft a decision record for a design choice"},
		"resources": []any{map[string]any{"uri": "skill://precedent-decide",
			"digest": "sha256:d8fd7411354d4e8babaa04e41c95748714b069a52273845e01dd32abede5bd22", "size": float64(1878)}},
	},
	map[string]any{
		"uri":         "skill://precedent-recall",
		"frontmatter": map[string]any{"name": "precedent-recall", "description": "Load earlier context when a session starts"},
		"resources": []any{map[string]any{"uri": "skill://precedent-recall",
			"digest": "sha256:88c855e77063f66899d23ce0788705bf3f7af18d6dd0eaa99048fae71646dd28", "size": float64(1449)}},
	},
	map[string]any{
		"uri":         "skill://precedent-refine",
		"frontmatter": map[string]any{"name": "precedent-refine", "description": "Revise a proposed decision until it is decided"},
		"resources": []any{map[string]any{"uri": "skill://precedent-refine",
			"digest": "sha256:ab4803bc676c2620a417ec719275d34685cb812d3378ab3c77c42c0c5afb01ff", "size": float64(1913)}},
	},
	map[string]any{
		"uri":         "skill://precedent-review",
		"frontmatter": map[string]any{"name": "precedent-review", "description": "Score how an accepted decision held up"},
		"resources": []any{map[string]any{"uri": "skill://precedent-review",
			"digest": "sha256:b56bcb25b5ec8c66f9709fb142f3e15e5499bfac4c8163b48db93d250871832f", "size": float64(1513)}},
	},
}

func (s *E2ESuite) TestSkillsList() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "five skills",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/list", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/list"}},
			wantStatus:   http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"ttlMs":      float64(300000),
				"cacheScope": "public",
				"skills":     skillEntries,
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
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestSkillsGet() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "precedent-capture",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "skill://precedent-capture", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusOK,
			want:         map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta, "ttlMs": float64(300000), "cacheScope": "public", "skill": skillEntries[0]}},
		},
		{
			name:         "precedent-decide",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "skill://precedent-decide", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusOK,
			want:         map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta, "ttlMs": float64(300000), "cacheScope": "public", "skill": skillEntries[1]}},
		},
		{
			name:         "precedent-recall",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "skill://precedent-recall", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusOK,
			want:         map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta, "ttlMs": float64(300000), "cacheScope": "public", "skill": skillEntries[2]}},
		},
		{
			name:         "precedent-refine",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "skill://precedent-refine", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusOK,
			want:         map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta, "ttlMs": float64(300000), "cacheScope": "public", "skill": skillEntries[3]}},
		},
		{
			name:         "precedent-review",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "skill://precedent-review", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusOK,
			want:         map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta, "ttlMs": float64(300000), "cacheScope": "public", "skill": skillEntries[4]}},
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
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestReadSkill() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		wantDigest   string
		wantSize     int
		want         map[string]any
	}{
		{
			name:         "precedent-capture",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-capture", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-capture"}},
			wantStatus: http.StatusOK,
			wantDigest: "sha256:99add05c9551d7354437de589b7872effed4f40447d64a70aa0ba540f7a48bcb",
			wantSize:   2943,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "skill://precedent-capture", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-decide",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-decide", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-decide"}},
			wantStatus: http.StatusOK,
			wantDigest: "sha256:d8fd7411354d4e8babaa04e41c95748714b069a52273845e01dd32abede5bd22",
			wantSize:   1878,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "skill://precedent-decide", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-recall",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-recall", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-recall"}},
			wantStatus: http.StatusOK,
			wantDigest: "sha256:88c855e77063f66899d23ce0788705bf3f7af18d6dd0eaa99048fae71646dd28",
			wantSize:   1449,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "skill://precedent-recall", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-refine",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-refine", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-refine"}},
			wantStatus: http.StatusOK,
			wantDigest: "sha256:ab4803bc676c2620a417ec719275d34685cb812d3378ab3c77c42c0c5afb01ff",
			wantSize:   1913,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "skill://precedent-refine", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-review",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-review", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-review"}},
			wantStatus: http.StatusOK,
			wantDigest: "sha256:b56bcb25b5ec8c66f9709fb142f3e15e5499bfac4c8163b48db93d250871832f",
			wantSize:   1513,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "skill://precedent-review", "mimeType": "text/markdown"}}}},
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
			text, _ := got["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)["text"].(string)
			sum := sha256.Sum256([]byte(text))
			s.Equal(test.wantDigest, "sha256:"+hex.EncodeToString(sum[:]))
			s.Equal(test.wantSize, len(text))
			test.want["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)["text"] = text
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestReadExamples() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "precedent-capture",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://precedent-capture", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://precedent-capture"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "examples://precedent-capture", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-decide",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://precedent-decide", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://precedent-decide"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "examples://precedent-decide", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-recall",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://precedent-recall", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://precedent-recall"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "examples://precedent-recall", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-refine",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://precedent-refine", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://precedent-refine"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "examples://precedent-refine", "mimeType": "text/markdown"}}}},
		},
		{
			name:         "precedent-review",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://precedent-review", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://precedent-review"}},
			wantStatus: http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{"resultType": "complete", "_meta": serverMeta,
				"ttlMs": float64(300000), "cacheScope": "public",
				"contents": []any{map[string]any{"uri": "examples://precedent-review", "mimeType": "text/markdown"}}}},
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
			text, _ := got["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)["text"].(string)
			s.NotEmpty(text)
			test.want["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)["text"] = text
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestResourcesList() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "skill and examples resources",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/list", "params": map[string]any{"_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/list"}},
			wantStatus:   http.StatusOK,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "result": map[string]any{
				"resultType": "complete",
				"_meta":      serverMeta,
				"ttlMs":      float64(300000),
				"cacheScope": "public",
				"resources": []any{
					map[string]any{"uri": "skill://precedent-capture", "name": "precedent-capture",
						"description": "Save durable knowledge from the conversation", "mimeType": "text/markdown", "size": float64(2943)},
					map[string]any{"uri": "skill://precedent-decide", "name": "precedent-decide",
						"description": "Draft a decision record for a design choice", "mimeType": "text/markdown", "size": float64(1878)},
					map[string]any{"uri": "skill://precedent-recall", "name": "precedent-recall",
						"description": "Load earlier context when a session starts", "mimeType": "text/markdown", "size": float64(1449)},
					map[string]any{"uri": "skill://precedent-refine", "name": "precedent-refine",
						"description": "Revise a proposed decision until it is decided", "mimeType": "text/markdown", "size": float64(1913)},
					map[string]any{"uri": "skill://precedent-review", "name": "precedent-review",
						"description": "Score how an accepted decision held up", "mimeType": "text/markdown", "size": float64(1513)},
					map[string]any{"uri": "examples://precedent-capture", "name": "precedent-capture", "mimeType": "text/markdown",
						"description": "Examples for the precedent-capture skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.",
						"annotations": map[string]any{"audience": []any{"user"}}},
					map[string]any{"uri": "examples://precedent-decide", "name": "precedent-decide", "mimeType": "text/markdown",
						"description": "Examples for the precedent-decide skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.",
						"annotations": map[string]any{"audience": []any{"user"}}},
					map[string]any{"uri": "examples://precedent-recall", "name": "precedent-recall", "mimeType": "text/markdown",
						"description": "Examples for the precedent-recall skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.",
						"annotations": map[string]any{"audience": []any{"user"}}},
					map[string]any{"uri": "examples://precedent-refine", "name": "precedent-refine", "mimeType": "text/markdown",
						"description": "Examples for the precedent-refine skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.",
						"annotations": map[string]any{"audience": []any{"user"}}},
					map[string]any{"uri": "examples://precedent-review", "name": "precedent-review", "mimeType": "text/markdown",
						"description": "Examples for the precedent-review skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md.",
						"annotations": map[string]any{"audience": []any{"user"}}},
				},
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
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestResourceURIRejected() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "skills get unknown skill",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "skill://nope", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "resource not found", "data": map[string]any{"uri": "skill://nope"}}},
		},
		{
			name:         "skills get examples uri",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "skills/get", "params": map[string]any{"uri": "examples://precedent-recall", "_meta": meta}},
			givenHeader:  http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"skills/get"}},
			wantStatus:   http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "unsupported resource scheme", "data": map[string]any{"uri": "examples://precedent-recall"}}},
		},
		{
			name:         "read unknown skill",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://nope", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://nope"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "resource not found", "data": map[string]any{"uri": "skill://nope"}}},
		},
		{
			name:         "read unknown examples",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://nope", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://nope"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "resource not found", "data": map[string]any{"uri": "examples://nope"}}},
		},
		{
			name:         "read empty skill name",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "resource URI has no name", "data": map[string]any{"uri": "skill://"}}},
		},
		{
			name:         "read skill file path",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-recall/SKILL.md", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-recall/SKILL.md"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "resource URI must not have a path", "data": map[string]any{"uri": "skill://precedent-recall/SKILL.md"}}},
		},
		{
			name:         "read examples file path",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "examples://precedent-recall/examples.md", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"examples://precedent-recall/examples.md"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "resource URI must not have a path", "data": map[string]any{"uri": "examples://precedent-recall/examples.md"}}},
		},
		{
			name:         "read file scheme",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "file:///precedent-recall/SKILL.md", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"file:///precedent-recall/SKILL.md"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1), "error": map[string]any{
				"code": float64(-32602), "message": "unsupported resource scheme", "data": map[string]any{"uri": "file:///precedent-recall/SKILL.md"}}},
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
			s.Equal(test.want, got)
		})
	}
}

func (s *E2ESuite) TestResourcesReadMcpNameMismatch() {
	tests := []struct {
		name         string
		givenRequest map[string]any
		givenHeader  http.Header
		wantStatus   int
		want         map[string]any
	}{
		{
			name:         "Mcp-Name names another skill",
			givenRequest: map[string]any{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": map[string]any{"uri": "skill://precedent-recall", "_meta": meta}},
			givenHeader: http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"resources/read"},
				"Mcp-Name": {"skill://precedent-capture"}},
			wantStatus: http.StatusBadRequest,
			want: map[string]any{"jsonrpc": "2.0", "id": float64(1),
				"error": map[string]any{"code": float64(-32020), "message": "MCP headers missing or do not match the request body"}},
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
			s.Equal(test.want, got)
		})
	}
}
