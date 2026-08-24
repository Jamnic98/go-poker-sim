package deck

import "go-poker-sim/internal/card"

type Deck struct {
	Cards []card.Card
}

func New() *Deck {
	createCards := func() []card.Card {
		Suits := [4]string{"C", "D", "S", "H"}
		FaceValues := [13]string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}

		cards := make([]card.Card, 0, 52)
		for _, Suit := range Suits {
			for _, FaceValue := range FaceValues {
				cards = append(cards, card.Card{
					Suit:      Suit,
					FaceValue: FaceValue,
				})
			}
		}

		return cards
	}

	return &Deck{
		Cards: createCards(),
	}
}

func Shuffle() {

}
