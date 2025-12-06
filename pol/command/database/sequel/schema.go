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

type PolygonConfig struct {
	Dir     string `yaml:"dir"`
	Dialect string `yaml:"dialect"`
}

// Schema objects
type Table struct {
	Name        string
	Columns     []Column
	Indexes     []Index
	Constraints []Constraint
}

type Column struct {
	Name        string
	Type        string
	Nullable    bool
	Default     string
	Constraints []string
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Type    string
}

type Constraint struct {
	Name       string
	Type       string
	Columns    []string
	References string
}

type Function struct {
	Name       string
	Parameters []string
	Returns    string
	Body       string
	Language   string
}

type Trigger struct {
	Name       string
	Table      string
	Before     bool
	After      bool
	InsteadOf  bool
	Events     []string
	Function   string
	ForEachRow bool
}

// Helper functions for parsing SQL migrations using sqlc
func removeRollbackStatements(contents string) string {
	return migrations.RemoveRollbackStatements(contents)
}

func isDown(filename string) bool {
	return migrations.IsDown(filename)
}

func parseMigration(content string, tables map[string]*Table, functions map[string]*Function, triggers map[string]*Trigger) {
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// Parse CREATE TABLE
		if strings.HasPrefix(strings.ToUpper(line), "CREATE TABLE") {
			table := parseCreateTable(lines, &i)
			if table != nil {
				tables[table.Name] = table
			}
			continue
		}

		// Parse CREATE FUNCTION
		if strings.HasPrefix(strings.ToUpper(line), "CREATE FUNCTION") {
			function := parseCreateFunction(lines, &i)
			if function != nil {
				functions[function.Name] = function
			}
			continue
		}

		// Parse CREATE TRIGGER
		if strings.HasPrefix(strings.ToUpper(line), "CREATE TRIGGER") {
			trigger := parseCreateTrigger(lines, &i)
			if trigger != nil {
				triggers[trigger.Name] = trigger
			}
			continue
		}

		// Parse ALTER TABLE
		if strings.HasPrefix(strings.ToUpper(line), "ALTER TABLE") {
			parseAlterTable(lines, &i, tables)
			continue
		}
	}
}

func parseCreateTable(lines []string, index *int) *Table {
	line := strings.TrimSpace(lines[*index])

	// Extract table name
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil
	}

	tableName := strings.Trim(parts[2], `";`)
	table := &Table{
		Name: tableName,
	}

	// Parse table definition
	var definition strings.Builder
	inDefinition := false
	parenCount := 0

	for *index < len(lines) {
		currentLine := strings.TrimSpace(lines[*index])

		if strings.Contains(currentLine, "(") {
			inDefinition = true
		}

		if inDefinition {
			definition.WriteString(currentLine)
			definition.WriteString(" ")

			for _, char := range currentLine {
				if char == '(' {
					parenCount++
				} else if char == ')' {
					parenCount--
				}
			}

			if parenCount == 0 {
				break
			}
		}

		*index++
	}

	// Parse columns and constraints from definition
	tableDefinition := definition.String()
	parseTableDefinition(tableDefinition, table)

	return table
}

