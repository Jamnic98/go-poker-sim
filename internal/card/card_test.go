package card

import "testing"

func TestParseValidTokens(t *testing.T) {
	cases := []struct {
		token string
		rank  int8
		suit  Suit
	}{
		{"2C", 2, Clubs},
		{"9D", 9, Diamonds},
		{"10H", 10, Hearts},
		{"JS", 11, Spades},
		{"QD", 12, Diamonds},
		{"KH", 13, Hearts},
		{"AC", 14, Clubs},
	}
	for _, tc := range cases {
		c, err := Parse(tc.token)
		if err != nil {
			t.Fatalf("Parse(%q) returned error: %v", tc.token, err)
		}
		if c.Rank != tc.rank || c.Suit != tc.suit {
			t.Errorf("Parse(%q) = {Rank:%d Suit:%d}, want {Rank:%d Suit:%d}", tc.token, c.Rank, c.Suit, tc.rank, tc.suit)
		}
		if got := c.Token(); got != tc.token {
			t.Errorf("Card.Token() = %q, want round-trip %q", got, tc.token)
		}
	}
}

func TestParseInvalidTokens(t *testing.T) {
	for _, token := range []string{"", "A", "10", "1H", "0C", "XH", "10X", "ASX", "11H", "A1", "10HH"} {
		if _, err := Parse(token); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", token)
		}
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(1, Clubs); err == nil {
		t.Error("New(1, Clubs) expected error, got nil")
	}
	if _, err := New(15, Clubs); err == nil {
		t.Error("New(15, Clubs) expected error, got nil")
	}
	if _, err := New(2, Suit(42)); err == nil {
		t.Error("New(2, Suit(42)) expected error, got nil")
	}
	c, err := New(14, Spades)
	if err != nil {
		t.Fatalf("New(14, Spades) unexpected error: %v", err)
	}
	if c.Token() != "AS" {
		t.Errorf("New(14, Spades).Token() = %q, want %q", c.Token(), "AS")
	}
}

func TestCardString(t *testing.T) {
	cases := []struct {
		c    Card
		want string
	}{
		{Card{Rank: 2, Suit: Clubs}, "Two of Clubs"},
		{Card{Rank: 10, Suit: Hearts}, "Ten of Hearts"},
		{Card{Rank: 11, Suit: Spades}, "Jack of Spades"},
		{Card{Rank: 12, Suit: Diamonds}, "Queen of Diamonds"},
		{Card{Rank: 13, Suit: Hearts}, "King of Hearts"},
		{Card{Rank: 14, Suit: Spades}, "Ace of Spades"},
	}
	for _, tc := range cases {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("Card%+v.String() = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestSuitTokenNameValid(t *testing.T) {
	suits := []Suit{Clubs, Diamonds, Hearts, Spades}
	tokens := []string{"C", "D", "H", "S"}
	for i, s := range suits {
		if s.Token() != tokens[i] {
			t.Errorf("Suit(%d).Token() = %q, want %q", int8(s), s.Token(), tokens[i])
		}
		if !s.Valid() {
			t.Errorf("Suit(%d).Valid() = false, want true", int8(s))
		}
	}
	if Suit(9).Valid() {
		t.Error("Suit(9).Valid() = true, want false")
	}
	if got := Spades.Name(); got != "Spades" {
		t.Errorf("Spades.Name() = %q, want %q", got, "Spades")
	}
}

func TestCardComparable(t *testing.T) {
	a := Card{Rank: 14, Suit: Hearts}
	b := Card{Rank: 14, Suit: Hearts}
	c := Card{Rank: 14, Suit: Spades}
	if a != b {
		t.Error("identical cards should be equal")
	}
	if a == c {
		t.Error("same rank different suit should not be equal")
	}
	set := map[Card]bool{a: true}
	if !set[b] {
		t.Error("equal cards should share a map key")
	}
}
