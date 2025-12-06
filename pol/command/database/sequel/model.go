package sequel

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/polygon/external/sqlc/config"
	"go.scnd.dev/polygon/external/sqlc/engine/postgresql"
	"go.scnd.dev/polygon/external/sqlc/migrations"
	"go.scnd.dev/polygon/external/sqlc/sql/catalog"
	"go.scnd.dev/polygon/polygon/util"
)

func Model(migrationFiles []string, dirName, dialect string, sqlc config.Config) error {
	// * create sqlc catalog
	cat := catalog.New("public")

	// * parse all migration files to build our own table structure for relationships
	tables := make(map[string]*Table)

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// * remove rollback statements
		cleanContent := migrations.RemoveRollbackStatements(string(content))

		// * parse SQL using postgresql parser
		parser := postgresql.NewParser()
		stmts, err := parser.Parse(strings.NewReader(cleanContent))
		if err != nil {
			return fmt.Errorf("failed to parse SQL in %s: %w", file, err)
		}

		// * build catalog
		for _, stmt := range stmts {
			if err := cat.Update(stmt, nil); err != nil {
				log.Printf("Warning: failed to update catalog with statement from %s: %v", file, err)
			}
		}

		// * parse with our sequel parser to get constraint information
		ParseMigration(cleanContent, tables, make(map[string]*Function), make(map[string]*Trigger))
	}

	// * generate Go structs for each table
	for _, schema := range cat.Schemas {
		for _, table := range schema.Tables {
			if err := ModelGenerate(table, dirName, tables, sqlc); err != nil {
				return fmt.Errorf("failed to generate model for table %s: %w", table.Rel.Name, err)
			}
		}
	}

	return nil
}

func ModelGenerate(table *catalog.Table, dirName string, tables map[string]*Table, sqlc config.Config) error {
	// * construct model file paths using singular table name
	singularTableName := util.ToSingular(table.Rel.Name)
	generatedModelDir := filepath.Join("polygon", "model")
	generatedModelFile := filepath.Join(generatedModelDir, fmt.Sprintf("%s.%s.go", dirName, singularTableName))

	// * ensure output directory exists
	if err := os.MkdirAll(generatedModelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// * read existing file if it exists
	existingContent := ""
	if _, err := os.Stat(generatedModelFile); err == nil {
		content, err := os.ReadFile(generatedModelFile)
		if err != nil {
			return fmt.Errorf("failed to read existing model file: %w", err)
		}
		existingContent = string(content)
	}

	// * collect required imports from overrides - only include imports used by this table
	var requiredImports []string
	tableColumnPattern := fmt.Sprintf("%s.", table.Rel.Name) // e.g., "users."

	if len(sqlc.SQL) > 0 {
		for _, sql := range sqlc.SQL {
			if sql.Gen.Go != nil && sql.Gen.Go.Overrides != nil {
				for _, override := range sql.Gen.Go.Overrides {
					// * only add import if this override applies to this table
					if override.GoType.Path != "" &&
						!strings.HasPrefix(override.GoType.Path, "database/sql") &&
						strings.HasPrefix(override.Column, tableColumnPattern) {
						requiredImports = append(requiredImports, override.GoType.Path)
					}
				}
			}
		}
	}

	// * parse existing structs
	existingStructs := ModelParseExistingStructs(existingContent)

	// * generate struct name in title case (singular form)
	structName := util.ToSingularTitleCase(table.Rel.Name)

	// * generate main struct
	mainStruct := ModelGenerateMainStruct(structName, table, &sqlc)

	// * generate or get addition/contraction structs
	additionStruct := ModelGetOrCreateStruct(existingStructs, structName+"Addition", true)
	contractionStruct := ModelGetOrCreateStruct(existingStructs, structName+"Contraction", true)

	// * generate ModelAdded struct using reflection
	addedStruct := ModelGenerateAdded(structName, table, additionStruct, contractionStruct, &sqlc)

	// * generate ModelAdded struct first to use as base for Joined/Parented
	modelAddedBase := ModelGenerateAdded(structName, table, additionStruct, contractionStruct, &sqlc)

	// * generate ModelJoined struct with references
	joinedStruct := ModelGenerateJoined(structName, table, modelAddedBase, tables)

	// * generate ModelParented struct with parent references
	parentedStruct := ModelGenerateParented(structName, table, modelAddedBase, tables)

	// * organize structs in required order
	orderedStructs := ModelOrganizeStructs(structName, mainStruct, additionStruct, contractionStruct, addedStruct, joinedStruct, parentedStruct, existingStructs)

	// * write final file content with imports
	finalContent := ModelGenerateFileContent(orderedStructs, requiredImports)

	err := os.WriteFile(generatedModelFile, []byte(finalContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	return nil
}

func ModelParseExistingStructs(content string) map[string]string {
	structs := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentStruct strings.Builder
	var currentName string
	inStruct := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, "struct") {
			// * start of a struct
			if inStruct && currentName != "" {
				structs[currentName] = currentStruct.String()
			}

			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				currentName = parts[1]
			}
			currentStruct.Reset()
			currentStruct.WriteString(line + "\n")
			inStruct = true
		} else if inStruct && trimmed == "}" {
			// * end of current struct
			currentStruct.WriteString(line + "\n")
			if currentName != "" {
				structs[currentName] = currentStruct.String()
			}
			inStruct = false
			currentName = ""
		} else if inStruct {
			currentStruct.WriteString(line + "\n")
		}
	}

	// * catch case where file doesn't end with }
	if inStruct && currentName != "" {
		structs[currentName] = currentStruct.String()
	}

	return structs
}

