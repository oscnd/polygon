package code

import (
	"context"
	"go/ast"
	"strings"

	"go.scnd.dev/open/polygon"
)

func ParseFileInterface(ctx context.Context, node *ast.TypeSpec, file *ast.File, doc *ast.CommentGroup) *Interface {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	if node.Name == nil {
		return nil
	}

	iface := &Interface{
		Name:      &node.Name.Name,
		Node:      node,
		Methods:   []*Function{},
		Annotates: ParseFileAnnotations(doc),
	}

	if interfaceType, ok := node.Type.(*ast.InterfaceType); ok && interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			if method.Names == nil || len(method.Names) == 0 {
				continue
			}

			for _, name := range method.Names {
				met := &Function{
					Name:       &name.Name,
					Parameters: []*Parameter{},
					Results:    []*Parameter{},
				}

				// * parse parameters
				if method.Type != nil {
					if funcType, ok := method.Type.(*ast.FuncType); ok {
						if funcType.Params != nil {
							for _, param := range funcType.Params.List {
								params := ParseFileParameter(ctx, param, file)
								met.Parameters = append(met.Parameters, params...)
							}
						}

						if funcType.Results != nil {
							for _, result := range funcType.Results.List {
								results := ParseFileParameter(ctx, result, file)
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

func ParseFileStruct(ctx context.Context, node *ast.TypeSpec, file *ast.File, doc *ast.CommentGroup) *Struct {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	if node.Name == nil {
		return nil
	}

	strct := &Struct{
		Name:      &node.Name.Name,
		Node:      node,
		Fields:    []*Field{},
		Receivers: []*Receiver{},
		Annotates: ParseFileAnnotations(doc),
	}

	if structType, ok := node.Type.(*ast.StructType); ok && structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fields := ParseFileField(ctx, field, file)
			strct.Fields = append(strct.Fields, fields...)
		}
	}

	return strct
}

func ParseFileFunction(ctx context.Context, node *ast.FuncDecl, file *ast.File) *Function {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	if node.Name == nil {
		return nil
	}

	function := &Function{
		Name:       &node.Name.Name,
		Node:       node,
		Parameters: []*Parameter{},
		Results:    []*Parameter{},
		Annotates:  ParseFileAnnotations(node.Doc),
	}

	if node.Type.Params != nil {
		for _, param := range node.Type.Params.List {
			params := ParseFileParameter(ctx, param, file)
			function.Parameters = append(function.Parameters, params...)
		}
	}

	if node.Type.Results != nil {
		for _, result := range node.Type.Results.List {
			results := ParseFileParameter(ctx, result, file)
			function.Results = append(function.Results, results...)
		}
	}

	return function
}

func ParseFileReceiver(ctx context.Context, node *ast.FuncDecl, file *ast.File) *Receiver {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	if node.Recv == nil || len(node.Recv.List) == 0 {
		return nil
	}

	recv := node.Recv.List[0]

	// * get receiver name (or "_" if unnamed)
	recvName := "_"
	if len(recv.Names) > 0 && recv.Names[0] != nil {
		recvName = recv.Names[0].Name
	}

	annotates := ParseFileAnnotations(node.Doc)

	receiver := &Receiver{
		Name:      &recvName,
		Annotates: annotates,
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
		method := ParseFileFunction(ctx, node, file)
		if method != nil {
			receiver.Method = method
		}
	}

	return receiver
}

func ParseFileParameter(ctx context.Context, node *ast.Field, file *ast.File) []*Parameter {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

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

func ParseFileField(ctx context.Context, node *ast.Field, file *ast.File) []*Field {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	var fields []*Field

	typeStr := ExprToString(node.Type)
	if typeStr == "" {
		return fields
	}

	// * parse struct tags
	var tags []*Tag
	if node.Tag != nil {
		tags = ParseFileTags(node.Tag.Value)
	}

	// * parse annotations
	annotates := ParseFileAnnotations(node.Doc)

	// * if there are multiple names for the same type
	if len(node.Names) > 0 {
		for _, name := range node.Names {
			field := &Field{
				Name:      &name.Name,
				Type:      &typeStr,
				Tags:      tags,
				Annotates: annotates,
			}
			fields = append(fields, field)
		}
	} else {
		// * embedded field
		field := &Field{
			Type:      &typeStr,
			Tags:      tags,
			Annotates: annotates,
		}
		fields = append(fields, field)
	}

	return fields
}

func ParseFileTags(tagStr string) []*Tag {
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

func ParseFileAnnotations(commentGroup *ast.CommentGroup) []*Annotate {
	var annotates []*Annotate

	if commentGroup == nil {
		return annotates
	}

	for _, comment := range commentGroup.List {
		if comment == nil {
			continue
		}

		text := comment.Text
		text = strings.TrimSpace(text)
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)

		if !strings.HasPrefix(text, "@polygon") {
			continue
		}

		text = strings.TrimPrefix(text, "@polygon")
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}

		annotate := &Annotate{
			Name:      text,
			Variables: make(map[string]string),
		}

		spaceIndex := strings.Index(text, " ")
		if spaceIndex > 0 {
			annotate.Name = text[:spaceIndex]
			varsText := strings.TrimSpace(text[spaceIndex:])

			parts := strings.Fields(varsText)
			for _, part := range parts {
				colonIndex := strings.Index(part, ":")
				if colonIndex > 0 {
					key := part[:colonIndex]
					value := part[colonIndex+1:]
					annotate.Variables[key] = value
				}
			}
		}

		annotates = append(annotates, annotate)
	}

	return annotates
}
