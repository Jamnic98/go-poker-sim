package board

import (
	"testing"

	"go-poker-sim/internal/card"
)

func mustParse(t *testing.T, tokens ...string) []card.Card {
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

func TestNewBoardIsEmpty(t *testing.T) {
	b := New()
	if b.Len() != 0 {
		t.Errorf("new board len = %d, want 0", b.Len())
	}
	if len(b.Cards()) != 0 {
		t.Error("new board Cards() should be empty")
	}
}

func TestAddFlop(t *testing.T) {
	b := New()
	flop := mustParse(t, "AC", "10H", "3S")
	if err := b.AddFlop(flop); err != nil {
		t.Fatalf("AddFlop() unexpected error: %v", err)
	}
	if b.Len() != 3 {
		t.Errorf("board len after flop = %d, want 3", b.Len())
	}
	for i, c := range b.Cards() {
		if c != flop[i] {
			t.Errorf("board card %d = %s, want %s", i, c.Token(), flop[i].Token())
		}
	}
}

func TestAddFlopRejectsWrongCount(t *testing.T) {
	b := New()
	if err := b.AddFlop(mustParse(t, "AC", "10H")); err == nil {
		t.Error("AddFlop with 2 cards expected error, got nil")
	}
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S", "4H")); err == nil {
		t.Error("AddFlop with 4 cards expected error, got nil")
	}
	if b.Len() != 0 {
		t.Error("failed AddFlop should not modify the board")
	}
}

func TestAddFlopRejectsNonEmptyBoard(t *testing.T) {
	b := New()
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S")); err != nil {
		t.Fatal(err)
	}
	if err := b.AddFlop(mustParse(t, "4H", "5H", "6H")); err == nil {
		t.Error("AddFlop on non-empty board expected error, got nil")
	}
	if b.Len() != 3 {
		t.Errorf("board len = %d, want 3 after rejected flop", b.Len())
	}
}

func TestAddTurn(t *testing.T) {
	b := New()
	if err := b.AddTurn(mustParse(t, "4H")[0]); err == nil {
		t.Error("AddTurn on empty board expected error, got nil")
	}
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S")); err != nil {
		t.Fatal(err)
	}
	turn := mustParse(t, "4H")[0]
	if err := b.AddTurn(turn); err != nil {
		t.Fatalf("AddTurn() unexpected error: %v", err)
	}
	if b.Len() != 4 {
		t.Errorf("board len after turn = %d, want 4", b.Len())
	}
	if b.Cards()[3] != turn {
		t.Errorf("turn card = %s, want %s", b.Cards()[3].Token(), turn.Token())
	}
	if err := b.AddTurn(mustParse(t, "5H")[0]); err == nil {
		t.Error("second AddTurn expected error, got nil")
	}
}

func TestAddRiver(t *testing.T) {
	b := New()
	if err := b.AddRiver(mustParse(t, "2C")[0]); err == nil {
		t.Error("AddRiver on empty board expected error, got nil")
	}
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S")); err != nil {
		t.Fatal(err)
	}
	if err := b.AddRiver(mustParse(t, "2C")[0]); err == nil {
		t.Error("AddRiver with only 3 board cards expected error, got nil")
	}
	if err := b.AddTurn(mustParse(t, "4H")[0]); err != nil {
		t.Fatal(err)
	}
	river := mustParse(t, "2C")[0]
	if err := b.AddRiver(river); err != nil {
		t.Fatalf("AddRiver() unexpected error: %v", err)
	}
	if b.Len() != 5 {
		t.Errorf("board len after river = %d, want 5", b.Len())
	}
	if b.Cards()[4] != river {
		t.Errorf("river card = %s, want %s", b.Cards()[4].Token(), river.Token())
	}
}

func TestFullBoardSequenceKeepsOrder(t *testing.T) {
	b := New()
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S")); err != nil {
		t.Fatal(err)
	}
	if err := b.AddTurn(mustParse(t, "4H")[0]); err != nil {
		t.Fatal(err)
	}
	if err := b.AddRiver(mustParse(t, "2C")[0]); err != nil {
		t.Fatal(err)
	}
	want := []string{"AC", "10H", "3S", "4H", "2C"}
	got := b.Cards()
	if len(got) != len(want) {
		t.Fatalf("board len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Token() != want[i] {
			t.Errorf("board card %d = %s, want %s", i, got[i].Token(), want[i])
		}
	}
}

func TestCardsReturnsCopy(t *testing.T) {
	b := New()
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S")); err != nil {
		t.Fatal(err)
	}
	cs := b.Cards()
	cs[0] = card.Card{}
	if b.Cards()[0] == (card.Card{}) {
		t.Error("Cards() exposes the internal slice; mutating the copy changed the board")
	}
}

func TestReset(t *testing.T) {
	b := New()
	if err := b.AddFlop(mustParse(t, "AC", "10H", "3S")); err != nil {
		t.Fatal(err)
	}
	b.Reset()
	if b.Len() != 0 {
		t.Errorf("board len after Reset = %d, want 0", b.Len())
	}
	if err := b.AddFlop(mustParse(t, "5D", "6D", "7D")); err != nil {
		t.Errorf("board should accept a flop after Reset: %v", err)
	}
}
