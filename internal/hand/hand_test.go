package hand

import (
	"math/rand/v2"
	"testing"

	"go-poker-sim/internal/card"
	"go-poker-sim/internal/deck"
)

func mustCards(t *testing.T, tokens ...string) []card.Card {
	t.Helper()
	cards := make([]card.Card, 0, len(tokens))
	for _, tok := range tokens {
		c, err := card.Parse(tok)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tok, err)
		}
		cards = append(cards, c)
	}
	return cards
}

func TestEvaluateCategories(t *testing.T) {
	cases := []struct {
		name     string
		tokens   []string
		category Category
		tiebreak [5]int8
	}{
		{"royal flush", []string{"AH", "KH", "QH", "JH", "10H"}, RoyalFlush, [5]int8{14, 13, 12, 11, 10}},
		{"straight flush 9-high", []string{"5H", "6H", "7H", "8H", "9H"}, StraightFlush, [5]int8{9, 8, 7, 6, 5}},
		{"wheel straight flush", []string{"AH", "2H", "3H", "4H", "5H"}, StraightFlush, [5]int8{5, 4, 3, 2, 1}},
		{"quads with ace kicker", []string{"4D", "4C", "4H", "4S", "AC"}, FourOfAKind, [5]int8{4, 14, 0, 0, 0}},
		{"quads 7 cards kicker is trips rank", []string{"4D", "4C", "4H", "4S", "AC", "AD", "AH"}, FourOfAKind, [5]int8{4, 14, 0, 0, 0}},
		{"full house aces over kings", []string{"AH", "AS", "AD", "KH", "KS"}, FullHouse, [5]int8{14, 13, 0, 0, 0}},
		{"full house from double trips", []string{"AH", "AS", "AD", "KH", "KS", "KD", "2C"}, FullHouse, [5]int8{14, 13, 0, 0, 0}},
		{"full house lower trips still beats higher pair-less trips", []string{"5H", "5S", "5D", "KH", "KS", "KD", "2C"}, FullHouse, [5]int8{13, 5, 0, 0, 0}},
		{"flush uses top 5 of 6 suited", []string{"2H", "4H", "6H", "8H", "10H", "QH", "2D"}, Flush, [5]int8{12, 10, 8, 6, 4}},
		{"flush beats pair", []string{"2H", "4H", "6H", "8H", "10H", "7C", "7S"}, Flush, [5]int8{10, 8, 6, 4, 2}},
		{"straight 9 to K", []string{"KD", "QD", "JS", "10H", "9H"}, Straight, [5]int8{13, 12, 11, 10, 9}},
		{"broadway straight", []string{"AH", "KD", "QC", "JS", "10D"}, Straight, [5]int8{14, 13, 12, 11, 10}},
		{"wheel straight is 5-high", []string{"AH", "2D", "3C", "4S", "5H"}, Straight, [5]int8{5, 4, 3, 2, 1}},
		{"best straight chosen in 7 cards", []string{"9H", "8D", "7C", "6S", "5H", "KD", "KC"}, Straight, [5]int8{9, 8, 7, 6, 5}},
		{"trips with two kickers", []string{"3H", "3S", "3C", "AH", "KD"}, ThreeOfAKind, [5]int8{3, 14, 13, 0, 0}},
		{"two pair with kicker", []string{"8S", "8D", "6H", "6C", "KC"}, TwoPair, [5]int8{8, 6, 13, 0, 0}},
		{"two pair picks two highest pairs", []string{"2H", "2S", "4D", "4C", "8S", "8D", "KC"}, TwoPair, [5]int8{8, 4, 13, 0, 0}},
		{"pair with three kickers", []string{"AS", "AC", "KD", "QH", "JH"}, OnePair, [5]int8{14, 13, 12, 11, 0}},
		{"high card top 5", []string{"AC", "5H", "2S", "7D", "10D", "JH", "9S"}, HighCard, [5]int8{14, 11, 10, 9, 7}},
		{"7-card high card uses top 5", []string{"AC", "5H", "2S", "7D", "10D", "3C", "9S"}, HighCard, [5]int8{14, 10, 9, 7, 5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Evaluate(mustCards(t, tc.tokens...))
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			if r.Category != tc.category {
				t.Errorf("category = %s, want %s", r.Category, tc.category)
			}
			if r.Tiebreak != tc.tiebreak {
				t.Errorf("tiebreak = %v, want %v", r.Tiebreak, tc.tiebreak)
			}
		})
	}
}

