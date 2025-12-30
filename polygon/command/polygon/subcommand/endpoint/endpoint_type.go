package endpoint

import (
	"go/token"
	"regexp"

	"go.scnd.dev/open/polygon/utility/code"
)

// EndpointInfo represents comprehensive information about an endpoint
type EndpointInfo struct {
	Name        string       // Handler function name
	Method      string       // HTTP method (GET, POST, etc.)
	Path        string       // Endpoint path
	Tag         string       // Tag extracted from path
	ReturnType  string       // Response type from response.Success()
	ErrorType   string       // Error type
	QueryType   string       // Query parameter type
	BodyType    string       // Request body type from c.Bind().Body()
	FormType    string       // Form type from c.Bind().Form()
	FormFields  []*FormField // Form fields when c.Bind().Form() is detected
	Description string       // Description from function comments
}

// RouteInfo represents information about a route registration
type RouteInfo struct {
	Method      string // HTTP method
	Path        string // Route path
	Group       string // Route group name
	HandlerName string // Handler function name
}

// FormField represents a form field definition
type FormField struct {
	Name     string // Field name
	Type     string // Field type
	Required bool   // Whether field is required
	IsFile   bool   // Whether field is a file
}

// VariableInfo represents variable type information for AST tracking
type VariableInfo struct {
	Name     string    // Variable name
	Type     string    // Variable type
	IsArray  bool      // Whether variable is an array
	Position token.Pos // Position in source
}

// Scanner handles AST scanning for endpoint patterns
type Scanner struct {
	Config    *Config
	Routes    []RouteInfo
	Endpoints []EndpointInfo
	FileSet   *token.FileSet
	Parser    *code.Parser
	// AST tracking
	Variables map[string]map[string]VariableInfo // filename -> varName -> VariableInfo
}

// Generator handles file generation
type Generator struct {
	Config    *Config
	Endpoints []EndpointInfo
}

// NewGenerator creates a new generator instance
func NewGenerator(config *Config, endpoints []EndpointInfo) *Generator {
	return &Generator{
		Config:    config,
		Endpoints: endpoints,
	}
}

// AST parsing result structure
type ParseResult struct {
	Endpoints []EndpointInfo
	Routes    []RouteInfo
	Parser    *code.Parser
}

// Form field detection from struct types
type StructFieldInfo struct {
	Name     string
	Type     string
	Tag      string // Struct tag information
	Required bool
}

// Common regex patterns for AST parsing
var (
	// Route extraction regex: groupName.Method("/path", handler.Func)
	routeRegex = regexp.MustCompile(`(\w+)\.(Get|Post|Put|Delete|Patch)\("([^"]+)",\s*\w+\.(\w+)\)`)

	// Handler function detection
	handlerRegex = regexp.MustCompile(`func\s+\([^)]*\)\s+(Handle[^\s(]+)`)

	// Import detection for response package
	responseImportRegex = regexp.MustCompile(`(?s)import\s*\([^)]*"go\.scnd\.dev/open/polygon/compat/response"[^)]*\)`)

	// Response success call patterns
	responseSuccessPatterns = []string{
		"response.Success",
		"response.SuccessResponse",
	}
)
