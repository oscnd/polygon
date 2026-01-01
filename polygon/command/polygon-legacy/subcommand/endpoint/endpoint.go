package endpoint

import (
	"log"

	"go.scnd.dev/open/polygon/command/polygon-legacy/index"
)

// EndpointGenerate is the main entry point for endpoint swagger generation
func EndpointGenerate(app index.App) error {
	log.Printf("starting endpoint swagger generation...")

	// Load configuration
	config, err := LoadConfig(app)
	if err != nil {
		return err
	}

	// Store the app reference in config for use by parser
	config.App = app

	log.Printf("loaded configuration: endpoint_dir=%s, endpoint_file=%s", config.EndpointDir, config.EndpointFile)

	// Parse endpoints using the new AST-based parser
	result, err := Parse(config)
	if err != nil {
		return err
	}

	if len(result.Endpoints) == 0 {
		log.Printf("no endpoints found with c.Bind().Body(body) or c.Bind().Form(body) patterns")
		return nil
	}

	// Create generator and generate files
	generator := NewGenerator(config, result.Endpoints)

	// Generate Go declaration file
	if err := generator.GenerateDeclaration(); err != nil {
		log.Printf("failed to generate Go declaration: %v", err)
		// Continue with markdown generation even if Go generation fails
	}

	// Generate markdown file using shared parser
	if err := generator.GenerateMarkdown(result.Parser); err != nil {
		log.Printf("failed to generate markdown: %v", err)
		// Return error if both generations fail
	}

	log.Printf("successfully generated swagger documentation for %d endpoints", len(result.Endpoints))
	return nil
}
