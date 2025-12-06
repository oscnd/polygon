package sequel

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"go.scnd.dev/polygon/external/sqlc/migrations"
)

func Schema() error {
	// * read polygon.yml to get dialect
	configPath := filepath.Join("polygon.yml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read polygon.yml: %w", err)
	}

	var polygonConfig PolygonConfig
	err = yaml.Unmarshal(configData, &polygonConfig)
	if err != nil {
		return fmt.Errorf("failed to parse polygon.yml: %w", err)
	}

	// * validate dialect
	switch polygonConfig.Dialect {
	case "postgres", "postgresql", "mysql", "sqlite":
		// * supported dialects
	default:
		return fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", polygonConfig.Dialect)
	}

	// * find all directories in ./sequel
	sequelDir := filepath.Join("sequel")
	entries, err := os.ReadDir(sequelDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("sequel directory not found: %s", sequelDir)
		}
		return fmt.Errorf("failed to read sequel directory: %w", err)
	}

	// * process each sequel directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		migrationDir := filepath.Join(sequelDir, dirName, "migration")

		// * check if migration directory exists
		if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
			log.Printf("Skipping %s: no migration directory found", dirName)
			continue
		}

		log.Printf("Processing schema for %s...", dirName)

		// * generate schema for this directory
		err := SchemaGenerate(migrationDir, dirName, polygonConfig.Dialect)
		if err != nil {
			log.Printf("Error generating schema for %s: %v", dirName, err)
			continue
		}

		log.Printf("Generated schema for %s", dirName)
	}

	return nil
}

func SchemaGenerate(migrationDir, dirName, dialect string) error {
	// * find all sql files in migration directory
	migrationFiles := make([]string, 0)
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		if migrations.IsDown(name) {
			continue
		}
		migrationFiles = append(migrationFiles, filepath.Join(migrationDir, name))
	}
	if len(migrationFiles) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationDir)
	}

	// * sort files for consistent order
	sort.Strings(migrationFiles)

	// * parse migrations and build schema state
	tables := make(map[string]*Table)
	functions := make(map[string]*Function)
	triggers := make(map[string]*Trigger)

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// * remove rollback statements
		cleanContent := migrations.RemoveRollbackStatements(string(content))

		// * parse cleaned migration
		ParseMigration(cleanContent, tables, functions, triggers)
	}

	// * generate final schema content
	var schemaContent strings.Builder
	schemaContent.WriteString("-- POLYGON GENERATED\n")
	schemaContent.WriteString("-- database schema: ")
	schemaContent.WriteString(dirName)
	schemaContent.WriteString("\n-- dialect: ")
	schemaContent.WriteString(dialect)
	schemaContent.WriteString("\n\n")

	// * write create table statements
	for _, tableName := range SortedTableKeys(tables) {
		table := tables[tableName]
		schemaContent.WriteString(table.GenerateStatement())
		schemaContent.WriteString("\n\n")
	}

	// * write create function statements
	for _, funcName := range SortedFunctionKeys(functions) {
		function := functions[funcName]
		schemaContent.WriteString(function.GenerateStatement())
		schemaContent.WriteString("\n\n")
	}

	// * write create trigger statements
	for _, triggerName := range SortedTriggerKeys(triggers) {
		trigger := triggers[triggerName]
		schemaContent.WriteString(trigger.GenerateStatement())
		schemaContent.WriteString("\n\n")
	}

	// * construct schema file path
	schemaDir := filepath.Join("polygon", "generate", "sequel")
	schemaFile := filepath.Join(schemaDir, fmt.Sprintf("%s.sql", dirName))

	// * ensure directory
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}

	err = os.WriteFile(schemaFile, []byte(schemaContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}
