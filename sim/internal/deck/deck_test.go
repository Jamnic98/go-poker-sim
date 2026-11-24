package deck

import (
	"math/rand/v2"
	"testing"

	"go-poker-sim/internal/card"
)

func TestNewDeckHas52UniqueCards(t *testing.T) {
	d := New()
	if d.Len() != 52 {
		t.Fatalf("New() deck len = %d, want 52", d.Len())
	}
	seen := make(map[card.Card]int)
	suitCount := make(map[card.Suit]int)
	rankCount := make(map[int8]int)
	for _, c := range d.Cards {
		seen[c]++
		suitCount[c.Suit]++
		rankCount[c.Rank]++
	}
	if len(seen) != 52 {
		t.Errorf("expected 52 unique cards, got %d", len(seen))
	}
	for s := card.Clubs; s <= card.Spades; s++ {
		if suitCount[s] != 13 {
			t.Errorf("suit %s count = %d, want 13", s.Token(), suitCount[s])
		}
	}
	for r := card.MinRank; r <= card.MaxRank; r++ {
		if rankCount[r] != 4 {
			t.Errorf("rank %d count = %d, want 4", r, rankCount[r])
		}
	}
}

func TestShuffleIsAPermutation(t *testing.T) {
	fresh := New()
	before := make(map[card.Card]int)
	for _, c := range fresh.Cards {
		before[c]++
	}

	d := New()
	d.Shuffle(rand.New(rand.NewPCG(1, 2)))

	if d.Len() != 52 {
		t.Fatalf("shuffled deck len = %d, want 52", d.Len())
	}
	after := make(map[card.Card]int)
	for _, c := range d.Cards {
		after[c]++
	}
	for c, n := range before {
		if after[c] != n {
			t.Errorf("card %s count changed after shuffle: %d -> %d", c.Token(), n, after[c])
		}
	}
	changed := false
	for i := range d.Cards {
		if d.Cards[i] != fresh.Cards[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("shuffle did not change the deck order")
	}
}

func TestShuffleDeterministicForSameSeed(t *testing.T) {
	d1 := New()
	d2 := New()
	d1.Shuffle(rand.New(rand.NewPCG(42, 7)))
	d2.Shuffle(rand.New(rand.NewPCG(42, 7)))
	for i := range d1.Cards {
		if d1.Cards[i] != d2.Cards[i] {
			t.Fatalf("shuffle not deterministic: cards differ at index %d", i)
		}
	}
}

func TestPop(t *testing.T) {
	d := New()
	want := d.Cards[len(d.Cards)-1]
	c, err := d.Pop()
	if err != nil {
		t.Fatalf("Pop() unexpected error: %v", err)
	}
	if c != want {
		t.Errorf("Pop() = %s, want top card %s", c.Token(), want.Token())
	}
	if d.Len() != 51 {
		t.Errorf("deck len after Pop = %d, want 51", d.Len())
	}
	for d.Len() > 0 {
		if _, err := d.Pop(); err != nil {
			t.Fatalf("Pop() unexpected error with %d cards left: %v", d.Len()+1, err)
		}
	}
	if _, err := d.Pop(); err == nil {
		t.Error("Pop() on empty deck expected error, got nil")
	}
}

func TestReset(t *testing.T) {
	d := New()
	if _, err := d.Pop(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Pop(); err != nil {
		t.Fatal(err)
	}
	d.Reset()
	if d.Len() != 52 {
		t.Errorf("deck len after Reset = %d, want 52", d.Len())
	}
	fresh := New()
	for i := range d.Cards {
		if d.Cards[i] != fresh.Cards[i] {
			t.Fatalf("Reset() did not restore original order at index %d", i)
		}
	}
}
