package endpoint

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// Parse performs the complete parsing workflow
func Parse(config *Config) (*ParseResult, error) {
	log.Printf("starting endpoint parsing...")

	// Step 1: Parse AST to extract routes and endpoints
	parseResult, err := ParseAst(config)
	if err != nil {
		return nil, fmt.Errorf("AST parsing failed: %w", err)
	}

	log.Printf("parsed %d routes and %d endpoints", len(parseResult.Routes), len(parseResult.Endpoints))

	return parseResult, nil
}

// ParseEndpoints parses only endpoints (for backward compatibility)
func ParseEndpoints(config *Config) ([]EndpointInfo, error) {
	result, err := Parse(config)
	if err != nil {
		return nil, err
	}
	return result.Endpoints, nil
}

// ParseRoutes parses only routes from the endpoint file
func ParseRoutes(config *Config) ([]RouteInfo, error) {
	result, err := Parse(config)
	if err != nil {
		return nil, err
	}
	return result.Routes, nil
}

// ParseWithValidation performs parsing with additional validation
func ParseWithValidation(config *Config) ([]EndpointInfo, error) {
	result, err := Parse(config)
	if err != nil {
		return nil, err
	}

	// Validate that all endpoints have corresponding routes
	validationErrors := validateEndpoints(result.Endpoints, result.Routes)
	if len(validationErrors) > 0 {
		log.Printf("validation warnings: %s", strings.Join(validationErrors, "; "))
	}

	return result.Endpoints, nil
}

// validateEndpoints validates that endpoints are properly configured
func validateEndpoints(endpoints []EndpointInfo, routes []RouteInfo) []string {
	var errors []string
	routeMap := make(map[string]bool)

	// Create route map
	for _, route := range routes {
		routeMap[route.HandlerName] = true
	}

	// Check each endpoint has a corresponding route
	for _, endpoint := range endpoints {
		if !routeMap[endpoint.Name] {
			errors = append(errors, fmt.Sprintf("endpoint %s has no corresponding route", endpoint.Name))
		}

		// Validate endpoint has at least a body or form type
		if endpoint.BodyType == "" && endpoint.FormType == "" {
			errors = append(errors, fmt.Sprintf("endpoint %s has no body or form type", endpoint.Name))
		}

		// Validate path is not empty
		if endpoint.Path == "" {
			errors = append(errors, fmt.Sprintf("endpoint %s has empty path", endpoint.Name))
		}

		// Validate method is not empty
		if endpoint.Method == "" {
			errors = append(errors, fmt.Sprintf("endpoint %s has empty method", endpoint.Name))
		}
	}

	return errors
}

// ParseEndpointFile parses a specific endpoint file
func ParseEndpointFile(filename string) ([]EndpointInfo, error) {
	// Create a minimal config for single file parsing
	config := &Config{
		EndpointDir:  filepath.Dir(filename),
		EndpointFile: filepath.Base(filename),
	}

	result, err := Parse(config)
	if err != nil {
		return nil, err
	}
	return result.Endpoints, nil
}

// ParseConfig loads and validates the configuration
func ParseConfig(app interface{}) (*Config, error) {
	// This would integrate with the existing LoadConfig function
	// For now, return an error to indicate it needs implementation
	return nil, fmt.Errorf("ParseConfig not implemented - use LoadConfig instead")
}

// FilterEndpointsByMethod filters endpoints by HTTP method
func FilterEndpointsByMethod(endpoints []EndpointInfo, method string) []EndpointInfo {
	var filtered []EndpointInfo
	method = strings.ToLower(method)

	for _, endpoint := range endpoints {
		if strings.ToLower(endpoint.Method) == method {
			filtered = append(filtered, endpoint)
		}
	}

	return filtered
}

// FilterEndpointsByTag filters endpoints by tag
func FilterEndpointsByTag(endpoints []EndpointInfo, tag string) []EndpointInfo {
	var filtered []EndpointInfo

	for _, endpoint := range endpoints {
		if endpoint.Tag == tag {
			filtered = append(filtered, endpoint)
		}
	}

	return filtered
}

