package evaluator

import (
	"sort"

	"go-poker-sim/internal/hand"
)

// RankHands groups hand indexes by strength, strongest group first. Every
// index within a group is tied with the others in that group.
func RankHands(ranks []hand.Rank) [][]int {
	idx := make([]int, len(ranks))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		return ranks[idx[a]].Compare(ranks[idx[b]]) > 0
	})

	var groups [][]int
	for _, i := range idx {
		last := len(groups) - 1
		if last >= 0 && ranks[groups[last][0]].Compare(ranks[i]) == 0 {
			groups[last] = append(groups[last], i)
		} else {
			groups = append(groups, []int{i})
		}
	}
	return groups
}

// Winners returns the indexes of the winning hands. More than one index
// means a split pot (draw).
func Winners(ranks []hand.Rank) []int {
	if len(ranks) == 0 {
		return nil
	}
	return RankHands(ranks)[0]
}
