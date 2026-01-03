package code

import "go/ast"

type File struct {
	Package    *Package     `json:"package"`
	Name       *string      `json:"name"`
	Node       *ast.File    `json:"node"`
	Import     *Import      `json:"import"`
	Interfaces []*Interface `json:"interfaces"`
	Structs    []*Struct    `json:"structs"`
	Receivers  []*Receiver  `json:"receivers"`
	Functions  []*Function  `json:"functions"`
}
