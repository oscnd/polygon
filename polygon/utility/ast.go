package utility

import (
	"fmt"
	"go/ast"
	"strings"
)

// ExprToString converts an AST expression to string representation
func ExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + ExprToString(t.X)
	case *ast.ArrayType:
		return "[]" + ExprToString(t.Elt)
	case *ast.SelectorExpr:
		xStr := ExprToString(t.X)
		// * if the selector is "index.Type", simplify to just "Type" since it's in the same package
		if xStr == "index" {
			return t.Sel.Name
		}
		return xStr + "." + t.Sel.Name
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", ExprToString(t.Key), ExprToString(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.Ellipsis:
		return "..." + ExprToString(t.Elt)
	case *ast.ChanType:
		return "chan " + ExprToString(t.Value)
	case *ast.FuncType:
		// * handle function types
		params := []string{}
		if t.Params != nil {
			for _, param := range t.Params.List {
				paramType := ExprToString(param.Type)
				params = append(params, paramType)
			}
		}
		results := []string{}
		if t.Results != nil {
			for _, result := range t.Results.List {
				resultType := ExprToString(result.Type)
				results = append(results, resultType)
			}
		}
		sig := "func(" + strings.Join(params, ", ") + ")"
		if len(results) > 0 {
			sig += " (" + strings.Join(results, ", ") + ")"
		}
		return sig
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// IsBuiltinType checks if a type is a Go built-in type that doesn't need import
func IsBuiltinType(typ string) bool {
	builtinTypes := map[string]bool{
		"uint64":   true,
		"string":   true,
		"bool":     true,
		"int":      true,
		"int8":     true,
		"int16":    true,
		"int32":    true,
		"int64":    true,
		"uint":     true,
		"uint8":    true,
		"uint16":   true,
		"uint32":   true,
		"float32":  true,
		"float64":  true,
		"error":    true,
		"rune":     true,
		"byte":     true,
		"struct{}": true,
	}

	return builtinTypes[typ]
}
