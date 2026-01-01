package endpoint

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.scnd.dev/open/polygon/utility/code"
)

// GenerateMarkdown generates the markdown documentation file using swaggermd format
func (g *Generator) GenerateMarkdown(parser *code.Parser) error {
	log.Printf("generating declaration.md...")

	// Create output directory
	outputDir := filepath.Join("generate", "swagger")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate markdown directly from endpoint info using shared code parser
	return g.generateMarkdownFromEndpoints(outputDir, parser)
}

// generateMarkdownFromEndpoints generates markdown directly from endpoint info
func (g *Generator) generateMarkdownFromEndpoints(outputDir string, parser *code.Parser) error {
	var output strings.Builder

	// Write file header
	output.WriteString("# API Documentation\n\n")
	output.WriteString(fmt.Sprintf("Generated at: %s\n\n", time.Now().Format(time.RFC3339)))
	output.WriteString("This documentation is automatically generated from endpoint handlers.\n\n")

	// Write endpoint sections using swaggermd format
	for _, endpoint := range g.Endpoints {
		output.WriteString(g.formatEndpointMarkdown(endpoint, parser))
	}

	// Write to file
	outputPath := filepath.Join(outputDir, "declaration.md")
	return os.WriteFile(outputPath, []byte(output.String()), 0644)
}

// formatEndpointMarkdown formats an endpoint directly from EndpointInfo using code parser
func (g *Generator) formatEndpointMarkdown(endpoint EndpointInfo, parser *code.Parser) string {
	var output strings.Builder

	// Generate service name: backend.tag.operationId
	serviceName := fmt.Sprintf("backend.%s.%s", endpoint.Tag, strings.TrimPrefix(endpoint.Name, "Handle"))
	output.WriteString(fmt.Sprintf("# %s\n\n", serviceName))

	// Add query parameters
	if endpoint.QueryType != "" {
		output.WriteString(" - query\n")
		output.WriteString(g.formatQueryParameters(endpoint.QueryType, parser))
		output.WriteString("\n")
	}

	// Add form parameters when c.Bind().Form() is detected
	if endpoint.FormType != "" {
		output.WriteString(" - form\n")
		output.WriteString(g.formatFormParameters(endpoint, parser))
		output.WriteString("\n")
	}

	// Add body parameters
	if endpoint.BodyType != "" {
		output.WriteString(" - body\n")
		output.WriteString(g.formatBodyParameters(endpoint.BodyType, parser))
		output.WriteString("\n")
	}

	// Add response
	output.WriteString(" - response\n")
	output.WriteString(g.formatResponseParameters(endpoint.ReturnType, parser))
	output.WriteString("\n")

	return output.String()
}

// formatQueryParameters formats query parameters using code parser
func (g *Generator) formatQueryParameters(queryType string, parser *code.Parser) string {
	if parser == nil {
		return fmt.Sprintf("   - query: %s | \"example\"\n", queryType)
	}

	// Try to find struct details for the query type
	structInfo := g.findStructInfo(queryType, parser)
	if structInfo != nil {
		return g.formatStructFields(structInfo, parser, 3, true)
	}

	return fmt.Sprintf("   - query: %s | \"example\"\n", queryType)
}

// formatFormParameters formats form parameters using code parser
func (g *Generator) formatFormParameters(endpoint EndpointInfo, parser *code.Parser) string {
	if len(endpoint.FormFields) > 0 {
		// Use pre-extracted form fields
		var output strings.Builder
		for _, field := range endpoint.FormFields {
			required := ""
			if field.Required {
				required = " (required)"
			}
			example := getExampleForType(field.Type)
			output.WriteString(fmt.Sprintf("   - %s: %s%s | %s\n", field.Name, field.Type, required, example))
		}
		return output.String()
	}

	// Try to find struct details for the form type
	structInfo := g.findStructInfo(endpoint.FormType, parser)
	if structInfo != nil {
		return g.formatStructFields(structInfo, parser, 3, true)
	}

	return fmt.Sprintf("   - form: %s | \"example\"\n", endpoint.FormType)
}

// formatBodyParameters formats body parameters using code parser
func (g *Generator) formatBodyParameters(bodyType string, parser *code.Parser) string {
	if parser == nil {
		return fmt.Sprintf("   - body: %s | \"example\"\n", bodyType)
	}

	// Check if it's an array type
	if strings.HasSuffix(bodyType, "[]") {
		arrayType := strings.TrimSuffix(bodyType, "[]")
		structInfo := g.findStructInfo(arrayType, parser)
		if structInfo != nil {
			return fmt.Sprintf("   - items: %s[]\n%s", arrayType, g.formatStructFields(structInfo, parser, 5, true))
		}
		return fmt.Sprintf("   - items: %s[]\n", arrayType)
	}

	// Try to find struct details for the body type
	structInfo := g.findStructInfo(bodyType, parser)
	if structInfo != nil {
		return g.formatStructFields(structInfo, parser, 3, true)
	}

	return fmt.Sprintf("   - body: %s | \"example\"\n", bodyType)
}

// formatResponseParameters formats response parameters using code parser
func (g *Generator) formatResponseParameters(responseType string, parser *code.Parser) string {
	if parser == nil {
		return fmt.Sprintf("   - response: %s | \"example\"\n", responseType)
	}

	// Extract the actual type from response.GenericResponse[Type] or response.GenericResponse[[]Type]
	actualType := g.extractTypeFromResponse(responseType)
	if actualType == "" {
		return fmt.Sprintf("   - response: %s | \"example\"\n", responseType)
	}

	// Check if it's an array type
	if strings.HasSuffix(actualType, "[]") {
		arrayType := strings.TrimSuffix(actualType, "[]")
		structInfo := g.findStructInfo(arrayType, parser)
		if structInfo != nil {
			return fmt.Sprintf("   - items: %s[]\n%s", arrayType, g.formatStructFields(structInfo, parser, 5, true))
		}
		return fmt.Sprintf("   - items: %s[]\n", arrayType)
	}

	// Try to find struct details for the response type
	structInfo := g.findStructInfo(actualType, parser)
	if structInfo != nil {
		return g.formatStructFields(structInfo, parser, 3, true)
	}

	return fmt.Sprintf("   - response: %s | \"example\"\n", responseType)
}

