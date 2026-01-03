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

func ParsePackageFile(ctx context.Context, pkg *Package, filePath string) (*File, error) {
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

	// * create file struct
	file := &File{
		Package:    pkg,
		Name:       &fileName,
		Interfaces: []*Interface{},
		Structs:    []*Struct{},
		Receivers:  []*Receiver{},
		Functions:  []*Method{},
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
						iface := ParsePackageInterface(typeSpec, node)
						if iface != nil {
							file.Interfaces = append(file.Interfaces, iface)
						}
					}

					// * check if it's a struct
					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						strct := ParsePackageStruct(typeSpec, node)
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
				fnc := ParsePackageFunction(typedNode, node)
				if fnc != nil {
					file.Functions = append(file.Functions, fnc)
				}
			} else {
				// * method with receiver
				receiver := ParsePackageReceiver(typedNode, node)
				if receiver != nil {
					file.Receivers = append(file.Receivers, receiver)
					// * store receiver type for association
					recv := typedNode.Recv.List[0]
					recvType := ExprToString(recv.Type)
					receiverTypeMap[receiver] = recvType
				}

				// * also add the method to functions
				method := ParsePackageFunction(typedNode, node)
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

func ParsePackageInterface(node *ast.TypeSpec, file *ast.File) *Interface {
	if node.Name == nil {
		return nil
	}

	iface := &Interface{
		Name:    &node.Name.Name,
		Methods: []*Method{},
	}

	if interfaceType, ok := node.Type.(*ast.InterfaceType); ok && interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			if method.Names == nil || len(method.Names) == 0 {
				continue
			}

			for _, name := range method.Names {
				met := &Method{
					Name:       &name.Name,
					Parameters: []*Parameter{},
					Results:    []*Parameter{},
				}

				// * parse parameters
				if method.Type != nil {
					if funcType, ok := method.Type.(*ast.FuncType); ok {
						if funcType.Params != nil {
							for _, param := range funcType.Params.List {
								params := ParsePackageParameter(param, file)
								met.Parameters = append(met.Parameters, params...)
							}
						}

						if funcType.Results != nil {
							for _, result := range funcType.Results.List {
								results := ParsePackageParameter(result, file)
								met.Results = append(met.Results, results...)
							}
						}
					}
				}

				iface.Methods = append(iface.Methods, met)
			}
		}
	}

	return iface
}

func ParsePackageStruct(node *ast.TypeSpec, file *ast.File) *Struct {
	if node.Name == nil {
		return nil
	}

	strct := &Struct{
		Name:      &node.Name.Name,
		Fields:    []*Field{},
		Receivers: []*Receiver{},
	}

	if structType, ok := node.Type.(*ast.StructType); ok && structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fields := ParsePackageField(field, file)
			strct.Fields = append(strct.Fields, fields...)
		}
	}

	return strct
}

func ParsePackageMethod(node *ast.Field, file *ast.File) *Method {
	if node.Names == nil || len(node.Names) == 0 {
		return nil
	}

	// * use the first name
	name := node.Names[0].Name

	method := &Method{
		Name:       &name,
		Parameters: []*Parameter{},
		Results:    []*Parameter{},
	}

	if funcType, ok := node.Type.(*ast.FuncType); ok {
		if funcType.Params != nil {
			for _, param := range funcType.Params.List {
				params := ParsePackageParameter(param, file)
				method.Parameters = append(method.Parameters, params...)
			}
		}

		if funcType.Results != nil {
			for _, result := range funcType.Results.List {
				results := ParsePackageParameter(result, file)
				method.Results = append(method.Results, results...)
			}
		}
	}

	return method
}