func parseTableDefinition(definition string, table *Table) {
	// Remove CREATE TABLE name and outer parentheses
	definition = strings.TrimSpace(definition)
	startIndex := strings.Index(definition, "(")
	endIndex := strings.LastIndex(definition, ")")

	if startIndex == -1 || endIndex == -1 {
		return
	}

	content := definition[startIndex+1 : endIndex]

	// More robust splitting that handles comma-separated items properly
	// Split by comma while respecting parentheses (for function calls, etc.)
	items := splitTableItems(content)

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		upperItem := strings.ToUpper(item)

		// Parse standalone constraints
		if strings.HasPrefix(upperItem, "PRIMARY KEY") {
			// Extract columns for multi-column primary key
			constraint := Constraint{
				Type: "PRIMARY KEY",
			}
			// Parse columns from PRIMARY KEY (col1, col2)
			startParen := strings.Index(item, "(")
			endParen := strings.LastIndex(item, ")")
			if startParen != -1 && endParen != -1 && endParen > startParen+1 {
				columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
				if columnsStr != "" {
					columns := strings.Split(columnsStr, ",")
					for _, col := range columns {
						constraint.Columns = append(constraint.Columns, strings.TrimSpace(col))
					}
				}
			}
			table.Constraints = append(table.Constraints, constraint)
			continue
		}

		if strings.HasPrefix(upperItem, "FOREIGN KEY") {
			// Parse foreign key constraint
			constraint := Constraint{
				Type: "FOREIGN KEY",
			}
			// Extract columns from FOREIGN KEY (col1, col2)
			startParen := strings.Index(item, "(")
			endParen := strings.Index(item, ")")
			if startParen != -1 && endParen != -1 {
				columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
				columns := strings.Split(columnsStr, ",")
				for _, col := range columns {
					constraint.Columns = append(constraint.Columns, strings.TrimSpace(col))
				}
			}
			// Extract referenced table and column
			if refIndex := strings.Index(strings.ToUpper(item), "REFERENCES"); refIndex != -1 {
				refPart := strings.TrimSpace(item[refIndex+len("REFERENCES"):])
				// Handle format: table (column)
				if refParenIndex := strings.Index(refPart, "("); refParenIndex != -1 {
					tableName := strings.TrimSpace(refPart[:refParenIndex])
					// Extract column if specified
					endRefParen := strings.Index(refPart[refParenIndex:], ")")
					if endRefParen != -1 {
						columnName := strings.TrimSpace(refPart[refParenIndex+1 : refParenIndex+endRefParen])
						constraint.References = tableName + " (" + columnName + ")"
					} else {
						constraint.References = tableName
					}
				} else {
					refParts := strings.Fields(refPart)
					if len(refParts) > 0 {
						constraint.References = refParts[0]
					}
				}
			}
			table.Constraints = append(table.Constraints, constraint)
			continue
		}

		if strings.HasPrefix(upperItem, "UNIQUE") {
			// Parse unique constraint
			constraint := Constraint{
				Type: "UNIQUE",
			}
			// Extract columns from UNIQUE (col1, col2)
			startParen := strings.Index(item, "(")
			endParen := strings.Index(item, ")")
			if startParen != -1 && endParen != -1 {
				columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
				columns := strings.Split(columnsStr, ",")
				for _, col := range columns {
					constraint.Columns = append(constraint.Columns, strings.TrimSpace(col))
				}
			}
			table.Constraints = append(table.Constraints, constraint)
			continue
		}

		if strings.HasPrefix(upperItem, "CONSTRAINT") {
			// Parse named constraint - extract the actual constraint type
			if pkeyIndex := strings.Index(strings.ToUpper(item), "PRIMARY KEY"); pkeyIndex != -1 {
				constraint := Constraint{
					Type: "PRIMARY KEY",
				}
				// Extract constraint name
				parts := strings.Fields(item)
				if len(parts) > 1 {
					constraint.Name = strings.Trim(parts[1], `"`)
				}
				// Extract columns
				startParen := strings.Index(item, "(")
				endParen := strings.Index(item, ")")
				if startParen != -1 && endParen != -1 {
					columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
					columns := strings.Split(columnsStr, ",")
					for _, col := range columns {
						constraint.Columns = append(constraint.Columns, strings.TrimSpace(col))
					}
				}
				table.Constraints = append(table.Constraints, constraint)
			}
			continue
		}

		// Parse column with potential inline constraints
		parts := strings.Fields(item)
		if len(parts) >= 2 {
			column := Column{
				Name:     strings.Trim(parts[0], `";`),
				Type:     parts[1],
				Nullable: true,
			}

			// Parse column attributes including inline constraints
			for j := 2; j < len(parts); j++ {
				attr := strings.ToUpper(parts[j])
				if attr == "NOT" && j+1 < len(parts) && strings.ToUpper(parts[j+1]) == "NULL" {
					column.Nullable = false
					j++
				} else if attr == "DEFAULT" && j+1 < len(parts) {
					column.Default = parts[j+1]
					j++
				} else if attr == "PRIMARY" && j+1 < len(parts) && strings.ToUpper(parts[j+1]) == "KEY" {
					// Handle inline PRIMARY KEY constraint
					column.Nullable = false
					// Add as single-column primary key constraint
					constraint := Constraint{
						Type:    "PRIMARY KEY",
						Columns: []string{column.Name},
					}
					table.Constraints = append(table.Constraints, constraint)
					j++
				} else if attr == "UNIQUE" {
					// Handle inline UNIQUE constraint
					constraint := Constraint{
						Type:    "UNIQUE",
						Columns: []string{column.Name},
					}
					table.Constraints = append(table.Constraints, constraint)
				} else if attr == "REFERENCES" && j+1 < len(parts) {
					// Handle inline foreign key reference
					constraint := Constraint{
						Type:       "FOREIGN KEY",
						Columns:    []string{column.Name},
						References: parts[j+1],
					}
					table.Constraints = append(table.Constraints, constraint)
					j++
				}
			}

			table.Columns = append(table.Columns, column)
		}
	}
}

