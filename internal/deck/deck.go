package deck

import (
	"errors"
	"math/rand/v2"

	"go-poker-sim/internal/card"
)

// Deck is a standard 52-card deck. Cards are dealt from the end via Pop.
type Deck struct {
	Cards []card.Card
}

func New() *Deck {
	cards := make([]card.Card, 0, 52)
	for s := card.Clubs; s <= card.Spades; s++ {
		for r := card.MinRank; r <= card.MaxRank; r++ {
			cards = append(cards, card.Card{Rank: r, Suit: s})
		}
	}
	return &Deck{Cards: cards}
}

func (d *Deck) Len() int {
	return len(d.Cards)
}

// Shuffle performs a Fisher-Yates shuffle using the provided RNG.
// Each goroutine should use its own *rand.Rand to avoid lock contention.
func (d *Deck) Shuffle(r *rand.Rand) {
	r.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Pop removes and returns the top card of the deck.
func (d *Deck) Pop() (card.Card, error) {
	if len(d.Cards) == 0 {
		return card.Card{}, errors.New("cannot pop from an empty deck")
	}
	c := d.Cards[len(d.Cards)-1]
	d.Cards = d.Cards[:len(d.Cards)-1]
	return c, nil
}

func (d *Deck) Reset() {
	d.Cards = New().Cards
}