func ModelGenerateMainStruct(name string, table *catalog.Table, sqlcConfig *config.Config) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, col := range table.Columns {
		// * convert SQL type to Go type
		goType := ModelSqlTypeToGoType(col.Type.Name, col.IsNotNull, col.Name, table.Rel.Name, sqlcConfig)

		// * generate json tag
		jsonTag := util.ToCamelCase(col.Name)

		// * generate field
		if col.IsNotNull {
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\" validate:\"required\"`\n", util.ToTitleCase(col.Name), goType, jsonTag))
		} else {
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n", util.ToTitleCase(col.Name), goType, jsonTag))
		}
	}

	builder.WriteString("}\n")
	return builder.String()
}

func ModelSqlTypeToGoType(sqlType string, notNull bool, columnName string, tableName string, sqlcConfig *config.Config) string {
	sqlType = strings.ToLower(sqlType)

	// * check for sqlc overrides first
	if sqlcConfig != nil {
		for _, sql := range sqlcConfig.SQL {
			if sql.Gen.Go != nil && sql.Gen.Go.Overrides != nil {
				for _, override := range sql.Gen.Go.Overrides {
					// * check column-specific overrides (format: "table.column" or "schema.table.column")
					if override.Column != "" {
						// * try exact match: table.column
						columnPattern := fmt.Sprintf("%s.%s", tableName, columnName)
						if override.Column == columnPattern {
							// * build qualified type name with package prefix
							goType := override.GoType.Name
							if override.GoType.Path != "" {
								// * extract package name from import path (last segment)
								pathParts := strings.Split(override.GoType.Path, "/")
								if len(pathParts) > 0 {
									packageName := pathParts[len(pathParts)-1]
									goType = packageName + "." + goType
								}
							} else if override.GoType.Package != "" {
								goType = override.GoType.Package + "." + goType
							}
							if override.GoType.Pointer {
								goType = "*" + goType
							}
							return goType
						}

						// * try schema-qualified match: public.table.column
						schemaPattern := fmt.Sprintf("public.%s.%s", tableName, columnName)
						if override.Column == schemaPattern {
							// * build qualified type name with package prefix
							goType := override.GoType.Name
							if override.GoType.Path != "" {
								// * extract package name from import path (last segment)
								pathParts := strings.Split(override.GoType.Path, "/")
								if len(pathParts) > 0 {
									packageName := pathParts[len(pathParts)-1]
									goType = packageName + "." + goType
								}
							} else if override.GoType.Package != "" {
								goType = override.GoType.Package + "." + goType
							}
							if override.GoType.Pointer {
								goType = "*" + goType
							}
							return goType
						}
					}

					// * check for type-specific overrides (db_type)
					if override.DBType == sqlType && override.Column == "" {
						// * build qualified type name with package prefix
						goType := override.GoType.Name
						if override.GoType.Package != "" {
							goType = override.GoType.Package + "." + goType
						}
						if override.GoType.Pointer {
							goType = "*" + goType
						}
						return goType
					}
				}
			}
		}
	}

	// * fallback to default type mapping
	switch {
	case strings.Contains(sqlType, "int"):
		if strings.HasSuffix(strings.ToLower(columnName), "id") {
			return "*uint64"
		}
		return "*int64"
	case strings.Contains(sqlType, "varchar"), strings.Contains(sqlType, "text"), strings.Contains(sqlType, "char"):
		return "*string"
	case strings.Contains(sqlType, "boolean"), sqlType == "bool":
		return "*bool"
	case strings.Contains(sqlType, "timestamp"), strings.Contains(sqlType, "date"), strings.Contains(sqlType, "time"):
		return "*time.Time"
	case strings.Contains(sqlType, "decimal"), strings.Contains(sqlType, "numeric"), strings.Contains(sqlType, "float"), strings.Contains(sqlType, "double"):
		return "*float64"
	case strings.Contains(sqlType, "json"), strings.Contains(sqlType, "jsonb"):
		return "any"
	case strings.Contains(sqlType, "uuid"):
		return "*string"
	default:
		return "*string"
	}
}

