package player

import (
	"fmt"

	"go-poker-sim/internal/card"
)

const MaxHoleCards = 2

// Player is a seat at the table holding up to two hole cards.
type Player struct {
	ID     int
	pocket []card.Card
}

func New(id int) *Player {
	return &Player{ID: id}
}

// Pocket returns a copy of the player's hole cards.
func (p *Player) Pocket() []card.Card {
	return append([]card.Card(nil), p.pocket...)
}

func (p *Player) AddCards(cards ...card.Card) error {
	if len(p.pocket)+len(cards) > MaxHoleCards {
		return fmt.Errorf("player %d cannot hold more than %d hole cards (has %d, adding %d)",
			p.ID, MaxHoleCards, len(p.pocket), len(cards))
	}
	p.pocket = append(p.pocket, cards...)
	return nil
}

func (p *Player) Reset() {
	p.pocket = nil
}
