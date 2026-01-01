package endpoint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon/utility/code"
)

// ParseAst performs comprehensive AST parsing of endpoint files
func ParseAst(config *Config) (*ParseResult, error) {
	// Get absolute path for the current directory
	absPath, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create code parser once and reuse it
	parser, err := code.NewParser(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create code parser: %w", err)
	}

	// Parse the module to get struct information
	if err := parser.ParseModule(); err != nil {
		log.Printf("warning: failed to parse module for struct information: %v", err)
	}

	scanner := &Scanner{
		Config:    config,
		FileSet:   token.NewFileSet(),
		Variables: make(map[string]map[string]VariableInfo),
		Parser:    parser,
	}

	// First, extract routes from the main endpoint file
	routes, err := scanner.extractRoutes(config.EndpointFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract routes: %w", err)
	}
	scanner.Routes = routes

	// Then scan all endpoint files for handlers
	endpoints, err := scanner.scanEndpointFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to scan endpoint files: %w", err)
	}

	return &ParseResult{
		Endpoints: endpoints,
		Routes:    routes,
		Parser:    parser,
	}, nil
}

// extractRoutes parses the main endpoint file to extract route registrations
func (s *Scanner) extractRoutes(endpointFile string) ([]RouteInfo, error) {
	node, err := parser.ParseFile(s.FileSet, endpointFile, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint file: %w", err)
	}

	var routes []RouteInfo

	// Walk through AST to find route registrations
	ast.Inspect(node, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Look for pattern: group.Method("/path", handler.Func)
		route := s.extractRouteFromCall(callExpr)
		if route != nil {
			routes = append(routes, *route)
			log.Printf("found route: %s %s -> %s", route.Method, route.Path, route.HandlerName)
		}

		return true
	})

	return routes, nil
}

// extractRouteFromCall extracts route information from a call expression
func (s *Scanner) extractRouteFromCall(call *ast.CallExpr) *RouteInfo {
	if len(call.Args) < 2 {
		return nil
	}

	// Check if this is a method call: group.Method(...)
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// Check method name
	method := strings.ToUpper(selExpr.Sel.Name)
	if !isHTTPMethod(method) {
		return nil
	}

	// Extract path from first argument (should be a basic literal)
	path, ok := call.Args[0].(*ast.BasicLit)
	if !ok || path.Kind != token.STRING {
		return nil
	}
	pathStr := strings.Trim(path.Value, `"`)

	// Extract handler from second argument
	handlerExpr := call.Args[1]
	handlerName := s.extractHandlerName(handlerExpr)
	if handlerName == "" {
		return nil
	}

	// Extract group name from the selector expression
	groupName := s.extractGroupName(selExpr.X)
	if groupName == "" {
		groupName = "api" // default group
	}

	// Build full path
	fullPath := pathStr
	if groupName != "app" && groupName != "api" {
		fullPath = "/" + strings.ToLower(groupName) + pathStr
	}

	return &RouteInfo{
		Method:      strings.ToLower(method),
		Path:        fullPath,
		Group:       groupName,
		HandlerName: handlerName,
	}
}

// extractHandlerName extracts handler function name from expression
func (s *Scanner) extractHandlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// handler.HandleFunc
		return e.Sel.Name
	case *ast.Ident:
		// direct function reference
		return e.Name
	default:
		return ""
	}
}

// extractGroupName extracts group name from expression
func (s *Scanner) extractGroupName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// scanEndpointFiles scans all endpoint files for handler implementations
func (s *Scanner) scanEndpointFiles() ([]EndpointInfo, error) {
	endpointDir := s.Config.EndpointDir

	var endpoints []EndpointInfo
	err := filepath.Walk(endpointDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		fileInfo, ok := info.(interface{ Name() string })
		if !ok || !strings.HasSuffix(fileInfo.Name(), ".go") || strings.HasSuffix(fileInfo.Name(), "_test.go") {
			return nil
		}

		// Skip the main endpoint file we already processed for routes
		if filepath.Base(path) == filepath.Base(s.Config.EndpointFile) {
			return nil
		}

		log.Printf("scanning endpoint file: %s", path)

		fileEndpoints, err := s.parseEndpointFile(path)
		if err != nil {
			log.Printf("failed to parse endpoint file %s: %v", path, err)
			return nil // Continue with other files
		}

		endpoints = append(endpoints, fileEndpoints...)
		return nil
	})

	return endpoints, err
}

