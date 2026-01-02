package initialize

import (
	"fmt"

	"go.scnd.dev/open/polygon/command/polygon/app"
)

type Command struct {
}

func (r *Command) Run(app *app.App) error {
	fmt.Println("ls")
	return nil
}