func ModelGetOrCreateStruct(existingStructs map[string]string, name string, createIfMissing bool) string {
	if content, exists := existingStructs[name]; exists {
		return content
	}

	if createIfMissing {
		return fmt.Sprintf("type %s struct {\n}\n", name)
	}

	return ""
}

func ModelGenerateAdded(baseName string, table *catalog.Table, additionStruct, contractionStruct string, sqlcConfig *config.Config) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %sAdded struct {\n", baseName))

	// * parse addition and contraction structs to get fields
	additionFields := ModelParseStructFields(additionStruct)
	contractionFields := ModelParseStructFields(contractionStruct)

	// * add main struct fields
	for _, col := range table.Columns {
		goType := ModelSqlTypeToGoType(col.Type.Name, col.IsNotNull, col.Name, table.Rel.Name, sqlcConfig)
		jsonTag := util.ToCamelCase(col.Name)
		if col.IsNotNull {
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\" validate:\"required\"`\n", util.ToTitleCase(col.Name), goType, jsonTag))
		} else {
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n", util.ToTitleCase(col.Name), goType, jsonTag))
		}
	}

	// * add fields from addition that aren't in contraction
	for fieldName, fieldType := range additionFields {
		if _, exists := contractionFields[fieldName]; !exists {
			builder.WriteString(fmt.Sprintf("    %s %s\n", fieldName, fieldType))
		}
	}

	builder.WriteString("}\n")
	return builder.String()
}

func ModelParseStructFields(structContent string) map[string]string {
	fields := make(map[string]string)
	lines := strings.Split(structContent, "\n")

	inStruct := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "struct {") {
			inStruct = true
			continue
		}

		if inStruct && trimmed == "}" {
			break
		}

		if inStruct && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				fieldName := parts[0]
				fieldType := parts[1]
				// * remove backticks if present
				if strings.Contains(fieldType, "`") {
					fieldType = parts[1]
				}
				fields[fieldName] = fieldType
			}
		}
	}

	return fields
}

func ModelGenerateJoined(baseName string, table *catalog.Table, modelAddedBase string, tables map[string]*Table) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %sJoined struct {\n", baseName))

	// * parse modelAddedBase to get field content without type declaration
	lines := strings.Split(modelAddedBase, "\n")
	inStruct := false
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("type %sAdded struct {", baseName)) {
			inStruct = true
			continue
		}
		if inStruct && line == "}" {
			break
		}
		if inStruct && strings.TrimSpace(line) != "" {
			builder.WriteString(line + "\n")
		}
	}

	// * add child relationships based on table name
	relationships := ModelGetRelationships(baseName, "child", tables)
	for _, rel := range relationships {
		builder.WriteString(fmt.Sprintf("    %s\n", rel))
	}

	builder.WriteString("}\n")
	return builder.String()
}

func ModelGenerateParented(baseName string, table *catalog.Table, modelAddedBase string, tables map[string]*Table) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %sParented struct {\n", baseName))

	// * parse modelAddedBase to get field content without type declaration
	lines := strings.Split(modelAddedBase, "\n")
	inStruct := false
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("type %sAdded struct {", baseName)) {
			inStruct = true
			continue
		}
		if inStruct && line == "}" {
			break
		}
		if inStruct && strings.TrimSpace(line) != "" {
			builder.WriteString(line + "\n")
		}
	}

	// * add parent relationships based on table name
	relationships := ModelGetRelationships(baseName, "parent", tables)
	for _, rel := range relationships {
		builder.WriteString(fmt.Sprintf("    %s\n", rel))
	}

	builder.WriteString("}\n")
	return builder.String()
}

