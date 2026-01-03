package code

import "go/ast"

type Interface struct {
	Name    *string       `json:"name"`
	Node    *ast.TypeSpec `json:"node"`
	Methods []*Function   `json:"methods"`
}

type Struct struct {
	Name      *string       `json:"name"`
	Node      *ast.TypeSpec `json:"node"`
	Fields    []*Field      `json:"fields"`
	Receivers []*Receiver   `json:"receivers"`
}

type Receiver struct {
	Name   *string   `json:"name"`
	Struct *Struct   `json:"struct"`
	Method *Function `json:"method"`
}
