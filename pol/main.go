package main

import (
	"fmt"
	"os"
)

func main() {
	// * parse args
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Polygon Core\n\n")
		fmt.Printf("Usage: %s <subcommand>\n", args[0])
		return
	}

	subcommand := args[1]

	switch subcommand {
	default:
		println("Unknown subcommand:", subcommand)
	}
}
