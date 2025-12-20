package sequel

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"go.scnd.dev/open/polygon/command/polygon/index"
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

	// * run sqlc generate programmatically
	log.Printf("running sqlc generate...")
	if err := RunSqlcGenerate(); err != nil {
		log.Printf("Error running sqlc generate: %v", err)
		return fmt.Errorf("failed to run sqlc generate: %w", err)
	}

	// * apply type replacements
	log.Printf("applying type replacements...")
	if err := ReplaceGeneratedTypes(parser); err != nil {
		log.Printf("Error applying type replacements: %v", err)
		return fmt.Errorf("failed to apply type replacements: %w", err)
	}
	return nil
}

// RunSqlcGenerate runs sqlc generate programmatically
func RunSqlcGenerate() error {
	// Change to the directory containing sqlc.yml
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find and change to the directory containing sqlc.yml
	sqlcDir := originalDir
	for {
		if _, err := os.Stat("sqlc.yml"); err == nil {
			break
		}
		parent := filepath.Dir(sqlcDir)
		if parent == sqlcDir {
			return fmt.Errorf("sqlc.yml not found in any parent directory")
		}
		sqlcDir = parent
	}

	if err := os.Chdir(sqlcDir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", sqlcDir, err)
	}
	defer os.Chdir(originalDir)

	// Run sqlc generate
	cmd := exec.Command("sqlc", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlc generate failed: %w", err)
	}

	return nil
}
