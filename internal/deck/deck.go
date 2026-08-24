package deck

import (
	"go-poker-sim/internal/card"
	"math/rand"
)

type Deck struct {
	Cards []card.Card
}

func New() *Deck {
	createCards := func() []card.Card {
		Suits := [4]string{"C", "D", "S", "H"}
		FaceValues := [13]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}

		cards := make([]card.Card, 0, 52)
		for _, s := range Suits {
			for _, fV := range FaceValues {
				cards = append(cards, card.Card{
					Suit:      s,
					FaceValue: fV,
				})
			}
		}

		return cards
	}

	return &Deck{
		Cards: createCards(),
	}
}

func (d *Deck) Shuffle() {
	cards := d.Cards
    for i := range cards {
        j := rand.Intn(i + 1)
	    cards[i], cards[j] = cards[j], cards[i]
	}
}
