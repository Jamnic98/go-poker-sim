package evaluator

import (
	"reflect"
	"testing"

	"go-poker-sim/internal/card"
	"go-poker-sim/internal/hand"
)

func ranksOf(t *testing.T, hands ...[]string) []hand.Rank {
	t.Helper()
	ranks := make([]hand.Rank, 0, len(hands))
	for _, tokens := range hands {
		cards := make([]card.Card, 0, len(tokens))
		for _, tok := range tokens {
			c, err := card.Parse(tok)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tok, err)
			}
			cards = append(cards, c)
		}
		r, err := hand.Evaluate(cards)
		if err != nil {
			t.Fatalf("Evaluate(%v): %v", tokens, err)
		}
		ranks = append(ranks, r)
	}
	return ranks
}

func TestRankHandsOrdersByStrength(t *testing.T) {
	ranks := ranksOf(t,
		[]string{"2C", "3D", "5H", "9S", "JC"}, // high card J
		[]string{"KC", "KD", "5H", "9S", "JC"}, // pair kings
		[]string{"AC", "AD", "5H", "9S", "JC"}, // pair aces
	)
	groups := RankHands(ranks)
	want := [][]int{{2}, {1}, {0}}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("RankHands() = %v, want %v", groups, want)
	}
}

func TestWinnersSingleWinner(t *testing.T) {
	ranks := ranksOf(t,
		[]string{"AC", "AD", "5H", "9S", "JC"},
		[]string{"KC", "KD", "5H", "9S", "JC"},
		[]string{"2C", "3D", "5H", "9S", "JC"},
	)
	if got := Winners(ranks); !reflect.DeepEqual(got, []int{0}) {
		t.Errorf("Winners() = %v, want [0]", got)
	}
}

func TestWinnersSplitPotBoardStraight(t *testing.T) {
	// Both players play the board straight 5-9; hole cards are irrelevant.
	ranks := ranksOf(t,
		[]string{"5D", "6C", "7H", "8S", "9D", "2C", "3H"},
		[]string{"5D", "6C", "7H", "8S", "9D", "AC", "KH"},
	)
	got := Winners(ranks)
	if !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("Winners() = %v, want [0 1] (split pot)", got)
	}
}

func TestWinnersSplitPotBoardPair(t *testing.T) {
	// Board pair of aces with K Q J kickers; both players play the board.
	ranks := ranksOf(t,
		[]string{"AH", "AD", "KD", "QS", "JC", "2C", "3H"},
		[]string{"AH", "AD", "KD", "QS", "JC", "4S", "5D"},
	)
	got := Winners(ranks)
	if !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("Winners() = %v, want [0 1] (split pot)", got)
	}
}

func TestWinnersSplitPotIdenticalRanksDifferentSuits(t *testing.T) {
	ranks := ranksOf(t,
		[]string{"AH", "KH", "QH", "JH", "10H"}, // royal flush hearts
		[]string{"AS", "KS", "QS", "JS", "10S"}, // royal flush spades
	)
	got := Winners(ranks)
	if !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("Winners() = %v, want [0 1] (suits do not rank)", got)
	}
}

func TestWinnersKickerDecidesTwoPair(t *testing.T) {
	ranks := ranksOf(t,
		[]string{"8S", "8D", "6H", "6C", "KC", "2D", "3H"}, // two pair 8s+6s, K kicker
		[]string{"8C", "8H", "6D", "6S", "QC", "4D", "5H"}, // two pair 8s+6s, Q kicker
	)
	if got := Winners(ranks); !reflect.DeepEqual(got, []int{0}) {
		t.Errorf("Winners() = %v, want [0] (king kicker beats queen)", got)
	}
}

func TestWinnersWheelStraightLosesToSixHigh(t *testing.T) {
	ranks := ranksOf(t,
		[]string{"AH", "2D", "3C", "4S", "5H", "9C", "KD"}, // wheel
		[]string{"6H", "2D", "3C", "4S", "5H", "9C", "KD"}, // 6-high straight
	)
	if got := Winners(ranks); !reflect.DeepEqual(got, []int{1}) {
		t.Errorf("Winners() = %v, want [1] (6-high beats wheel)", got)
	}
}

func TestWinnersThreeWaySplit(t *testing.T) {
	// Board: 9 10 J Q K rainbow straight; all three players play the board.
	board := []string{"9D", "10C", "JH", "QS", "KD"}
	hands := [][]string{
		append(append([]string{}, board...), "2C", "3H"),
		append(append([]string{}, board...), "4S", "5D"),
		append(append([]string{}, board...), "6C", "7D"),
	}
	ranks := ranksOf(t, hands...)
	got := Winners(ranks)
	if !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Errorf("Winners() = %v, want [0 1 2] (three-way split)", got)
	}
}

func TestWinnersEmpty(t *testing.T) {
	if got := Winners(nil); got != nil {
		t.Errorf("Winners(nil) = %v, want nil", got)
	}
}

func TestRankHandsGroupsAllTied(t *testing.T) {
	ranks := ranksOf(t,
		[]string{"2C", "3D", "5H", "9S", "JC"},
		[]string{"AC", "AD", "5H", "9S", "JC"},
		[]string{"KC", "KD", "5H", "9S", "JC"},
		[]string{"AS", "AH", "5H", "9S", "JC"}, // ties with index 1
	)
	groups := RankHands(ranks)
	want := [][]int{{1, 3}, {2}, {0}}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("RankHands() = %v, want %v", groups, want)
	}
}
