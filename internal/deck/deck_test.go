package deck

import (
	"go-poker-sim/internal/card"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDeckSpec(t *testing.T) {
	Convey("Given a new card deck", t, func() {
		deck := New()

		Convey("It should contain exactly 52 cards", func() {
			So(len(deck.Cards), ShouldEqual, 52)
		})

		Convey("It should contain the 10 of Hearts", func() {
			testCard := card.Card{FaceValue: "10", Suit: "H"}
			So(deck.Cards, ShouldContain, testCard)
		})

		Convey("It should contain the A of Spades", func() {
			testCard := card.Card{FaceValue: "A", Suit: "S"}
			So(deck.Cards, ShouldContain, testCard)
		})
	})
}