// parseEndpointFile parses a single endpoint file for handler implementations
func (s *Scanner) parseEndpointFile(filename string) ([]EndpointInfo, error) {
	node, err := parser.ParseFile(s.FileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	// Initialize variable tracking for this file
	if s.Variables[filename] == nil {
		s.Variables[filename] = make(map[string]VariableInfo)
	}

	var endpoints []EndpointInfo

	// Walk through AST to find function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Only process handler functions (prefix "Handle")
		if !strings.HasPrefix(funcDecl.Name.Name, "Handle") {
			return true
		}

		endpoint := s.extractEndpointFromFunction(funcDecl, filename)
		if endpoint != nil {
			endpoints = append(endpoints, *endpoint)
			log.Printf("found endpoint: %s %s (handler: %s)", endpoint.Method, endpoint.Path, endpoint.Name)
		}

		return true
	})

	return endpoints, nil
}

// extractEndpointFromFunction extracts endpoint information from a function declaration
func (s *Scanner) extractEndpointFromFunction(funcDecl *ast.FuncDecl, filename string) *EndpointInfo {
	if funcDecl.Body == nil {
		return nil
	}

	// Find matching route for this handler
	var route *RouteInfo
	for _, r := range s.Routes {
		if r.HandlerName == funcDecl.Name.Name {
			route = &r
			break
		}
	}
	if route == nil {
		return nil
	}

	endpoint := &EndpointInfo{
		Name:      funcDecl.Name.Name,
		Method:    route.Method,
		Path:      route.Path,
		ErrorType: "response.ErrorResponse",
		Tag:       extractTagFromPath(route.Path),
	}

	// Extract description from function comments
	if funcDecl.Doc != nil {
		endpoint.Description = strings.TrimSpace(funcDecl.Doc.Text())
	}

	// Scan function body for patterns
	s.scanFunctionBody(funcDecl.Body, filename, endpoint)

	return endpoint
}

// scanFunctionBody scans function body for c.Bind(), response.Success, and variable patterns
func (s *Scanner) scanFunctionBody(body *ast.BlockStmt, filename string, endpoint *EndpointInfo) {
	// Track variables in this function
	variables := make(map[string]VariableInfo)

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.DeclStmt:
			// Variable declarations: var body *Type
			s.trackVariableDeclaration(node, variables)

		case *ast.AssignStmt:
			// Variable assignments: body := new(Type) or body := &Type{}
			s.trackVariableAssignment(node, variables)

		case *ast.CallExpr:
			// Function calls: c.Bind().Body(body), c.Bind().Form(body), response.Success()
			s.scanFunctionCall(node, variables, endpoint)

		case *ast.ReturnStmt:
			// Return statements: return c.JSON(response.Success(c, variable))
			s.scanReturnStatement(node, variables, endpoint)
		}
		return true
	})

	// Store variables for this file
	s.Variables[filename] = variables
}

// trackVariableDeclaration tracks variable declarations like: var body *Type
func (s *Scanner) trackVariableDeclaration(decl *ast.DeclStmt, variables map[string]VariableInfo) {
	genDecl, ok := decl.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range valueSpec.Names {
			varType := "interface{}"
			isArray := false

			if valueSpec.Type != nil {
				varType, isArray = s.extractTypeString(valueSpec.Type)
			}

			variables[name.Name] = VariableInfo{
				Name:    name.Name,
				Type:    varType,
				IsArray: isArray,
			}
		}
	}
}

// trackVariableAssignment tracks variable assignments like: body := new(Type) or body := &Type{}
func (s *Scanner) trackVariableAssignment(assign *ast.AssignStmt, variables map[string]VariableInfo) {
	if assign.Tok != token.DEFINE {
		return
	}

	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}

		if i >= len(assign.Rhs) {
			continue
		}

		varType := "interface{}"
		isArray := false

		switch rhs := assign.Rhs[i].(type) {
		case *ast.CallExpr:
			// new(Type) or make([]Type, 0)
			varType, isArray = s.extractTypeFromCall(rhs)

		case *ast.UnaryExpr:
			// &Type{...}
			if rhs.Op == token.AND {
				varType, isArray = s.extractTypeString(rhs.X)
			}

		case *ast.CompositeLit:
			// Type{...} or []Type{...}
			if rhs.Type != nil {
				varType, isArray = s.extractTypeString(rhs.Type)
			}
		}

		variables[ident.Name] = VariableInfo{
			Name:    ident.Name,
			Type:    varType,
			IsArray: isArray,
		}
	}
}

// scanFunctionCall scans function calls for c.Bind() patterns
func (s *Scanner) scanFunctionCall(call *ast.CallExpr, variables map[string]VariableInfo, endpoint *EndpointInfo) {
	// Check for c.Bind().Body(body) or c.Bind().Form(body) pattern
	if !s.isBindCallPattern(call) {
		return
	}

	if len(call.Args) != 1 {
		return
	}

	// Extract variable name
	ident, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return
	}

	varName := ident.Name
	varInfo, exists := variables[varName]
	if !exists {
		// Fallback to interface{}
		varInfo = VariableInfo{Type: "interface{}"}
	}

	// Check if this is .Body() or .Form() call
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	switch selExpr.Sel.Name {
	case "Body":
		endpoint.BodyType = varInfo.Type
	case "Form":
		endpoint.FormType = varInfo.Type
		// Extract form fields from struct type
		endpoint.FormFields = s.extractFormFields(varInfo.Type)
	}
}

