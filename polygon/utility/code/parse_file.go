package code

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon"
)

func ParseFile(ctx context.Context, pkg *Package, filePath string) (*File, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()
	s.Variable("filePath", filePath)

	if pkg == nil {
		return nil, s.Error("package cannot be nil", nil)
	}

	if filePath == "" {
		return nil, s.Error("file path cannot be empty", nil)
	}

	// * parse the Go file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, s.Error("failed to parse file", err)
	}

	// * extract file name
	fileName := filepath.Base(filePath)

	// * create import structure and extract imports
	fileImport := &Import{
		Imports: []*ImportItem{},
	}

	// * extract imports from AST
	for _, importSpec := range node.Imports {
		importPath := strings.Trim(importSpec.Path.Value, `"`)
		importItem := &ImportItem{
			Path: &importPath,
		}

		// * extract alias if present
		if importSpec.Name != nil {
			importItem.Alias = &importSpec.Name.Name
		}

		fileImport.Imports = append(fileImport.Imports, importItem)
	}

	// * create file struct
	file := &File{
		Package:    pkg,
		Name:       &fileName,
		Node:       node,
		Import:     fileImport,
		Interfaces: []*Interface{},
		Structs:    []*Struct{},
		Receivers:  []*Receiver{},
		Functions:  []*Function{},
	}

	// * walk the AST to extract types and functions
	receiverTypeMap := make(map[*Receiver]string)
	ast.Inspect(node, func(n ast.Node) bool {
		switch typedNode := n.(type) {
		case *ast.GenDecl:
			// * handle type declarations
			if typedNode.Tok == token.TYPE {
				for _, spec := range typedNode.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					// * check if it's an interface
					if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						iface := ParseFileInterface(ctx, typeSpec, node)
						if iface != nil {
							file.Interfaces = append(file.Interfaces, iface)
						}
					}

					// * check if it's a struct
					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						strct := ParseFileStruct(ctx, typeSpec, node)
						if strct != nil {
							file.Structs = append(file.Structs, strct)
						}
					}
				}
			}

		case *ast.FuncDecl:
			// * handle function declarations
			if typedNode.Recv == nil {
				// * regular function
				fnc := ParseFileFunction(ctx, typedNode, node)
				if fnc != nil {
					file.Functions = append(file.Functions, fnc)
				}
			} else {
				// * method with receiver
				receiver := ParseFileReceiver(ctx, typedNode, node)
				if receiver != nil {
					file.Receivers = append(file.Receivers, receiver)
					// * store receiver type for association
					recv := typedNode.Recv.List[0]
					recvType := ExprToString(recv.Type)
					receiverTypeMap[receiver] = recvType
				}

				// * also add the method to functions
				method := ParseFileFunction(ctx, typedNode, node)
				if method != nil {
					file.Functions = append(file.Functions, method)
				}
			}
		}

		return true
	})

	// * associate receivers with their structs
	for _, receiver := range file.Receivers {
		recvType, ok := receiverTypeMap[receiver]
		if !ok || recvType == "" {
			continue
		}
		// * find struct by receiver type name
		for _, strct := range file.Structs {
			if strct.Name == nil {
				continue
			}
			// * handle pointer receivers
			structName := *strct.Name
			if recvType == structName || recvType == "*"+structName {
				receiver.Struct = strct
				strct.Receivers = append(strct.Receivers, receiver)
				break
			}
		}
	}

	return file, nil
}
