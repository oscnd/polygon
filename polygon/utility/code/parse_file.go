package code

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func ParsePackageFile(pkg *Package, filePath string) (*File, error) {
	if pkg == nil {
		return nil, fmt.Errorf("package cannot be nil")
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Parse the Go file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	// Extract file name
	fileName := filepath.Base(filePath)

	// Create file struct
	file := &File{
		Package:    pkg,
		Name:       &fileName,
		Interfaces: []*Interface{},
		Structs:    []*Struct{},
		Receivers:  []*Receiver{},
		Functions:  []*Method{},
	}

	// Walk the AST to extract types and functions
	ast.Inspect(node, func(n ast.Node) bool {
		switch typedNode := n.(type) {
		case *ast.GenDecl:
			// Handle type declarations
			if typedNode.Tok == token.TYPE {
				for _, spec := range typedNode.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					// Check if it's an interface
					if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						iface := ParsePackageInterface(typeSpec, node)
						if iface != nil {
							file.Interfaces = append(file.Interfaces, iface)
						}
					}

					// Check if it's a struct
					if _, ok := typeSpec.Type.(*ast.StructType); ok {
						strct := ParsePackageStruct(typeSpec, node)
						if strct != nil {
							file.Structs = append(file.Structs, strct)
						}
					}
				}
			}

		case *ast.FuncDecl:
			// Handle function declarations
			if typedNode.Recv == nil {
				// Regular function
				fnc := ParsePackageFunction(typedNode, node)
				if fnc != nil {
					file.Functions = append(file.Functions, fnc)
				}
			} else {
				// Method with receiver
				receiver := ParsePackageReceiver(typedNode, node)
				if receiver != nil {
					file.Receivers = append(file.Receivers, receiver)
				}

				// Also add the method to functions
				method := ParsePackageFunction(typedNode, node)
				if method != nil {
					file.Functions = append(file.Functions, method)
				}
			}
		}

		return true
	})

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

				// Parse parameters
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
		Name:   &node.Name.Name,
		Fields: []*Field{},
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

	// Use the first name (interface methods can have multiple names)
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

	// Get receiver name (or "_" if unnamed)
	recvName := "_"
	if len(recv.Names) > 0 && recv.Names[0] != nil {
		recvName = recv.Names[0].Name
	}

	receiver := &Receiver{
		Name: &recvName,
	}

	// Find the struct that this receiver belongs to
	if file.Scope != nil {
		typeStr := ExprToString(recv.Type)
		if typeStr != "" {
			// Look for struct by type name (simplified approach)
			for _, obj := range file.Scope.Objects {
				if obj.Kind == ast.Typ && obj.Name == typeStr {
					// This is a simplified approach - in a real implementation,
					// you might want to maintain a mapping of types to structs
					break
				}
			}
		}
	}

	// Set the method
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

	// If there are multiple names for the same type
	if len(node.Names) > 0 {
		for _, name := range node.Names {
			param := &Parameter{
				Name: &name.Name,
				Type: &typeStr,
			}
			parameters = append(parameters, param)
		}
	} else {
		// Unnamed parameter (common in interface methods)
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

	// Parse struct tags
	var tags []*Tag
	if node.Tag != nil {
		tags = ParsePackageTags(node.Tag.Value)
	}

	// If there are multiple names for the same type
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
		// Embedded field
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

	// Remove backticks from tag string
	tagStr = strings.Trim(tagStr, "`")
	if tagStr == "" {
		return nil
	}

	var tags []*Tag

	// Split tags by space, but preserve quoted strings
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

		// Handle last character
		if i == len(tagStr)-1 && current.Len() > 0 {
			parts = append(parts, current.String())
		}
	}

	// Parse each tag part
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split tag name and value by colon
		colonIndex := strings.Index(part, ":")
		if colonIndex <= 0 {
			continue
		}

		name := strings.TrimSpace(part[:colonIndex])
		value := strings.TrimSpace(part[colonIndex:])

		// Remove quotes from value
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
