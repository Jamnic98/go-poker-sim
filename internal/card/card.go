package card

import "fmt"

type Suit int8

const (
	Clubs Suit = iota
	Diamonds
	Hearts
	Spades
)

var suitTokens = map[Suit]string{
	Clubs:    "C",
	Diamonds: "D",
	Hearts:   "H",
	Spades:   "S",
}

var suitNames = map[Suit]string{
	Clubs:    "Clubs",
	Diamonds: "Diamonds",
	Hearts:   "Hearts",
	Spades:   "Spades",
}

var suitByToken = map[string]Suit{
	"C": Clubs,
	"D": Diamonds,
	"H": Hearts,
	"S": Spades,
}

func (s Suit) Token() string {
	if token, ok := suitTokens[s]; ok {
		return token
	}
	return "?"
}

func (s Suit) Name() string {
	if name, ok := suitNames[s]; ok {
		return name
	}
	return "Unknown"
}

func (s Suit) Valid() bool {
	return s >= Clubs && s <= Spades
}

var faceTokens = map[int8]string{
	2:  "2",
	3:  "3",
	4:  "4",
	5:  "5",
	6:  "6",
	7:  "7",
	8:  "8",
	9:  "9",
	10: "10",
	11: "J",
	12: "Q",
	13: "K",
	14: "A",
}

var faceNames = map[int8]string{
	2:  "Two",
	3:  "Three",
	4:  "Four",
	5:  "Five",
	6:  "Six",
	7:  "Seven",
	8:  "Eight",
	9:  "Nine",
	10: "Ten",
	11: "Jack",
	12: "Queen",
	13: "King",
	14: "Ace",
}

var rankByFaceToken = map[string]int8{
	"2":  2,
	"3":  3,
	"4":  4,
	"5":  5,
	"6":  6,
	"7":  7,
	"8":  8,
	"9":  9,
	"10": 10,
	"J":  11,
	"Q":  12,
	"K":  13,
	"A":  14,
}

const (
	MinRank int8 = 2
	MaxRank int8 = 14 // Ace
)

// Card is a single playing card. Rank is 2-14 (11=J, 12=Q, 13=K, 14=A).
type Card struct {
	Rank int8
	Suit Suit
}

func New(rank int8, suit Suit) (Card, error) {
	if rank < MinRank || rank > MaxRank {
		return Card{}, fmt.Errorf("invalid rank %d: must be between %d and %d", rank, MinRank, MaxRank)
	}
	if !suit.Valid() {
		return Card{}, fmt.Errorf("invalid suit %d", int8(suit))
	}
	return Card{Rank: rank, Suit: suit}, nil
}

// Parse parses a token such as "10H" or "AS" into a Card.
func Parse(token string) (Card, error) {
	if len(token) < 2 {
		return Card{}, fmt.Errorf("invalid card token %q: too short", token)
	}
	suit, ok := suitByToken[token[len(token)-1:]]
	if !ok {
		return Card{}, fmt.Errorf("invalid card token %q: unknown suit", token)
	}
	rank, ok := rankByFaceToken[token[:len(token)-1]]
	if !ok {
		return Card{}, fmt.Errorf("invalid card token %q: unknown face", token)
	}
	return Card{Rank: rank, Suit: suit}, nil
}

// Token returns the compact representation, e.g. "10H".
func (c Card) Token() string {
	return faceTokens[c.Rank] + c.Suit.Token()
}

// String returns the long representation, e.g. "Ten of Hearts".
func (c Card) String() string {
	return fmt.Sprintf("%s of %s", faceNames[c.Rank], c.Suit.Name())
}
