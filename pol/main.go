package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"go.scnd.dev/polygon/pol/common/config"
	"go.scnd.dev/polygon/pol/index"
	"go.scnd.dev/polygon/pol/subcommand/database/sequel"
	"go.scnd.dev/polygon/pol/subcommand/inter"
)

type App struct {
	verbose   *bool
	directory *string
	config    *index.Config
}

func (a *App) Verbose() *bool {
	return a.verbose
}

func (a *App) Directory() *string {
	return a.directory
}

func (a *App) Config() *index.Config {
	return a.config
}

func main() {
	app := &App{
		verbose:   new(bool),
		directory: new(string),
		config:    nil,
	}

	// * parse directory flag
	flag.BoolVar(app.verbose, "v", false, "Verbose output")
	flag.StringVar(app.directory, "d", "", "Directory path")
	flag.Parse()

	// * check for remaining args after parsing flags
	args := flag.Args()
	if slices.Index(args, "-d") != -1 {
		// * remove -d and its value from args
		in := slices.Index(args, "-d")
		if in+1 < len(args) {
			args = append(args[:in], args[in+2:]...)
		} else {
			args = args[:in]
		}
	}

	if len(args) == 0 {
		if *app.Directory() == "" {
			fmt.Printf("Polygon Command Line Interface\n\n")
			fmt.Printf("Usage:\n")
			fmt.Printf("  %s -d <directory> <subcommand>\n\n", filepath.Base(os.Args[0]))
			fmt.Printf("Flags:\n")
			fmt.Printf("  -d: configuration directory\n")
			fmt.Printf("  -v: verbose output\n\n")
			fmt.Printf("Subcommands:\n")
			fmt.Printf("  database sequel schema\t generate database schema from migrations\n")
			fmt.Printf("  interface\t\t\t generate interfaces from receiver methods\n")
			return
		}
		fmt.Printf("Usage: %s -d <directory> <subcommand>\n", filepath.Base(os.Args[0]))
		return
	}

	// * load config
	var err error
	app.config, err = config.New[index.Config](*app.Directory())
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	subcommand := args[0]
	switch subcommand {
	case "database":
		if len(args) < 3 {
			println("Usage:", filepath.Base(os.Args[0]), "-d <directory> database sequel schema")
			return
		}
		if args[1] == "sequel" && args[2] == "schema" {
			err := sequel.Schema(app)
			if err != nil {
				log.Fatalf("error generating schemas: %v", err)
			}
		} else {
			println("unknown database subcommand:", args[1], args[2])
		}
	case "interface":
		err := inter.InterfaceGenerate(app)
		if err != nil {
			log.Fatalf("error generating interfaces: %v", err)
		}
	default:
		println("unknown subcommand:", subcommand)
	}
}