func parseCreateFunction(lines []string, index *int) *Function {
	// Simple function parser - collect all lines until semicolon
	var functionLines []string

	for *index < len(lines) {
		line := lines[*index]
		functionLines = append(functionLines, line)

		if strings.Contains(line, ";") {
			break
		}
		*index++
	}

	functionText := strings.Join(functionLines, " ")
	return &Function{
		Name: "function", // Simplified - would need proper parsing
		Body: functionText,
	}
}

func parseCreateTrigger(lines []string, index *int) *Trigger {
	// Simple trigger parser - collect all lines until semicolon
	var triggerLines []string

	for *index < len(lines) {
		line := lines[*index]
		triggerLines = append(triggerLines, line)

		if strings.Contains(line, ";") {
			break
		}
		*index++
	}

	triggerText := strings.Join(triggerLines, " ")
	return &Trigger{
		Name:     "trigger", // Simplified - would need proper parsing
		Function: triggerText,
	}
}

func parseAlterTable(lines []string, index *int, tables map[string]*Table) {
	// Parse ALTER TABLE statements - handle the specific format in our migration files
	line := strings.TrimSpace(lines[*index])

	// Extract table name from ALTER TABLE line
	if strings.HasPrefix(strings.ToUpper(line), "ALTER TABLE") {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			return
		}
		tableName := strings.Trim(parts[2], `";`)

		table, exists := tables[tableName]
		if !exists {
			return
		}

		// Look ahead for the next lines that contain the ALTER commands
		if *index+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[*index+1])

			// Handle DROP COLUMN
			if strings.HasPrefix(strings.ToUpper(nextLine), "DROP COLUMN") {
				dropParts := strings.Fields(nextLine)
				if len(dropParts) >= 3 {
					columnName := strings.Trim(dropParts[2], `";`)
					// Remove "IF EXISTS" if present
					if len(dropParts) >= 4 && strings.ToUpper(dropParts[2]) == "IF" && strings.ToUpper(dropParts[3]) == "EXISTS" {
						if len(dropParts) >= 5 {
							columnName = strings.Trim(strings.TrimSuffix(dropParts[4], ";"), `";`)
						}
					} else {
						columnName = strings.Trim(strings.TrimSuffix(columnName, ";"), `";`)
					}
					// Remove the column from the table
					for i, col := range table.Columns {
						if col.Name == columnName {
							table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
							break
						}
					}
				}
				*index += 1
			}

			// Handle ALTER COLUMN TYPE
			if strings.HasPrefix(strings.ToUpper(nextLine), "ALTER COLUMN") {
				alterParts := strings.Fields(nextLine)
				if len(alterParts) >= 4 {
					columnName := strings.Trim(alterParts[2], `";`)
					// Look for TYPE (should be at position 4)
					if len(alterParts) > 3 && strings.ToUpper(alterParts[3]) == "TYPE" && len(alterParts) > 4 {
						newType := strings.TrimSuffix(alterParts[4], ";")
						// Update the column type
						for i, col := range table.Columns {
							if col.Name == columnName {
								table.Columns[i].Type = newType
								break
							}
						}
					}
				}
				*index += 1
			}
		}
	}
}

