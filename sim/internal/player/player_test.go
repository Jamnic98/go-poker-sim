package player

import (
	"testing"

	"go-poker-sim/internal/card"
)

func mustCard(t *testing.T, token string) card.Card {
	t.Helper()
	c, err := card.Parse(token)
	if err != nil {
		t.Fatalf("Parse(%q): %v", token, err)
	}
	return c
}

func TestNewPlayer(t *testing.T) {
	p := New(3)
	if p.ID != 3 {
		t.Errorf("player ID = %d, want 3", p.ID)
	}
	if len(p.Pocket()) != 0 {
		t.Error("new player should have an empty pocket")
	}
}

func TestAddCardsUpToTwo(t *testing.T) {
	p := New(0)
	if err := p.AddCards(mustCard(t, "AS")); err != nil {
		t.Fatalf("AddCards(AS) unexpected error: %v", err)
	}
	if err := p.AddCards(mustCard(t, "KD")); err != nil {
		t.Fatalf("AddCards(KD) unexpected error: %v", err)
	}
	pocket := p.Pocket()
	if len(pocket) != 2 {
		t.Fatalf("pocket len = %d, want 2", len(pocket))
	}
	if pocket[0].Token() != "AS" || pocket[1].Token() != "KD" {
		t.Errorf("pocket = [%s %s], want [AS KD]", pocket[0].Token(), pocket[1].Token())
	}
	if err := p.AddCards(mustCard(t, "QH")); err == nil {
		t.Error("AddCards with a third card expected error, got nil")
	}
	if len(p.Pocket()) != 2 {
		t.Error("rejected AddCards should not modify the pocket")
	}
}

func TestAddCardsRejectsOvershootInOneCall(t *testing.T) {
	p := New(1)
	if err := p.AddCards(mustCard(t, "AS"), mustCard(t, "KD"), mustCard(t, "QH")); err == nil {
		t.Error("AddCards with 3 cards on empty pocket expected error, got nil")
	}
	if len(p.Pocket()) != 0 {
		t.Error("rejected AddCards should leave the pocket empty")
	}
}

func TestPocketReturnsCopy(t *testing.T) {
	p := New(0)
	if err := p.AddCards(mustCard(t, "AS"), mustCard(t, "KD")); err != nil {
		t.Fatal(err)
	}
	pocket := p.Pocket()
	pocket[0] = card.Card{}
	if p.Pocket()[0] == (card.Card{}) {
		t.Error("Pocket() exposes the internal slice; mutating the copy changed the player")
	}
}

func TestReset(t *testing.T) {
	p := New(0)
	if err := p.AddCards(mustCard(t, "AS"), mustCard(t, "KD")); err != nil {
		t.Fatal(err)
	}
	p.Reset()
	if len(p.Pocket()) != 0 {
		t.Errorf("pocket len after Reset = %d, want 0", len(p.Pocket()))
	}
	if err := p.AddCards(mustCard(t, "2C"), mustCard(t, "3D")); err != nil {
		t.Errorf("player should accept new cards after Reset: %v", err)
	}
}