func ModelOrganizeStructs(baseName, mainStruct, additionStruct, contractionStruct, addedStruct, joinedStruct, parentedStruct string, existingStructs map[string]string) []string {
	var ordered []string

	// * 1. main model (e.g., User, Profile)
	if mainStruct != "" {
		ordered = append(ordered, mainStruct)
	}

	// * 2. model addition (e.g., UserAddition)
	if additionStruct != "" {
		ordered = append(ordered, additionStruct)
	}

	// * 3. model contraction (e.g., UserContraction)
	if contractionStruct != "" {
		ordered = append(ordered, contractionStruct)
	}

	// * 4. Model added (e.g., UserAdded)
	if addedStruct != "" {
		ordered = append(ordered, addedStruct)
	}

	// * 5. Model joined (e.g., UserJoined)
	if joinedStruct != "" {
		ordered = append(ordered, joinedStruct)
	}

	// * 6. Model parented (e.g., ProfileParented)
	if parentedStruct != "" {
		ordered = append(ordered, parentedStruct)
	}

	// * add other existing structs that weren't processed
	for name, content := range existingStructs {
		// * skip if it's one of the ones we already processed
		if name == baseName || name == baseName+"Addition" || name == baseName+"Contraction" ||
			name == baseName+"Added" || name == baseName+"Joined" || name == baseName+"Parented" {
			continue
		}
		ordered = append(ordered, content)
	}

	return ordered
}

func ModelGetRelationships(tableName, relationshipType string, tables map[string]*Table) []string {
	var relationships []string

	// * iterate through all tables to find foreign key relationships
	for _, table := range tables {
		for _, constraint := range table.Constraints {
			if constraint.Type == "FOREIGN KEY" {
				// * extract referenced table name (remove column references if present)
				referencedTable := constraint.References
				if parenIndex := strings.Index(referencedTable, "("); parenIndex != -1 {
					referencedTable = strings.TrimSpace(referencedTable[:parenIndex])
				}

				// * normalize table names for comparison
				targetStructName := util.ToSingularTitleCase(tableName)
				currentStructName := util.ToSingularTitleCase(table.Name)
				referencedStructName := util.ToSingularTitleCase(referencedTable)

				// * skip self-referencing relationships
				if currentStructName == referencedStructName {
					continue
				}

				if relationshipType == "child" && referencedStructName == targetStructName {
					// * this table references the target table (target is parent)
					// * so target has many of this table
					baseFieldName := currentStructName + "s"
					fieldName := baseFieldName

					// * if multiple foreign keys to same table, add column-based suffix
					for i, rel := range relationships {
						if strings.Contains(rel, currentStructName+"s []*"+currentStructName) {
							// * create unique name based on first FK column
							if len(constraint.Columns) > 0 {
								suffix := util.ToTitleCase(constraint.Columns[0])
								fieldName = currentStructName + "s" + suffix
							} else {
								fieldName = baseFieldName + fmt.Sprintf("%d", i+1)
							}
							break
						}
					}

					relationshipField := fmt.Sprintf("%s []*%s `json:\"%s\"`",
						fieldName, currentStructName, util.ToCamelCase(fieldName))
					relationships = append(relationships, relationshipField)
				} else if relationshipType == "parent" && currentStructName == targetStructName {
					// * this table is the target table and references another table (target is child)
					// * so target has one of the referenced table
					fieldName := referencedStructName

					// * if multiple foreign keys to same parent table, add column-based suffix
					if len(referencedStructName) > 0 {
						for i, rel := range relationships {
							if strings.Contains(rel, referencedStructName+" *"+referencedStructName) {
								if len(constraint.Columns) > 0 {
									suffix := util.ToTitleCase(constraint.Columns[0])
									fieldName = referencedStructName + suffix
								} else {
									fieldName = referencedStructName + fmt.Sprintf("%d", i+1)
								}
								break
							}
						}
					}

					relationshipField := fmt.Sprintf("%s *%s `json:\"%s\"`",
						fieldName, referencedStructName, util.ToCamelCase(fieldName))
					relationships = append(relationships, relationshipField)
				}
			}
		}
	}

	return relationships
}

func ModelGenerateFileContent(orderedStructs []string, requiredImports []string) string {
	var builder strings.Builder

	// * add package declaration
	builder.WriteString("package model\n\n")

	// * check if time.Time is used and add import if needed
	needsTimeImport := false
	for _, structContent := range orderedStructs {
		if strings.Contains(structContent, "time.Time") {
			needsTimeImport = true
			break
		}
	}

	// * build imports
	var imports []string
	if needsTimeImport {
		imports = append(imports, "time")
	}
	// * add required imports from sqlc overrides
	imports = append(imports, requiredImports...)

	// * write imports if any
	if len(imports) > 0 {
		builder.WriteString("import (\n")
		for _, imp := range imports {
			builder.WriteString(fmt.Sprintf("    \"%s\"\n", imp))
		}
		builder.WriteString(")\n\n")
	}

	// * add all ordered structs
	for _, structContent := range orderedStructs {
		builder.WriteString(structContent)
		if !strings.HasSuffix(structContent, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// * TODO: preserve other non-struct declarations from existing content

	return builder.String()
}
