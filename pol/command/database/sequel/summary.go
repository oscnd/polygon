package sequel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Summary(parser *Parser, dirName string) error {
	// * get connection for this directory
	connection, exists := parser.Connections[dirName]
	if !exists {
		return fmt.Errorf("connection not found for directory: %s", dirName)
	}

	// * generate final schema content
	var schemaContent strings.Builder
	schemaContent.WriteString("-- POLYGON GENERATED\n")
	schemaContent.WriteString("-- database schema: ")
	schemaContent.WriteString(dirName)
	schemaContent.WriteString("\n-- dialect: ")
	schemaContent.WriteString(*connection.Dialect)
	schemaContent.WriteString("\n\n")

	functions := make(map[string]*Function)
	triggers := make(map[string]*Trigger)

	// * write create table statements for this connection only
	for _, tableName := range SortedTableKeys(connection.Tables) {
		table := connection.Tables[tableName]
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
	schemaDir := filepath.Join("generate", "polygon", "sequel")
	schemaFile := filepath.Join(schemaDir, fmt.Sprintf("%s.sql", dirName))

	// * ensure directory
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}

	err := os.WriteFile(schemaFile, []byte(schemaContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	if *parser.App.Verbose() {
		fmt.Printf("Generated schema for %s\n", dirName)
	}

	return nil
}