// scanReturnStatement scans return statements for response.Success patterns
func (s *Scanner) scanReturnStatement(ret *ast.ReturnStmt, variables map[string]VariableInfo, endpoint *EndpointInfo) {
	if len(ret.Results) != 1 {
		return
	}

	// Look for: c.JSON(response.Success(c, variable))
	callExpr, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return
	}

	// Check if this is c.JSON()
	if !s.isCJSONCall(callExpr) {
		return
	}

	if len(callExpr.Args) != 1 {
		return
	}

	// Look for response.Success() inside
	successCall, ok := callExpr.Args[0].(*ast.CallExpr)
	if !ok {
		return
	}

	if !s.isResponseSuccessCall(successCall) {
		return
	}

	// Extract the variable argument from response.Success()
	successArg := successCall.Args[1] // response.Success(c, variable)
	if successArg == nil {
		return
	}

	// Determine the response type
	responseType := s.extractResponseVariableType(successArg, variables)
	if responseType != "" {
		endpoint.ReturnType = responseType
	}
}

// Helper functions for type extraction and pattern matching

func (s *Scanner) extractTypeString(typeExpr ast.Expr) (string, bool) {
	if typeExpr == nil {
		return "interface{}", false
	}

	switch t := typeExpr.(type) {
	case *ast.Ident:
		return t.Name, false
	case *ast.StarExpr:
		typeStr, isArray := s.extractTypeString(t.X)
		return "*" + typeStr, isArray
	case *ast.ArrayType:
		typeStr, _ := s.extractTypeString(t.Elt)
		return "[]" + typeStr, true
	case *ast.SelectorExpr:
		typeStr, _ := s.extractTypeString(t.X)
		return typeStr + "." + t.Sel.Name, false
	case *ast.MapType:
		keyType, _ := s.extractTypeString(t.Key)
		valueType, _ := s.extractTypeString(t.Value)
		return fmt.Sprintf("map[%s]%s", keyType, valueType), false
	case *ast.InterfaceType:
		return "interface{}", false
	case *ast.CompositeLit:
		// For &payload.Type{...}, the CompositeLit's Type field contains the actual type
		if t.Type != nil {
			return s.extractTypeString(t.Type)
		}
		return "interface{}", false
	default:
		log.Printf("extractTypeString: unhandled type %T", t)
		return "interface{}", false
	}
}

func (s *Scanner) extractTypeFromCall(call *ast.CallExpr) (string, bool) {
	if len(call.Args) == 0 {
		return "interface{}", false
	}

	// Check for new(Type)
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "new" {
		return s.extractTypeString(call.Args[0])
	}

	// Check for make([]Type, 0)
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "make" {
		if len(call.Args) >= 1 {
			return s.extractTypeString(call.Args[0])
		}
	}

	return "interface{}", false
}

func (s *Scanner) extractResponseVariableType(arg ast.Expr, variables map[string]VariableInfo) string {
	switch expr := arg.(type) {
	case *ast.Ident:
		// Direct variable reference
		if varInfo, exists := variables[expr.Name]; exists {
			if varInfo.IsArray {
				return fmt.Sprintf("response.GenericResponse[[]%s]", varInfo.Type)
			}
			return fmt.Sprintf("response.GenericResponse[%s]", varInfo.Type)
		}

	case *ast.UnaryExpr:
		// &Type{...}
		if expr.Op == token.AND {
			typeStr, isArray := s.extractTypeString(expr.X)
			if isArray {
				return fmt.Sprintf("response.GenericResponse[[]%s]", typeStr)
			}
			return fmt.Sprintf("response.GenericResponse[%s]", typeStr)
		}

	case *ast.CompositeLit:
		// Type{...}
		typeStr, isArray := s.extractTypeString(expr.Type)
		if isArray {
			return fmt.Sprintf("response.GenericResponse[[]%s]", typeStr)
		}
		return fmt.Sprintf("response.GenericResponse[%s]", typeStr)
	}

	return "response.SuccessResponse" // default fallback
}

func (s *Scanner) isBindCallPattern(call *ast.CallExpr) bool {
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check if this is .Body() or .Form() call
	if selExpr.Sel.Name != "Body" && selExpr.Sel.Name != "Form" {
		return false
	}

	// Check if the X of .Body()/.Form() is a CallExpr (c.Bind())
	bindCall, ok := selExpr.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	// Check if the function being called is .Bind
	bindSel, ok := bindCall.Fun.(*ast.SelectorExpr)
	if !ok || bindSel.Sel.Name != "Bind" {
		return false
	}

	return true
}