func getSortedTableKeys(m map[string]*Table) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getSortedFunctionKeys(m map[string]*Function) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getSortedTriggerKeys(m map[string]*Trigger) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Helper function to split table items by comma while respecting parentheses
func splitTableItems(content string) []string {
	var items []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range content {
		if char == '(' {
			parenDepth++
			current.WriteRune(char)
		} else if char == ')' {
			parenDepth--
			current.WriteRune(char)
		} else if char == ',' && parenDepth == 0 {
			// Split here only if we're not inside parentheses
			items = append(items, current.String())
			current.Reset()
		} else {
			current.WriteRune(char)
		}
	}

	// Add the last item
	if current.Len() > 0 {
		items = append(items, current.String())
	}

	return items
}

// Generate CREATE statements
func (t *Table) GenerateCreateStatement() string {
	var stmt strings.Builder
	stmt.WriteString("CREATE TABLE ")
	stmt.WriteString(t.Name)
	stmt.WriteString(" (\n")

	// Track which constraints have been processed inline
	inlineProcessed := make(map[string]bool)

	// Collect remaining constraints that need to be added at table level
	var remainingConstraints []Constraint

	// Write columns with inline constraints where appropriate
	for i, column := range t.Columns {
		stmt.WriteString("    ")
		stmt.WriteString(column.Name)
		stmt.WriteString(" ")
		stmt.WriteString(column.Type)

		if !column.Nullable {
			stmt.WriteString(" NOT NULL")
		} else {
			stmt.WriteString(" NULL")
		}

		if column.Default != "" {
			stmt.WriteString(" DEFAULT ")
			stmt.WriteString(column.Default)
		}

		// Check for single-column constraints that can be inlined
		for _, constraint := range t.Constraints {
			if len(constraint.Columns) == 1 && constraint.Columns[0] == column.Name {
				constraintKey := constraint.Type + "_" + column.Name
				if constraint.Type == "UNIQUE" {
					stmt.WriteString(" UNIQUE")
					inlineProcessed[constraintKey] = true
				} else if constraint.Type == "FOREIGN KEY" && constraint.References != "" {
					stmt.WriteString(" REFERENCES ")
					stmt.WriteString(constraint.References)
					inlineProcessed[constraintKey] = true
				}
				// For PRIMARY KEY, we'll add it at the table level for consistency
			}
		}

		// Add comma if this is not the last column or if there are constraints
		if i < len(t.Columns)-1 || len(t.Constraints) > 0 {
			stmt.WriteString(",")
		}
		stmt.WriteString("\n")
	}

	// Collect remaining constraints that weren't inlined
	for _, constraint := range t.Constraints {
		constraintKey := ""
		if len(constraint.Columns) == 1 {
			constraintKey = constraint.Type + "_" + constraint.Columns[0]
		}

		// Skip constraints that were already inlined
		if !inlineProcessed[constraintKey] {
			remainingConstraints = append(remainingConstraints, constraint)
		}
	}

	// Write remaining table-level constraints
	for i, constraint := range remainingConstraints {
		switch constraint.Type {
		case "PRIMARY KEY":
			stmt.WriteString("    PRIMARY KEY (")
			for j, col := range constraint.Columns {
				if j > 0 {
					stmt.WriteString(", ")
				}
				stmt.WriteString(col)
			}
			stmt.WriteString(")")
		case "FOREIGN KEY":
			stmt.WriteString("    FOREIGN KEY (")
			for j, col := range constraint.Columns {
				if j > 0 {
					stmt.WriteString(", ")
				}
				stmt.WriteString(col)
			}
			stmt.WriteString(") REFERENCES ")
			stmt.WriteString(constraint.References)
		case "UNIQUE":
			stmt.WriteString("    UNIQUE (")
			for j, col := range constraint.Columns {
				if j > 0 {
					stmt.WriteString(", ")
				}
				stmt.WriteString(col)
			}
			stmt.WriteString(")")
		}

		// Add comma between constraints, but not after the last one
		if i < len(remainingConstraints)-1 {
			stmt.WriteString(",")
		}
		stmt.WriteString("\n")
	}

	stmt.WriteString(");")
	return stmt.String()
}

