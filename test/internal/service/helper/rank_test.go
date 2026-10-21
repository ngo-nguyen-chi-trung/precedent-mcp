// Test plan: rank.go
//
// Logic flow
// | Function                     | Step | Action                                                                      | Effect                                                   |
// |------------------------------|------|-----------------------------------------------------------------------------|----------------------------------------------------------|
// | WeightedReciprocalRankFusion | 1    | Order each list by score, highest first                                     | Equal scores keep the given order                        |
// | WeightedReciprocalRankFusion | 2    | Position p (from 1) gets the share weight ÷ (RankReciprocalConstant 60 + p) | A higher position or a heavier list gives a larger share |
// | WeightedReciprocalRankFusion | 3    | Add each ID's shares across all lists                                       | An ID missing from a list gets nothing from that list    |
// | WeightedReciprocalRankFusion | 4    | Divide each sum by the best possible sum: the first position in every list  | Score from 0 to 1; 1 means first in every list           |
// | WeightedReciprocalRankFusion | 5    | Return the score per ID                                                     | No entries give an empty result                          |
//
// Scenarios
// | Number | Function                     | Result  | Value class    | Input                                          | Expected                                       |
// |--------|------------------------------|---------|----------------|------------------------------------------------|------------------------------------------------|
// | 1      | WeightedReciprocalRankFusion | Success | Boundary value | One list, weight 2: a 0.9, b 0.5               | a 1, b 61/62 (steps 1, 2, 4)                   |
// | 2      | WeightedReciprocalRankFusion | Success | Middle value   | One list, weight 2, lowest first: a 0.2, b 0.8 | b 1, a 61/62 (step 1)                          |
// | 3      | WeightedReciprocalRankFusion | Success | Boundary value | One list, weight 2: a 0.5, b 0.5               | a 1, b 61/62; given order kept (step 1)        |
// | 4      | WeightedReciprocalRankFusion | Success | Middle value   | Weight 2: a 0.9, b 0.5; weight 3: a 1, b 5     | a 307/310, b 308/310; b above a (steps 2 to 4) |
// | 5      | WeightedReciprocalRankFusion | Success | Boundary value | Weight 2: a 0.9, b 0.5; weight 3: a 5 only     | a 1, b 61/155 (steps 3, 4)                     |
// | 6      | WeightedReciprocalRankFusion | Success | Boundary value | No lists                                       | Empty result (step 5)                          |
// | 7      | WeightedReciprocalRankFusion | Success | Boundary value | One list without entries                       | Empty result (step 5)                          |
//
// Failure scenarios: none. WeightedReciprocalRankFusion has no error path.
//
// Branch coverage
// | Function                     | Condition                                | True | False |
// |------------------------------|------------------------------------------|------|-------|
// | WeightedReciprocalRankFusion | Score order differs from the given order | 2    | 1     |
// | WeightedReciprocalRankFusion | Equal scores                             | 3    | 1     |
// | WeightedReciprocalRankFusion | ID present in more than one list         | 4    | 5     |
// | WeightedReciprocalRankFusion | Lists with entries given                 | 1    | 6, 7  |
//
// Test mapping
// | Test                                                                     | Scenarios |
// |--------------------------------------------------------------------------|-----------|
// | TestWeightedReciprocalRankFusion/higher score takes the first position   | 1         |
// | TestWeightedReciprocalRankFusion/lowest first input is reordered         | 2         |
// | TestWeightedReciprocalRankFusion/equal scores keep the given order       | 3         |
// | TestWeightedReciprocalRankFusion/heavier list outranks a better position | 4         |
// | TestWeightedReciprocalRankFusion/identifier missing from a list          | 5         |
// | TestWeightedReciprocalRankFusion/no lists                                | 6         |
// | TestWeightedReciprocalRankFusion/list without entries                    | 7         |
// | No test                                                                  | None      |

package helper_test

import . "github.com/ngo/precedent_mcp/internal/service/helper"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeightedReciprocalRankFusion(t *testing.T) {
	tests := []struct {
		name       string
		givenLists []RankedList[string]
		want       map[string]float64
	}{
		{
			name: "higher score takes the first position",
			givenLists: []RankedList[string]{
				{Weight: 2, Entries: []RankEntry[string]{{ID: "a", Score: 0.9}, {ID: "b", Score: 0.5}}},
			},
			want: map[string]float64{"a": 1, "b": 61.0 / 62},
		},
		{
			name: "lowest first input is reordered",
			givenLists: []RankedList[string]{
				{Weight: 2, Entries: []RankEntry[string]{{ID: "a", Score: 0.2}, {ID: "b", Score: 0.8}}},
			},
			want: map[string]float64{"a": 61.0 / 62, "b": 1},
		},
		{
			name: "equal scores keep the given order",
			givenLists: []RankedList[string]{
				{Weight: 2, Entries: []RankEntry[string]{{ID: "a", Score: 0.5}, {ID: "b", Score: 0.5}}},
			},
			want: map[string]float64{"a": 1, "b": 61.0 / 62},
		},
		{
			name: "heavier list outranks a better position",
			givenLists: []RankedList[string]{
				{Weight: 2, Entries: []RankEntry[string]{{ID: "a", Score: 0.9}, {ID: "b", Score: 0.5}}},
				{Weight: 3, Entries: []RankEntry[string]{{ID: "a", Score: 1}, {ID: "b", Score: 5}}},
			},
			want: map[string]float64{"a": 307.0 / 310, "b": 308.0 / 310},
		},
		{
			name: "identifier missing from a list",
			givenLists: []RankedList[string]{
				{Weight: 2, Entries: []RankEntry[string]{{ID: "a", Score: 0.9}, {ID: "b", Score: 0.5}}},
				{Weight: 3, Entries: []RankEntry[string]{{ID: "a", Score: 5}}},
			},
			want: map[string]float64{"a": 1, "b": 61.0 / 155},
		},
		{
			name:       "no lists",
			givenLists: nil,
			want:       map[string]float64{},
		},
		{
			name:       "list without entries",
			givenLists: []RankedList[string]{{Weight: 2}},
			want:       map[string]float64{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := WeightedReciprocalRankFusion(test.givenLists...)

			assert.InDeltaMapValues(t, test.want, got, 1e-12)
		})
	}
}
