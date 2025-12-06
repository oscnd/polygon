package main

import (
	"fmt"
	"log"
	"os"

	"go.scnd.dev/polygon/pol/command/database/sequel"
)

func main() {
	// * parse args
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Polygon Core\n\n")
		fmt.Printf("Usage: %s <subcommand>\n", args[0])
		fmt.Printf("Available subcommands:\n")
		fmt.Printf("  database sequel schema - Generate database schema from migrations\n")
		return
	}

	subcommand := args[1]

	switch subcommand {
	case "database":
		if len(args) < 4 {
			println("Usage:", args[0], "database sequel schema")
			return
		}
		if args[2] == "sequel" && args[3] == "schema" {
			err := sequel.Schema()
			if err != nil {
				log.Fatalf("Error generating schemas: %v", err)
			}
		} else {
			println("Unknown database subcommand:", args[2], args[3])
		}
	default:
		println("Unknown subcommand:", subcommand)
	}
}
