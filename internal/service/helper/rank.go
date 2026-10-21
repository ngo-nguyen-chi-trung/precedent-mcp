package helper

import (
	"cmp"
	"slices"

	"github.com/ngo/precedent_mcp/internal/env"
)

type RankEntry[K comparable] struct {
	ID             K
	Score          float64
	ReciprocalRank float64
}

type RankedList[K comparable] struct {
	Entries []RankEntry[K]
	Weight  env.RankWeight
}

func WeightedReciprocalRankFusion[K comparable](lists ...RankedList[K]) map[K]float64 {
	ranked := make([][]RankEntry[K], 0, len(lists))
	var best float64
	for _, list := range lists {
		ranked = append(ranked, reciprocalRanks(list.Entries, list.Weight))
		best += float64(list.Weight) / float64(env.RankReciprocalConstant+1)
	}
	fused := mergeReciprocalRanks(ranked...)
	for id := range fused {
		fused[id] /= best
	}
	return fused
}

func reciprocalRanks[K comparable](entries []RankEntry[K], weight env.RankWeight) []RankEntry[K] {
	slices.SortStableFunc(entries, func(a, b RankEntry[K]) int {
		return cmp.Compare(b.Score, a.Score)
	})
	for position := range entries {
		entries[position].ReciprocalRank = float64(weight) / (float64(env.RankReciprocalConstant) + float64(position+1))
	}
	return entries
}

func mergeReciprocalRanks[K comparable](lists ...[]RankEntry[K]) map[K]float64 {
	fused := make(map[K]float64)
	for _, entries := range lists {
		for _, e := range entries {
			fused[e.ID] += e.ReciprocalRank
		}
	}
	return fused
}
