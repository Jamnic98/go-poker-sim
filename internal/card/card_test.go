package card

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCard(t *testing.T) {
	Convey("Given a card", t, func() {
		Convey("It shoud print correctly", func() {
			testCard := Card{
				FaceValue: "2",
				Suit:      "H",
			}

			So(testCard.String(), ShouldEqual, "Two of Hearts")

			testCard.FaceValue = "A"
			So(testCard.String(), ShouldEqual, "Ace of Hearts")
		})
	})
}
