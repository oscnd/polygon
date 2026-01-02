package initialize

import (
	"context"

	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/command/polygon/app"
	"go.scnd.dev/open/polygon/command/polygon/procedure/tree"
)

type Command struct {
	Force bool `help:"Force clean structure" short:"v"`
}

func (r *Command) Run(app *app.App) error {
	return Run(app, r)
}

func Run(app *app.App, command *Command) error {
	ctx := context.Background()
	span, ctx := polygon.With(ctx)

	// * check tree
	tree, err := tree.New(ctx)
	if err != nil {
		return span.Error("invalid tree", err)
	}

	_ = tree

	return nil
}