func TestEvaluateRejectsBadInput(t *testing.T) {
	if _, err := Evaluate(mustCards(t, "2C", "3D", "4H", "5S")); err == nil {
		t.Error("Evaluate with 4 cards expected error, got nil")
	}
	if _, err := Evaluate(mustCards(t, "2C", "3D", "4H", "5S", "6C", "7D", "8H", "9S")); err == nil {
		t.Error("Evaluate with 8 cards expected error, got nil")
	}
	if _, err := Evaluate(mustCards(t, "AS", "AS", "KD", "QH", "JC")); err == nil {
		t.Error("Evaluate with duplicate cards expected error, got nil")
	}
	if _, err := Evaluate(nil); err == nil {
		t.Error("Evaluate with no cards expected error, got nil")
	}
}

func TestCategoryOrdering(t *testing.T) {
	ordered := []Category{HighCard, OnePair, TwoPair, ThreeOfAKind, Straight, Flush, FullHouse, FourOfAKind, StraightFlush, RoyalFlush}
	for i := 0; i < len(ordered)-1; i++ {
		weaker := Rank{Category: ordered[i], Tiebreak: [5]int8{14, 13, 12, 11, 10}}
		stronger := Rank{Category: ordered[i+1], Tiebreak: [5]int8{2, 0, 0, 0, 0}}
		if weaker.Compare(stronger) >= 0 {
			t.Errorf("%s should lose to %s", ordered[i], ordered[i+1])
		}
		if stronger.Compare(weaker) <= 0 {
			t.Errorf("%s should beat %s", ordered[i+1], ordered[i])
		}
	}
}

func TestRankCompareTiebreaks(t *testing.T) {
	cases := []struct {
		name string
		a, b Rank
		want int
	}{
		{"same rank ties", Rank{OnePair, [5]int8{14, 13, 12, 11, 0}}, Rank{OnePair, [5]int8{14, 13, 12, 11, 0}}, 0},
		{"pair kicker decides", Rank{OnePair, [5]int8{14, 13, 12, 11, 0}}, Rank{OnePair, [5]int8{14, 13, 12, 10, 0}}, 1},
		{"two pair kicker decides", Rank{TwoPair, [5]int8{8, 6, 13, 0, 0}}, Rank{TwoPair, [5]int8{8, 6, 12, 0, 0}}, 1},
		{"two pair higher second pair wins", Rank{TwoPair, [5]int8{8, 7, 2, 0, 0}}, Rank{TwoPair, [5]int8{8, 6, 13, 0, 0}}, 1},
		{"wheel straight loses to 6-high", Rank{Straight, [5]int8{5, 4, 3, 2, 1}}, Rank{Straight, [5]int8{6, 5, 4, 3, 2}}, -1},
		{"broadway beats 9-high straight", Rank{Straight, [5]int8{14, 13, 12, 11, 10}}, Rank{Straight, [5]int8{9, 8, 7, 6, 5}}, 1},
		{"wheel straight flush loses to 6-high straight flush", Rank{StraightFlush, [5]int8{5, 4, 3, 2, 1}}, Rank{StraightFlush, [5]int8{6, 5, 4, 3, 2}}, -1},
		{"flush compares only top 5", Rank{Flush, [5]int8{14, 13, 12, 11, 9}}, Rank{Flush, [5]int8{14, 13, 12, 11, 9}}, 0},
		{"flush 6th card irrelevant", Rank{Flush, [5]int8{14, 13, 12, 11, 9}}, Rank{Flush, [5]int8{14, 13, 12, 11, 9}}, 0},
		{"full house trips decide", Rank{FullHouse, [5]int8{14, 2, 0, 0, 0}}, Rank{FullHouse, [5]int8{13, 14, 0, 0, 0}}, 1},
		{"full house pair decides on equal trips", Rank{FullHouse, [5]int8{14, 13, 0, 0, 0}}, Rank{FullHouse, [5]int8{14, 12, 0, 0, 0}}, 1},
		{"quads kicker decides", Rank{FourOfAKind, [5]int8{4, 14, 0, 0, 0}}, Rank{FourOfAKind, [5]int8{4, 13, 0, 0, 0}}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Compare(tc.b); got != tc.want {
				t.Errorf("Compare() = %d, want %d", got, tc.want)
			}
			if got := tc.b.Compare(tc.a); got != -tc.want {
				t.Errorf("reversed Compare() = %d, want %d", got, -tc.want)
			}
		})
	}
}