// findStructInfo finds struct information from the code parser
func (g *Generator) findStructInfo(typeName string, parser *code.Parser) *code.Struct {
	if parser == nil || parser.Module == nil {
		return nil
	}

	// Handle package-prefixed types (e.g., "payload.WishCreateRequest")
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		packageName := parts[0]
		onlyTypeName := parts[1]

		// Find the package by directory name or package name
		for _, pkg := range parser.Module.Packages {
			if (pkg.DirectoryName != nil && *pkg.DirectoryName == packageName) ||
				(pkg.PackageName != nil && *pkg.PackageName == packageName) {
				_, structInfo, _, _ := pkg.EntityByName(onlyTypeName)
				if structInfo != nil {
					return structInfo
				}
			}
		}

		// If not found by package name, try searching all packages for the type name
		for _, pkg := range parser.Module.Packages {
			_, structInfo, _, _ := pkg.EntityByName(onlyTypeName)
			if structInfo != nil {
				return structInfo
			}
		}

		return nil
	}

	// Look through all packages for the struct (no package prefix)
	for _, pkg := range parser.Module.Packages {
		_, structInfo, _, _ := pkg.EntityByName(typeName)
		if structInfo != nil {
			return structInfo
		}
	}

	return nil
}

// formatStructFields formats struct fields in swaggermd style
func (g *Generator) formatStructFields(structInfo *code.Struct, parser *code.Parser, indent int, includeExamples bool) string {
	var output strings.Builder
	indentation := strings.Repeat(" ", indent)

	if structInfo.Fields == nil || len(structInfo.Fields) == 0 {
		return fmt.Sprintf("%s- %s: interface{} | \"example\"\n", indentation, *structInfo.Name)
	}

	for _, field := range structInfo.Fields {
		if field.Name == nil || field.Type == nil {
			continue
		}

		required := ""
		// Check if field is required (could be enhanced to check struct tags)
		if g.isFieldRequired(field) {
			required = " (required)"
		}

		// Handle pointer types
		fieldType := *field.Type
		if strings.HasPrefix(fieldType, "*") {
			fieldType = strings.TrimPrefix(fieldType, "*")
		}

		// Handle array types
		if strings.HasPrefix(fieldType, "[]") {
			arrayType := strings.TrimPrefix(fieldType, "[]")
			if strings.HasPrefix(arrayType, "*") {
				arrayType = strings.TrimPrefix(arrayType, "*")
			}
			output.WriteString(fmt.Sprintf("%s- %s: %s[]%s\n", indentation, *field.Name, arrayType, required))
		} else {
			example := ""
			if includeExamples {
				example = fmt.Sprintf(" | %s", getExampleForType(fieldType))
			}
			output.WriteString(fmt.Sprintf("%s- %s: %s%s%s\n", indentation, *field.Name, fieldType, required, example))
		}

		// Expand nested struct types
		nestedStructType := fieldType
		if strings.HasPrefix(nestedStructType, "[]") {
			nestedStructType = strings.TrimPrefix(nestedStructType, "[]")
		}

		// Try to find and expand nested struct type
		if !code.IsBuiltinType(nestedStructType) {
			nestedStruct := g.findStructInfo(nestedStructType, parser)
			if nestedStruct != nil {
				output.WriteString(g.formatStructFields(nestedStruct, parser, indent+2, false))
			}
		}
	}

	return output.String()
}

// extractTypeFromResponse extracts the actual type from response type expressions
func (g *Generator) extractTypeFromResponse(responseType string) string {
	// Handle response.GenericResponse[Type] pattern
	if strings.HasPrefix(responseType, "response.GenericResponse[") && strings.HasSuffix(responseType, "]") {
		returnType := strings.TrimPrefix(responseType, "response.GenericResponse[")
		return strings.TrimSuffix(returnType, "]")
	}

	// Handle response.GenericResponse[[]Type] pattern
	if strings.HasPrefix(responseType, "response.GenericResponse[[]") && strings.HasSuffix(responseType, "]") {
		returnType := strings.TrimPrefix(responseType, "response.GenericResponse[")
		return strings.TrimSuffix(returnType, "]")
	}

	return ""
}

// isFieldRequired checks if a field is required based on struct tags or naming conventions
func (g *Generator) isFieldRequired(field *code.Field) bool {
	if field.Tags == nil {
		// Use naming convention: fields with "ID" or uppercase names are often required
		if field.Name != nil {
			name := *field.Name
			return strings.HasSuffix(name, "ID") || strings.HasSuffix(name, "Id") || name == name
		}
		return false
	}

	// Check struct tags for required validation
	for _, tag := range field.Tags {
		if tag.Name == nil || tag.Value == nil {
			continue
		}
		if *tag.Name == "validate" && strings.Contains(*tag.Value, "required") {
			return true
		}
		if *tag.Name == "json" && !strings.Contains(*tag.Value, "omitempty") {
			return true
		}
	}

	return false
}

// Helper functions

func getExampleForType(paramType string) string {
	switch paramType {
	case "string":
		return "\"example\""
	case "number":
		return "1"
	case "integer":
		return "1"
	case "boolean":
		return "true"
	case "file":
		return "file"
	default:
		return "\"example\""
	}
}