func ParsePackageFunction(node *ast.FuncDecl, file *ast.File) *Method {
	if node.Name == nil {
		return nil
	}

	function := &Method{
		Name:       &node.Name.Name,
		Parameters: []*Parameter{},
		Results:    []*Parameter{},
	}

	if node.Type.Params != nil {
		for _, param := range node.Type.Params.List {
			params := ParsePackageParameter(param, file)
			function.Parameters = append(function.Parameters, params...)
		}
	}

	if node.Type.Results != nil {
		for _, result := range node.Type.Results.List {
			results := ParsePackageParameter(result, file)
			function.Results = append(function.Results, results...)
		}
	}

	return function
}

func ParsePackageReceiver(node *ast.FuncDecl, file *ast.File) *Receiver {
	if node.Recv == nil || len(node.Recv.List) == 0 {
		return nil
	}

	recv := node.Recv.List[0]

	// * get receiver name (or "_" if unnamed)
	recvName := "_"
	if len(recv.Names) > 0 && recv.Names[0] != nil {
		recvName = recv.Names[0].Name
	}

	receiver := &Receiver{
		Name: &recvName,
	}

	// * find the struct that this receiver belongs to
	if file.Scope != nil {
		typeStr := ExprToString(recv.Type)
		if typeStr != "" {
			// * look for struct by type name
			for _, obj := range file.Scope.Objects {
				if obj.Kind == ast.Typ && obj.Name == typeStr {
					break
				}
			}
		}
	}

	// * set the method
	if node.Name != nil {
		method := ParsePackageFunction(node, file)
		if method != nil {
			receiver.Method = method
		}
	}

	return receiver
}

func ParsePackageParameter(node *ast.Field, file *ast.File) []*Parameter {
	var parameters []*Parameter

	typeStr := ExprToString(node.Type)
	if typeStr == "" {
		return parameters
	}

	// * if there are multiple names for the same type
	if len(node.Names) > 0 {
		for _, name := range node.Names {
			param := &Parameter{
				Name: &name.Name,
				Type: &typeStr,
			}
			parameters = append(parameters, param)
		}
	} else {
		// * unnamed parameter (common in interface methods)
		param := &Parameter{
			Type: &typeStr,
		}
		parameters = append(parameters, param)
	}

	return parameters
}

func ParsePackageField(node *ast.Field, file *ast.File) []*Field {
	var fields []*Field

	typeStr := ExprToString(node.Type)
	if typeStr == "" {
		return fields
	}

	// * parse struct tags
	var tags []*Tag
	if node.Tag != nil {
		tags = ParsePackageTags(node.Tag.Value)
	}

	// * if there are multiple names for the same type
	if len(node.Names) > 0 {
		for _, name := range node.Names {
			field := &Field{
				Name: &name.Name,
				Type: &typeStr,
				Tags: tags,
			}
			fields = append(fields, field)
		}
	} else {
		// * embedded field
		field := &Field{
			Type: &typeStr,
			Tags: tags,
		}
		fields = append(fields, field)
	}

	return fields
}

func ParsePackageTags(tagStr string) []*Tag {
	if tagStr == "" {
		return nil
	}

	// * remove backticks from tag string
	tagStr = strings.Trim(tagStr, "`")
	if tagStr == "" {
		return nil
	}

	var tags []*Tag

	// * split tags by space, but preserve quoted strings
	var parts []string
	var current strings.Builder
	inQuotes := false

	for i, char := range tagStr {
		switch char {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(char)
		case ' ':
			if inQuotes {
				current.WriteRune(char)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}

		// * handle last character
		if i == len(tagStr)-1 && current.Len() > 0 {
			parts = append(parts, current.String())
		}
	}

	// * parse each tag part
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// * split tag name and value by colon
		colonIndex := strings.Index(part, ":")
		if colonIndex <= 0 {
			continue
		}

		name := strings.TrimSpace(part[:colonIndex])
		value := strings.TrimSpace(part[colonIndex:])

		// * remove quotes from value
		value = strings.Trim(value, `"`)
		value = strings.Trim(value, `'`)

		if name != "" {
			tag := &Tag{
				Name:  &name,
				Value: &value,
			}
			tags = append(tags, tag)
		}
	}

	return tags
}
