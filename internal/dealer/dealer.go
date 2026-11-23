package dealer

import (
	"fmt"
	"math/rand/v2"

	"go-poker-sim/internal/board"
	"go-poker-sim/internal/card"
	"go-poker-sim/internal/deck"
	"go-poker-sim/internal/player"
)

// Dealer shuffles a deck and deals cards to players and the board.
type Dealer struct {
	deck *deck.Deck
}

func New(d *deck.Deck) *Dealer {
	return &Dealer{deck: d}
}

func (d *Dealer) Deck() *deck.Deck {
	return d.deck
}

func (d *Dealer) Shuffle(r *rand.Rand) {
	d.deck.Shuffle(r)
}

// DealHoleCards deals two hole cards to each player, one card per player
// per round (round-robin, starting from the first player).
func (d *Dealer) DealHoleCards(players []*player.Player) error {
	for round := 0; round < player.MaxHoleCards; round++ {
		for _, p := range players {
			c, err := d.deck.Pop()
			if err != nil {
				return fmt.Errorf("dealing hole cards to player %d: %w", p.ID, err)
			}
			if err := p.AddCards(c); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Dealer) DealFlop(b *board.Board) error {
	flop := make([]card.Card, 0, 3)
	for i := 0; i < 3; i++ {
		c, err := d.deck.Pop()
		if err != nil {
			return fmt.Errorf("dealing flop: %w", err)
		}
		flop = append(flop, c)
	}
	return b.AddFlop(flop)
}

func (d *Dealer) DealTurn(b *board.Board) error {
	c, err := d.deck.Pop()
	if err != nil {
		return fmt.Errorf("dealing turn: %w", err)
	}
	return b.AddTurn(c)
}

func (d *Dealer) DealRiver(b *board.Board) error {
	c, err := d.deck.Pop()
	if err != nil {
		return fmt.Errorf("dealing river: %w", err)
	}
	return b.AddRiver(c)
}

func (d *Dealer) Reset() {
	d.deck.Reset()
}
