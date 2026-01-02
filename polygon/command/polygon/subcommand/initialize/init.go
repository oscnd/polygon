package initialize

import (
	"context"

	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/command/polygon/app"
	"go.scnd.dev/open/polygon/command/polygon/procedure/tree"
)

type Command struct {
	App   *app.App
	Force bool `help:"Force clean structure" short:"f"`
}

func (r *Command) Run(app *app.App) error {
	r.App = app
	return Run(r)
}

func Run(command *Command) error {
	ctx := context.Background()
	span, ctx := polygon.With(ctx)

	// * check tree
	tr, err := tree.New(ctx)
	if err != nil {
		return span.Error("invalid tree", err)
	}

	// * print tree summary
	_ = tr

	return nil
}
