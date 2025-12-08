package sequel

import (
	"fmt"
	"log"

	"go.scnd.dev/polygon/pol/index"
)

func Schema(app index.App) error {
	// * 1. new parser (once)
	parser, err := NewParser(app)
	if err != nil {
		return fmt.Errorf("error creating parser: %w", err)
	}

	// * 2. process each directory for summary and models
	for dirName := range parser.Connections {
		log.Printf("Processing schema and models for %s...", dirName)

		// * call Summary for each directory
		if err := Summary(parser, dirName); err != nil {
			log.Printf("Error generating summary for %s: %v", dirName, err)
			continue
		}

		// * call Model for each directory
		if err := Model(parser, dirName); err != nil {
			log.Printf("Error generating models for %s: %v", dirName, err)
			continue
		}

		log.Printf("Generated schema and models for %s", dirName)
	}

	return nil
}
