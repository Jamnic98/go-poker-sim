package hand

import (
	"fmt"

	"go-poker-sim/internal/card"
)

type Category int8

const (
	HighCard Category = iota + 1
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
	RoyalFlush
)

var categoryNames = map[Category]string{
	HighCard:      "High Card",
	OnePair:       "Pair",
	TwoPair:       "Two Pair",
	ThreeOfAKind:  "Three of a Kind",
	Straight:      "Straight",
	Flush:         "Flush",
	FullHouse:     "Full House",
	FourOfAKind:   "Four of a Kind",
	StraightFlush: "Straight Flush",
	RoyalFlush:    "Royal Flush",
}

func (c Category) String() string {
	if name, ok := categoryNames[c]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", int8(c))
}

const (
	MinCards = 5
	MaxCards = 7
)

// Rank is a fully comparable hand strength: the category plus up to five
// tiebreak ranks in descending comparison order (zero-padded). Two hands are
// tied (a draw / split pot) iff their Ranks are equal. A wheel straight is
// stored as 5-high with the ace counted low (tiebreak {5,4,3,2,1}).
type Rank struct {
	Category Category
	Tiebreak [5]int8
}

// Compare returns 1 if r beats o, -1 if r loses to o, 0 if tied.
func (r Rank) Compare(o Rank) int {
	if r.Category != o.Category {
		if r.Category > o.Category {
			return 1
		}
		return -1
	}
	for i := range r.Tiebreak {
		if r.Tiebreak[i] != o.Tiebreak[i] {
			if r.Tiebreak[i] > o.Tiebreak[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// Evaluate scores 5-7 distinct cards and returns the best 5-card hand rank.
func Evaluate(cards []card.Card) (Rank, error) {
	if len(cards) < MinCards || len(cards) > MaxCards {
		return Rank{}, fmt.Errorf("hand must contain between %d and %d cards, got %d", MinCards, MaxCards, len(cards))
	}

	var rankCount [15]int8
	var suitCount [4]int8
	var rankMask uint16
	seen := make(map[card.Card]struct{}, len(cards))
	for _, c := range cards {
		if _, dup := seen[c]; dup {
			return Rank{}, fmt.Errorf("duplicate card %s", c.Token())
		}
		seen[c] = struct{}{}
		rankCount[c.Rank]++
		suitCount[c.Suit]++
		rankMask |= 1 << c.Rank
	}
	if rankMask&(1<<14) != 0 {
		rankMask |= 1 << 1 // ace also plays low as rank 1 for straights
	}

	flushSuit := card.Suit(-1)
	for s := card.Clubs; s <= card.Spades; s++ {
		if suitCount[s] >= 5 {
			flushSuit = s
			break
		}
	}
	var suitMask uint16
	if flushSuit >= 0 {
		for _, c := range cards {
			if c.Suit == flushSuit {
				suitMask |= 1 << c.Rank
			}
		}
		if suitMask&(1<<14) != 0 {
			suitMask |= 1 << 1 // ace-low bit
		}
		if high := straightHigh(suitMask); high > 0 {
			if high == 14 {
				return Rank{Category: RoyalFlush, Tiebreak: [5]int8{14, 13, 12, 11, 10}}, nil
			}
			return Rank{Category: StraightFlush, Tiebreak: straightTiebreak(high)}, nil
		}
	}

	var quads, trips, pairs []int8
	for r := int8(14); r >= 2; r-- {
		switch rankCount[r] {
		case 4:
			quads = append(quads, r)
		case 3:
			trips = append(trips, r)
		case 2:
			pairs = append(pairs, r)
		}
	}

	topRanksExcluding := func(exclude map[int8]bool, n int) []int8 {
		out := make([]int8, 0, n)
		for r := int8(14); r >= 2 && len(out) < n; r-- {
			if rankCount[r] > 0 && !exclude[r] {
				out = append(out, r)
			}
		}
		return out
	}

	if len(quads) > 0 {
		q := quads[0]
		k := topRanksExcluding(map[int8]bool{q: true}, 1)
		return Rank{Category: FourOfAKind, Tiebreak: [5]int8{q, k[0]}}, nil
	}

	if len(trips) > 0 {
		t := trips[0]
		pairRank := int8(0)
		for r := int8(14); r >= 2; r-- {
			if r != t && rankCount[r] >= 2 {
				pairRank = r
				break
			}
		}
		if pairRank > 0 {
			return Rank{Category: FullHouse, Tiebreak: [5]int8{t, pairRank}}, nil
		}
	}

	if flushSuit >= 0 {
		var tb [5]int8
		i := 0
		for r := int8(14); r >= 2 && i < 5; r-- {
			if suitMask&(1<<r) != 0 {
				tb[i] = r
				i++
			}
		}
		return Rank{Category: Flush, Tiebreak: tb}, nil
	}

	if high := straightHigh(rankMask); high > 0 {
		return Rank{Category: Straight, Tiebreak: straightTiebreak(high)}, nil
	}

	if len(trips) > 0 {
		t := trips[0]
		ks := topRanksExcluding(map[int8]bool{t: true}, 2)
		return Rank{Category: ThreeOfAKind, Tiebreak: [5]int8{t, ks[0], ks[1]}}, nil
	}

	if len(pairs) >= 2 {
		p1, p2 := pairs[0], pairs[1]
		ks := topRanksExcluding(map[int8]bool{p1: true, p2: true}, 1)
		return Rank{Category: TwoPair, Tiebreak: [5]int8{p1, p2, ks[0]}}, nil
	}

	if len(pairs) == 1 {
		p := pairs[0]
		ks := topRanksExcluding(map[int8]bool{p: true}, 3)
		return Rank{Category: OnePair, Tiebreak: [5]int8{p, ks[0], ks[1], ks[2]}}, nil
	}

	var tb [5]int8
	i := 0
	for r := int8(14); r >= 2 && i < 5; r-- {
		if rankMask&(1<<r) != 0 {
			tb[i] = r
			i++
		}
	}
	return Rank{Category: HighCard, Tiebreak: tb}, nil
}

// straightHigh returns the high rank of the best 5-card straight in mask,
// or 0 if there is none. Bit 1 holds the ace-low value, so the wheel (5-high)
// is detected as high=5.
func straightHigh(mask uint16) int8 {
	for high := int8(14); high >= 5; high-- {
		window := uint16(0x1F) << (high - 4)
		if mask&window == window {
			return high
		}
	}
	return 0
}

func straightTiebreak(high int8) [5]int8 {
	return [5]int8{high, high - 1, high - 2, high - 3, high - 4}
}