// GroupEndpointsByTag groups endpoints by their tag
func GroupEndpointsByTag(endpoints []EndpointInfo) map[string][]EndpointInfo {
	grouped := make(map[string][]EndpointInfo)

	for _, endpoint := range endpoints {
		tag := endpoint.Tag
		if tag == "" {
			tag = "default"
		}
		grouped[tag] = append(grouped[tag], endpoint)
	}

	return grouped
}

// ValidateEndpointInfo performs comprehensive validation of a single endpoint
func ValidateEndpointInfo(endpoint EndpointInfo) []string {
	var errors []string

	// Name validation
	if endpoint.Name == "" {
		errors = append(errors, "endpoint name is required")
	}

	// Method validation
	if endpoint.Method == "" {
		errors = append(errors, "endpoint method is required")
	} else if !isValidHTTPMethod(endpoint.Method) {
		errors = append(errors, fmt.Sprintf("invalid HTTP method: %s", endpoint.Method))
	}

	// Path validation
	if endpoint.Path == "" {
		errors = append(errors, "endpoint path is required")
	} else if !strings.HasPrefix(endpoint.Path, "/") {
		errors = append(errors, "endpoint path must start with /")
	}

	// Tag validation
	if endpoint.Tag == "" {
		errors = append(errors, "endpoint tag is required")
	}

	// Response type validation
	if endpoint.ReturnType == "" {
		errors = append(errors, "endpoint return type is required")
	}

	// Error type validation
	if endpoint.ErrorType == "" {
		errors = append(errors, "endpoint error type is required")
	}

	// Body/Form type validation
	if endpoint.BodyType == "" && endpoint.FormType == "" {
		errors = append(errors, "endpoint must have either body type or form type")
	}

	// Form validation
	if endpoint.FormType != "" && len(endpoint.FormFields) == 0 {
		errors = append(errors, "endpoint with form type should have form fields defined")
	}

	return errors
}

// isValidHTTPMethod checks if the method is a valid HTTP method
func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	method = strings.ToUpper(method)

	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}

	return false
}

// SanitizeEndpointInfo cleans and normalizes endpoint data
func SanitizeEndpointInfo(endpoint EndpointInfo) EndpointInfo {
	// Normalize method
	endpoint.Method = strings.ToUpper(endpoint.Method)

	// Normalize path
	if !strings.HasPrefix(endpoint.Path, "/") {
		endpoint.Path = "/" + endpoint.Path
	}

	// Normalize tag
	if endpoint.Tag == "" {
		endpoint.Tag = extractTagFromPath(endpoint.Path)
	}

	// Set defaults
	if endpoint.ErrorType == "" {
		endpoint.ErrorType = "response.ErrorResponse"
	}

	if endpoint.ReturnType == "" {
		endpoint.ReturnType = "response.SuccessResponse"
	}

	return endpoint
}

// MergeEndpoints merges endpoint information from multiple sources
func MergeEndpoints(endpointSets ...[]EndpointInfo) []EndpointInfo {
	merged := make([]EndpointInfo, 0)
	seen := make(map[string]bool)

	for _, endpoints := range endpointSets {
		for _, endpoint := range endpoints {
			if !seen[endpoint.Name] {
				merged = append(merged, endpoint)
				seen[endpoint.Name] = true
			}
		}
	}

	return merged
}

// ParseEndpointName extracts operation ID from endpoint name
func ParseEndpointName(handlerName string) string {
	// Remove "Handle" prefix and convert to operation ID format
	name := strings.TrimPrefix(handlerName, "Handle")
	if name == "" {
		return strings.ToLower(handlerName)
	}

	// Convert to camelCase starting with lowercase
	if len(name) == 0 {
		return ""
	}

	return strings.ToLower(name[:1]) + name[1:]
}

// ParsePathComponents extracts components from a path
func ParsePathComponents(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}
