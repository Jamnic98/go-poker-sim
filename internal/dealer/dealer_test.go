package dealer

import (
	"math/rand/v2"
	"testing"

	"go-poker-sim/internal/board"
	"go-poker-sim/internal/card"
	"go-poker-sim/internal/deck"
	"go-poker-sim/internal/player"
)

func newPlayers(n int) []*player.Player {
	players := make([]*player.Player, n)
	for i := range players {
		players[i] = player.New(i)
	}
	return players
}

func pocketTokens(p *player.Player) []string {
	pocket := p.Pocket()
	tokens := make([]string, len(pocket))
	for i, c := range pocket {
		tokens[i] = c.Token()
	}
	return tokens
}

// An unshuffled deck pops from the end: AS, KS, QS, JS, 10S, 9S, ...
func TestDealHoleCardsRoundRobinOrder(t *testing.T) {
	d := New(deck.New())
	players := newPlayers(3)
	if err := d.DealHoleCards(players); err != nil {
		t.Fatalf("DealHoleCards() unexpected error: %v", err)
	}
	// round 0: p0=AS p1=KS p2=QS; round 1: p0=JS p1=10S p2=9S
	want := [][]string{{"AS", "JS"}, {"KS", "10S"}, {"QS", "9S"}}
	for i, p := range players {
		got := pocketTokens(p)
		if len(got) != 2 || got[0] != want[i][0] || got[1] != want[i][1] {
			t.Errorf("player %d pocket = %v, want %v", i, got, want[i])
		}
	}
	if d.Deck().Len() != 52-6 {
		t.Errorf("deck len after hole cards = %d, want %d", d.Deck().Len(), 52-6)
	}
}

func TestDealHoleCardsAllUniqueAcrossPlayers(t *testing.T) {
	d := New(deck.New())
	d.Shuffle(rand.New(rand.NewPCG(5, 6)))
	players := newPlayers(10)
	if err := d.DealHoleCards(players); err != nil {
		t.Fatalf("DealHoleCards() unexpected error: %v", err)
	}
	seen := make(map[card.Card]bool)
	for _, p := range players {
		pocket := p.Pocket()
		if len(pocket) != 2 {
			t.Errorf("player %d has %d hole cards, want 2", p.ID, len(pocket))
		}
		for _, c := range pocket {
			if seen[c] {
				t.Errorf("duplicate hole card %s dealt", c.Token())
			}
			seen[c] = true
		}
	}
	if len(seen) != 20 {
		t.Errorf("dealt %d unique hole cards, want 20", len(seen))
	}
	if d.Deck().Len() != 32 {
		t.Errorf("deck len = %d, want 32", d.Deck().Len())
	}
}

func TestDealFlopFromUnshuffledDeck(t *testing.T) {
	d := New(deck.New())
	b := board.New()
	if err := d.DealFlop(b); err != nil {
		t.Fatalf("DealFlop() unexpected error: %v", err)
	}
	want := []string{"AS", "KS", "QS"}
	got := b.Cards()
	if len(got) != 3 {
		t.Fatalf("board len = %d, want 3", len(got))
	}
	for i := range want {
		if got[i].Token() != want[i] {
			t.Errorf("flop card %d = %s, want %s", i, got[i].Token(), want[i])
		}
	}
	if d.Deck().Len() != 49 {
		t.Errorf("deck len after flop = %d, want 49", d.Deck().Len())
	}
}

func TestDealFullBoardAllCardsUnique(t *testing.T) {
	d := New(deck.New())
	d.Shuffle(rand.New(rand.NewPCG(11, 12)))
	players := newPlayers(10)
	if err := d.DealHoleCards(players); err != nil {
		t.Fatal(err)
	}
	b := board.New()
	if err := d.DealFlop(b); err != nil {
		t.Fatal(err)
	}
	if err := d.DealTurn(b); err != nil {
		t.Fatal(err)
	}
	if err := d.DealRiver(b); err != nil {
		t.Fatal(err)
	}
	if b.Len() != 5 {
		t.Errorf("board len = %d, want 5", b.Len())
	}
	seen := make(map[card.Card]bool)
	for _, p := range players {
		for _, c := range p.Pocket() {
			if seen[c] {
				t.Errorf("duplicate card %s across players and board", c.Token())
			}
			seen[c] = true
		}
	}
	for _, c := range b.Cards() {
		if seen[c] {
			t.Errorf("duplicate card %s across players and board", c.Token())
		}
		seen[c] = true
	}
	if len(seen) != 25 {
		t.Errorf("dealt %d unique cards total, want 25", len(seen))
	}
	if d.Deck().Len() != 52-25 {
		t.Errorf("deck len = %d, want %d", d.Deck().Len(), 52-25)
	}
}

func TestDealErrorsWhenDeckEmpty(t *testing.T) {
	d := New(deck.New())
	for d.Deck().Len() > 0 {
		if _, err := d.Deck().Pop(); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.DealHoleCards(newPlayers(2)); err == nil {
		t.Error("DealHoleCards on empty deck expected error, got nil")
	}
	b := board.New()
	if err := d.DealFlop(b); err == nil {
		t.Error("DealFlop on empty deck expected error, got nil")
	}
	if err := b.AddFlop([]card.Card{{Rank: 2, Suit: card.Clubs}, {Rank: 3, Suit: card.Clubs}, {Rank: 4, Suit: card.Clubs}}); err != nil {
		t.Fatal(err)
	}
	if err := d.DealTurn(b); err == nil {
		t.Error("DealTurn on empty deck expected error, got nil")
	}
	if err := d.DealRiver(b); err == nil {
		t.Error("DealRiver with 3 board cards on empty deck expected error, got nil")
	}
}

func TestShuffleChangesDeckOrder(t *testing.T) {
	fresh := deck.New()
	d := New(deck.New())
	d.Shuffle(rand.New(rand.NewPCG(3, 4)))
	changed := false
	for i := range d.Deck().Cards {
		if d.Deck().Cards[i] != fresh.Cards[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("Shuffle did not change the deck order")
	}
}

func TestReset(t *testing.T) {
	d := New(deck.New())
	if err := d.DealHoleCards(newPlayers(4)); err != nil {
		t.Fatal(err)
	}
	d.Reset()
	if d.Deck().Len() != 52 {
		t.Errorf("deck len after Reset = %d, want 52", d.Deck().Len())
	}
}
