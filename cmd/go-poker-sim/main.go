package main

import (
	"fmt"
	"go-poker-sim/internal/deck"
)

func main() {
    d := deck.New()
	fmt.Println(d)
	d.Shuffle()
	fmt.Println(d)
}
