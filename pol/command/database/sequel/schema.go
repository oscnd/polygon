package sequel

import (
	"fmt"
	"log"

	"go.scnd.dev/polygon/pol/index"
)

func Schema(app index.App) error {
	// * construct parser
	parser, err := NewParser(app)
	if err != nil {
		return fmt.Errorf("error creating parser: %w", err)
	}

	// * process each connection directory
	for dirName := range parser.Connections {
		log.Printf("processing schema, models, and queriers for %s...", dirName)

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

		// * call Querier for each directory
		if err := Querier(parser, dirName); err != nil {
			log.Printf("Error generating queriers for %s: %v", dirName, err)
			continue
		}

		log.Printf("generated schema, models, and queriers for %s", dirName)
	}

	return nil
}
