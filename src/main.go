package main

import (
	"fmt"

	"github.com/eriklarko/license-checker/src/boolexpr"
)

func main() {
	fmt.Println("Hello, World!")

	dt, err := boolexpr.New("T && F")
	fmt.Println(dt)
	fmt.Println(err)
}
