package sequel

import (
	"fmt"
	"strings"

	"github.com/bsthun/gut"
	"go.scnd.dev/polygon/polygon/util"
)

func ParseMigration(content string, tables map[string]*Table, functions map[string]*Function, triggers map[string]*Trigger) {
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// * skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// * parse create table
		if strings.HasPrefix(strings.ToUpper(line), "CREATE TABLE") {
			table := ParseCreateTable(lines, &i)
			if table != nil {
				tables[*table.Name] = table
			}
			continue
		}

		// * parse create function
		if strings.HasPrefix(strings.ToUpper(line), "CREATE FUNCTION") {
			function := ParseCreateFunction(lines, &i)
			if function != nil {
				functions[*function.Name] = function
			}
			continue
		}

		// * parse create trigger
		if strings.HasPrefix(strings.ToUpper(line), "CREATE TRIGGER") {
			trigger := ParseCreateTrigger(lines, &i)
			if trigger != nil {
				triggers[*trigger.Name] = trigger
			}
			continue
		}

		// * parse alter table
		if strings.HasPrefix(strings.ToUpper(line), "ALTER TABLE") {
			ParseAlterTable(lines, &i, tables)
			continue
		}
	}
}

func ParseCreateTable(lines []string, index *int) *Table {
	line := strings.TrimSpace(lines[*index])

	// * extract table name
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil
	}

	tableName := strings.Trim(parts[2], `";`)
	singularName := util.ToSingular(tableName)
	table := &Table{
		Name:         &tableName,
		SingularName: &singularName,
		Columns:      nil,
		Indexes:      nil,
		Constraints:  nil,
	}

	// * parse table definition
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

	// * parse columns and constraints
	tableDefinition := definition.String()
	ParseTableDefinition(tableDefinition, table)

	return table
}

func ParseTableDefinition(definition string, table *Table) {
	// Remove CREATE TABLE name and outer parentheses
	definition = strings.TrimSpace(definition)
	startIndex := strings.Index(definition, "(")
	endIndex := strings.LastIndex(definition, ")")

	if startIndex == -1 || endIndex == -1 {
		return
	}

	content := definition[startIndex+1 : endIndex]

	// More robust splitting that handles comma-separated items properly
	items := SplitTableItems(content)

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		upperItem := strings.ToUpper(item)

		// * parse standalone constraints
		if strings.HasPrefix(upperItem, "PRIMARY KEY") {
			// * extract columns for primary key
			constraint := &Constraint{
				Type: gut.Ptr("PRIMARY KEY"),
			}
			// * parse columns from `PRIMARY KEY`
			startParen := strings.Index(item, "(")
			endParen := strings.LastIndex(item, ")")
			if startParen != -1 && endParen != -1 && endParen > startParen+1 {
				columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
				if columnsStr != "" {
					columns := strings.Split(columnsStr, ",")
					for _, column := range columns {
						column = strings.TrimSpace(column)
						constraint.Columns = append(constraint.Columns, &column)
					}
				}
			}
			table.Constraints = append(table.Constraints, constraint)
			continue
		}

		if strings.HasPrefix(upperItem, "FOREIGN KEY") {
			// * parse foreign key constraint
			constraint := &Constraint{
				Type: gut.Ptr("FOREIGN KEY"),
			}
			// * extract columns from `FOREIGN KEY`
			startParen := strings.Index(item, "(")
			endParen := strings.Index(item, ")")
			if startParen != -1 && endParen != -1 {
				columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
				columns := strings.Split(columnsStr, ",")
				for _, column := range columns {
					column = strings.TrimSpace(column)
					constraint.Columns = append(constraint.Columns, &column)
				}
			}
			// * extract referenced table
			if refIndex := strings.Index(strings.ToUpper(item), "REFERENCES"); refIndex != -1 {
				refPart := strings.TrimSpace(item[refIndex+len("REFERENCES"):])
				// Handle format: table (column)
				if refParenIndex := strings.Index(refPart, "("); refParenIndex != -1 {
					tableName := strings.TrimSpace(refPart[:refParenIndex])
					// * extract column if specified
					endRefParen := strings.Index(refPart[refParenIndex:], ")")
					if endRefParen != -1 {
						columnName := strings.TrimSpace(refPart[refParenIndex+1 : refParenIndex+endRefParen])
						constraint.References = gut.Ptr(fmt.Sprintf("%s (%s)", tableName, columnName))
					} else {
						constraint.References = &tableName
					}
				} else {
					refParts := strings.Fields(refPart)
					if len(refParts) > 0 {
						constraint.References = &refParts[0]
					}
				}
			}
			table.Constraints = append(table.Constraints, constraint)
			continue
		}

		if strings.HasPrefix(upperItem, "UNIQUE") {
			// * parse unique constraint
			constraint := &Constraint{
				Type: gut.Ptr("UNIQUE"),
			}
			// * extract columns from `UNIQUE`
			startParen := strings.Index(item, "(")
			endParen := strings.Index(item, ")")
			if startParen != -1 && endParen != -1 {
				columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
				columns := strings.Split(columnsStr, ",")
				for _, column := range columns {
					column = strings.TrimSpace(column)
					constraint.Columns = append(constraint.Columns, &column)
				}
			}
			table.Constraints = append(table.Constraints, constraint)
			continue
		}

		if strings.HasPrefix(upperItem, "CONSTRAINT") {
			// * parse named constraint
			if pkeyIndex := strings.Index(strings.ToUpper(item), "PRIMARY KEY"); pkeyIndex != -1 {
				constraint := &Constraint{
					Type: gut.Ptr("PRIMARY KEY"),
				}
				// * extract constraint name
				parts := strings.Fields(item)
				if len(parts) > 1 {
					constraint.Name = gut.Ptr(strings.Trim(parts[1], `"`))
				}
				// * extract columns
				startParen := strings.Index(item, "(")
				endParen := strings.Index(item, ")")
				if startParen != -1 && endParen != -1 {
					columnsStr := strings.TrimSpace(item[startParen+1 : endParen])
					columns := strings.Split(columnsStr, ",")
					for _, column := range columns {
						column = strings.TrimSpace(column)
						constraint.Columns = append(constraint.Columns, &column)
					}
				}
				table.Constraints = append(table.Constraints, constraint)
			}
			continue
		}

		// * parse column with inline constraints
		parts := strings.Fields(item)
		if len(parts) >= 2 {
			column := &Column{
				Name:     gut.Ptr(strings.Trim(parts[0], `";`)),
				Type:     gut.Ptr(parts[1]),
				Nullable: gut.Ptr(true),
			}

			// * parse column attributes
			for j := 2; j < len(parts); j++ {
				attr := strings.ToUpper(parts[j])
				if attr == "NOT" && j+1 < len(parts) && strings.ToUpper(parts[j+1]) == "NULL" {
					column.Nullable = gut.Ptr(false)
					j++
				} else if attr == "DEFAULT" && j+1 < len(parts) {
					column.Default = gut.Ptr(parts[j+1])
					j++
				} else if attr == "PRIMARY" && j+1 < len(parts) && strings.ToUpper(parts[j+1]) == "KEY" {
					// * handle inline `PRIMARY KEY`
					column.Nullable = gut.Ptr(false)
					// Add as single-column primary key constraint
					constraint := &Constraint{
						Type: gut.Ptr("PRIMARY KEY"),
						Columns: []*string{
							column.Name,
						},
					}
					table.Constraints = append(table.Constraints, constraint)
					j++
				} else if attr == "UNIQUE" {
					// * handle inline `UNIQUE`
					constraint := &Constraint{
						Type: gut.Ptr("UNIQUE"),
						Columns: []*string{
							column.Name,
						},
					}
					table.Constraints = append(table.Constraints, constraint)
				} else if attr == "REFERENCES" && j+1 < len(parts) {
					// * handle inline foreign key
					constraint := &Constraint{
						Type: gut.Ptr("FOREIGN KEY"),
						Columns: []*string{
							column.Name,
						},
						References: &parts[j+1],
					}
					table.Constraints = append(table.Constraints, constraint)
					j++
				}
			}

			table.Columns = append(table.Columns, column)
		}
	}
}

