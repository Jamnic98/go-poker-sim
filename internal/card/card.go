package card

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func CapitalizeFirst(s string) string {
	if len(s) == 0 {
		return ""
	}

	// Decode the first rune and get its byte size (handles multi-byte UTF-8 cleanly)
	r, size := utf8.DecodeRuneInString(s)

	return string(unicode.ToUpper(r)) + s[size:]
}

type Card struct {
	FaceValue string
	Suit      string
}

func (c *Card) String() string {
	faceValueStringMap := map[string]string{
		"2":  "two",
		"3":  "three",
		"4":  "four",
		"5":  "five",
		"6":  "six",
		"7":  "seven",
		"8":  "eight",
		"9":  "nine",
		"10": "ten",
		"J":  "jack",
		"Q":  "queen",
		"K":  "king",
		"A":  "ace",
	}

	valueString, ok := faceValueStringMap[c.FaceValue]
	if !ok {
		return ""
	}

	suitStringMap := map[string]string{
		"C": "clubs",
		"D": "diamonds",
		"H": "hearts",
		"S": "spades",
	}

	suitString, ok := suitStringMap[c.Suit]
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s of %s", CapitalizeFirst(valueString), CapitalizeFirst(suitString))
}
