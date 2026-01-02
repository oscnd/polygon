package main

import (
	"github.com/alecthomas/kong"
	"go.scnd.dev/open/polygon/command/polygon/app"
	"go.scnd.dev/open/polygon/command/polygon/subcommand/initialize"
)

type Command struct {
	Verbose bool                `help:"Enable verbose output." short:"v"`
	Init    *initialize.Command `cmd:"init" help:"Initialize a new Polygon project."`
}

func main() {
	command := new(Command)
	ctx := kong.Parse(
		command,
		kong.Name("polygon"),
		kong.Description("Polygon Command Line Interface"),
	)
	err := ctx.Run(&app.App{
		Verbose: command.Verbose,
	})
	ctx.FatalIfErrorf(err)
}