func (s *Scanner) isCJSONCall(call *ast.CallExpr) bool {
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return selExpr.Sel.Name == "JSON"
}

func (s *Scanner) isResponseSuccessCall(call *ast.CallExpr) bool {
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check for response.Success or similar patterns
	funType, _ := s.extractTypeString(selExpr.X)

	// Build the full function name: qualifier + "." + selectorName
	fullFuncName := funType + "." + selExpr.Sel.Name

	for _, pattern := range responseSuccessPatterns {
		if fullFuncName == pattern || strings.HasSuffix(fullFuncName, "."+pattern) {
			return selExpr.Sel.Name == "Success"
		}
	}
	return false
}

// Utility functions

func isHTTPMethod(method string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range methods {
		if method == m {
			return true
		}
	}
	return false
}

func extractTagFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "default"
}

// extractFormFields parses a struct type and extracts form field information
func (s *Scanner) extractFormFields(formType string) []*FormField {
	// Find the struct in the parsed packages
	structDef := s.findStructInPackages(nil, formType)
	if structDef == nil {
		log.Printf("struct %s not found", formType)
		return []*FormField{}
	}

	var formFields []*FormField
	for _, field := range structDef.Fields {
		if field.Name == nil || field.Type == nil {
			continue
		}

		formField := &FormField{
			Name:     *field.Name,
			Type:     s.getSwaggerType(*field.Type),
			Required: s.isRequiredField(field),
			IsFile:   s.isFileField(*field.Type),
		}

		formFields = append(formFields, formField)
	}

	return formFields
}

// findStructInPackages searches for a struct definition in parsed packages
func (s *Scanner) findStructInPackages(packages []*code.Package, structName string) *code.Struct {
	if s.Parser == nil || s.Parser.Module == nil {
		return nil
	}

	// Handle package-prefixed types (e.g., "payload.WishCreateRequest")
	if strings.Contains(structName, ".") {
		parts := strings.SplitN(structName, ".", 2)
		packageName := parts[0]
		typeName := parts[1]

		// Find the package by directory name or package name
		for _, pkg := range s.Parser.Module.Packages {
			if (pkg.DirectoryName != nil && *pkg.DirectoryName == packageName) ||
				(pkg.PackageName != nil && *pkg.PackageName == packageName) {
				_, structInfo, _, _ := pkg.EntityByName(typeName)
				if structInfo != nil {
					return structInfo
				}
			}
		}

		// If not found by package name, try searching all packages for the type name
		for _, pkg := range s.Parser.Module.Packages {
			_, structInfo, _, _ := pkg.EntityByName(typeName)
			if structInfo != nil {
				return structInfo
			}
		}

		return nil
	}

	// Look through all packages for the struct (no package prefix)
	for _, pkg := range s.Parser.Module.Packages {
		_, structInfo, _, _ := pkg.EntityByName(structName)
		if structInfo != nil {
			return structInfo
		}
	}

	return nil
}

// getSwaggerType converts Go type to Swagger type
func (s *Scanner) getSwaggerType(goType string) string {
	switch {
	case strings.HasPrefix(goType, "*"):
		return s.getSwaggerType(strings.TrimPrefix(goType, "*"))
	case strings.HasPrefix(goType, "[]"):
		// Special handling for file arrays
		if strings.Contains(goType, "FileHeader") {
			return "[]file"
		}
		return "[]string" // Default for arrays
	case goType == "string":
		return "string"
	case goType == "int", goType == "int64", goType == "uint64":
		return "integer"
	case goType == "float64", goType == "float32":
		return "number"
	case goType == "bool":
		return "boolean"
	case strings.HasSuffix(goType, "FileHeader"):
		return "file" // For multipart.FileHeader
	default:
		return "string"
	}
}

// isRequiredField checks if a field is required based on struct tags
func (s *Scanner) isRequiredField(field *code.Field) bool {
	if field.Tags == nil {
		return false
	}

	for _, tag := range field.Tags {
		if tag.Name == nil || tag.Value == nil {
			continue
		}

		// Check for validate:"required" tag
		if *tag.Name == "validate" && strings.Contains(*tag.Value, "required") {
			return true
		}

		// Check for binding:"required" tag
		if *tag.Name == "binding" && strings.Contains(*tag.Value, "required") {
			return true
		}
	}

	return false
}

// isFileField checks if a field type represents a file upload
func (s *Scanner) isFileField(goType string) bool {
	return strings.Contains(goType, "FileHeader")
}
