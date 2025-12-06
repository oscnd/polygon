package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go.scnd.dev/polygon/pol/command/database/sequel"
	"go.scnd.dev/polygon/pol/common/config"
	"go.scnd.dev/polygon/pol/index"
)

type App struct {
	directory *string
	config    *index.Config
}

func (a *App) Directory() *string {
	return a.directory
}

func (a *App) Config() *index.Config {
	return a.config
}

func main() {
	app := &App{
		directory: new(string),
		config:    nil,
	}

	// * parse directory flag
	flag.StringVar(app.directory, "d", "", "Directory path")
	flag.Parse()

	// * check for remaining args after parsing flags
	args := flag.Args()
	if len(args) < 2 {
		if *app.directory == "" {
			fmt.Printf("Polygon Core\n\n")
			fmt.Printf("Usage: %s -d <directory> <subcommand>\n", os.Args[0])
			fmt.Printf("  -d: Directory path (e.g., polygon)\n")
			fmt.Printf("Available subcommands:\n")
			fmt.Printf("  database sequel schema - Generate database schema from migrations\n")
			return
		}
		fmt.Printf("Usage: %s -d <directory> <subcommand>\n", os.Args[0])
		return
	}

	// * load config
	var err error
	app.config, err = config.New[index.Config](*app.directory)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	subcommand := args[1]
	switch subcommand {
	case "database":
		if len(args) < 4 {
			println("Usage:", os.Args[0], "-d <directory> database sequel schema")
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