func (f *Function) GenerateCreateStatement() string {
	return f.Body
}

func (t *Trigger) GenerateCreateStatement() string {
	return t.Function
}

// GenerateSchemas reads the polygon.yml configuration and generates schema files
// for all directories in ./sequel
func GenerateSchemas() error {
	// Read polygon.yml to get dialect
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

	// Validate dialect
	switch polygonConfig.Dialect {
	case "postgres", "postgresql", "mysql", "sqlite":
		// Supported dialects
	default:
		return fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", polygonConfig.Dialect)
	}

	// Find all directories in ./sequel
	sequelDir := filepath.Join("sequel")
	entries, err := os.ReadDir(sequelDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("sequel directory not found: %s", sequelDir)
		}
		return fmt.Errorf("failed to read sequel directory: %w", err)
	}

	// Process each sequel directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		migrationDir := filepath.Join(sequelDir, dirName, "migration")

		// Check if migration directory exists
		if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
			log.Printf("Skipping %s: no migration directory found", dirName)
			continue
		}

		log.Printf("Processing schema for %s...", dirName)

		// Generate schema for this directory
		err := generateSchemaForDir(migrationDir, dirName, polygonConfig.Dialect)
		if err != nil {
			log.Printf("Error generating schema for %s: %v", dirName, err)
			continue
		}

		log.Printf("Generated schema for %s", dirName)
	}

	return nil
}

func generateSchemaForDir(migrationDir, dirName, dialect string) error {
	// Find all SQL files in the migration directory
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
		if isDown(name) {
			continue
		}
		migrationFiles = append(migrationFiles, filepath.Join(migrationDir, name))
	}
	if len(migrationFiles) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationDir)
	}

	// Sort files to ensure consistent order
	sort.Strings(migrationFiles)

	// Parse all migrations and build final schema state using improved logic
	tables := make(map[string]*Table)
	functions := make(map[string]*Function)
	triggers := make(map[string]*Trigger)

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// Remove rollback statements using sqlc's migration utilities
		cleanContent := removeRollbackStatements(string(content))

		// Parse the cleaned migration using existing logic
		parseMigration(cleanContent, tables, functions, triggers)
	}

	// Generate final schema content
	var schemaContent strings.Builder
	schemaContent.WriteString("-- Generated final database schema for ")
	schemaContent.WriteString(dirName)
	schemaContent.WriteString("\n-- Engine: ")
	schemaContent.WriteString(dialect)
	schemaContent.WriteString("\n\n")

	// Write CREATE TABLE statements first
	for _, tableName := range getSortedTableKeys(tables) {
		table := tables[tableName]
		schemaContent.WriteString(table.GenerateCreateStatement())
		schemaContent.WriteString("\n\n")
	}

	// Write CREATE FUNCTION statements
	for _, funcName := range getSortedFunctionKeys(functions) {
		function := functions[funcName]
		schemaContent.WriteString(function.GenerateCreateStatement())
		schemaContent.WriteString("\n\n")
	}

	// Write CREATE TRIGGER statements
	for _, triggerName := range getSortedTriggerKeys(triggers) {
		trigger := triggers[triggerName]
		schemaContent.WriteString(trigger.GenerateCreateStatement())
		schemaContent.WriteString("\n\n")
	}

	// Write schema.sql to the root of the sequel directory
	schemaPath := filepath.Join("sequel", dirName, "schema.sql")
	err = os.WriteFile(schemaPath, []byte(schemaContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}
