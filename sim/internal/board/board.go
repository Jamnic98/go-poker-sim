package board

import (
	"errors"
	"fmt"

	"go-poker-sim/internal/card"
)

// Board holds the community cards. Cards must be added in order:
// flop (3 cards on an empty board), then turn, then river.
type Board struct {
	cards []card.Card
}

func New() *Board {
	return &Board{}
}

func (b *Board) Len() int {
	return len(b.cards)
}

// Cards returns a copy of the board cards.
func (b *Board) Cards() []card.Card {
	return append([]card.Card(nil), b.cards...)
}

func (b *Board) AddFlop(cards []card.Card) error {
	if len(cards) != 3 {
		return fmt.Errorf("flop requires exactly 3 cards, got %d", len(cards))
	}
	if len(b.cards) != 0 {
		return errors.New("flop can only be added to an empty board")
	}
	b.cards = append(b.cards, cards...)
	return nil
}

func (b *Board) AddTurn(c card.Card) error {
	if len(b.cards) != 3 {
		return fmt.Errorf("turn requires exactly 3 board cards, got %d", len(b.cards))
	}
	b.cards = append(b.cards, c)
	return nil
}

func (b *Board) AddRiver(c card.Card) error {
	if len(b.cards) != 4 {
		return fmt.Errorf("river requires exactly 4 board cards, got %d", len(b.cards))
	}
	b.cards = append(b.cards, c)
	return nil
}

func (b *Board) Reset() {
	b.cards = nil
}
