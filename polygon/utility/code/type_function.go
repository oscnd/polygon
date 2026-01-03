package code

import "go/ast"

type Function struct {
	Name       *string       `json:"name"`
	Node       *ast.FuncDecl `json:"node"`
	Parameters []*Parameter  `json:"parameters"`
	Results    []*Parameter  `json:"results"`
	Annotates  []*Annotate   `json:"annotates"`
}

type Parameter struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}