func TestEvaluateFlushBeatsStraightWhenBothPresent(t *testing.T) {
	// 5 hearts (flush) and a 6-10 straight; the flush wins.
	r, err := Evaluate(mustCards(t, "2H", "4H", "6H", "8H", "10H", "7C", "9D"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Category != Flush {
		t.Errorf("category = %s, want Flush", r.Category)
	}
	if r.Tiebreak != [5]int8{10, 8, 6, 4, 2} {
		t.Errorf("tiebreak = %v, want [10 8 6 4 2]", r.Tiebreak)
	}
}

func TestEvaluateStraightFlushBeatsQuads(t *testing.T) {
	// 5-9 straight flush plus four of a kind is impossible in 7 cards, but a
	// straight flush must beat a plain flush made with higher cards.
	r, err := Evaluate(mustCards(t, "5H", "6H", "7H", "8H", "9H", "AC", "AD"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Category != StraightFlush {
		t.Errorf("category = %s, want Straight Flush", r.Category)
	}
}

func TestCategoryString(t *testing.T) {
	cases := map[Category]string{
		HighCard: "High Card", OnePair: "Pair", TwoPair: "Two Pair",
		ThreeOfAKind: "Three of a Kind", Straight: "Straight", Flush: "Flush",
		FullHouse: "Full House", FourOfAKind: "Four of a Kind",
		StraightFlush: "Straight Flush", RoyalFlush: "Royal Flush",
	}
	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("Category(%d).String() = %q, want %q", int8(c), got, want)
		}
	}
}

func TestEvaluateCategoryFrequencies(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	const n = 30_000
	counts := make(map[Category]int)
	for i := 0; i < n; i++ {
		d := deck.New()
		d.Shuffle(rng)
		cards := make([]card.Card, 0, 7)
		for j := 0; j < 7; j++ {
			c, err := d.Pop()
			if err != nil {
				t.Fatal(err)
			}
			cards = append(cards, c)
		}
		r, err := Evaluate(cards)
		if err != nil {
			t.Fatal(err)
		}
		counts[r.Category]++
	}
	// Theoretical 7-card hand frequencies (percent).
	want := map[Category]float64{
		HighCard: 17.41, OnePair: 43.82, TwoPair: 23.50, ThreeOfAKind: 4.83,
		Straight: 4.62, Flush: 3.03, FullHouse: 2.60, FourOfAKind: 0.168,
		StraightFlush: 0.0279, RoyalFlush: 0.0032,
	}
	for cat, pct := range want {
		got := 100 * float64(counts[cat]) / n
		if diff := got - pct; diff > 1.5 || diff < -1.5 {
			t.Errorf("%s frequency = %.3f%%, theoretical %.3f%% (diff %+.3f)", cat, got, pct, diff)
		}
	}
}