func ParseCreateFunction(lines []string, index *int) *Function {
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
		Name: gut.Ptr("function"),
		Body: &functionText,
	}
}

func ParseCreateTrigger(lines []string, index *int) *Trigger {
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
		Name:     gut.Ptr("trigger"),
		Function: &triggerText,
	}
}

func ParseAlterTable(lines []string, index *int, tables map[string]*Table) {
	// * parse alter table statements
	line := strings.TrimSpace(lines[*index])

	// * extract table name from `ALTER TABLE`
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

		// * parse `ALTER TABLE` statements
		if *index+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[*index+1])

			// * handle drop column
			if strings.HasPrefix(strings.ToUpper(nextLine), "DROP COLUMN") {
				dropParts := strings.Fields(nextLine)
				if len(dropParts) >= 3 {
					columnName := strings.Trim(dropParts[2], `";`)
					// * remove `IF EXISTS` if present
					if len(dropParts) >= 4 && strings.ToUpper(dropParts[2]) == "IF" && strings.ToUpper(dropParts[3]) == "EXISTS" {
						if len(dropParts) >= 5 {
							columnName = strings.Trim(strings.TrimSuffix(dropParts[4], ";"), `";`)
						}
					} else {
						columnName = strings.Trim(strings.TrimSuffix(columnName, ";"), `";`)
					}
					// * remove column from table
					for i, column := range table.Columns {
						if *column.Name == columnName {
							table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
							break
						}
					}
				}
				*index += 1
			}

			// * handle alter column type
			if strings.HasPrefix(strings.ToUpper(nextLine), "ALTER COLUMN") {
				alterParts := strings.Fields(nextLine)
				if len(alterParts) >= 4 {
					columnName := strings.Trim(alterParts[2], `";`)
					// * look for `TYPE`
					if len(alterParts) > 3 && strings.ToUpper(alterParts[3]) == "TYPE" && len(alterParts) > 4 {
						newType := strings.TrimSuffix(alterParts[4], ";")
						// Update the column type
						for i, column := range table.Columns {
							if *column.Name == columnName {
								table.Columns[i].Type = &newType
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
