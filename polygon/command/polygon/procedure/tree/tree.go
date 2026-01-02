package tree

import (
	"context"
	"log"
)

func New(ctx context.Context) (_ *Tree, err error) {
	tree := &Tree{
		Path:       "",
		Template:   nil,
		Handlers:   nil,
		Procedures: nil,
		Services:   nil,
	}

	tree.Procedures, err = ParseStructuredProcedure(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return tree, nil
}
