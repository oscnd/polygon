package printer

import (
	"fmt"

	"github.com/ddddddO/gtree"
	"go.scnd.dev/open/polygon/command/polygon/procedure/tree"
	"go.scnd.dev/open/polygon/utility/code"
)

func PrintTree(module *code.Module, tr *tree.Tree) string {
	root := gtree.NewRoot(*module.Name)
	for _, proc := range tr.Procedures {
		node := root.Add(proc.Name)
	}
}
