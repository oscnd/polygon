package sequel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/polygon/polygon/util"
)

func Model(parser *Parser, dirName string) error {
	// * get connection
	connection, exists := parser.Connections[dirName]
	if !exists {
		return fmt.Errorf("connection not found for directory: %s", dirName)
	}

	// * generate structs
	for tableName, table := range connection.Tables {
		if err := ModelGenerate(tableName, table, parser, dirName); err != nil {
			return fmt.Errorf("failed to generate model for table %s: %w", tableName, err)
		}
	}

	return nil
}

func ModelGenerate(tableName string, table *Table, parser *Parser, dirName string) error {
	// * construct model file paths using singular table name
	singularTableName := util.ToSingular(tableName)
	generatedModelDir := filepath.Join("generate", "polygon", "model")
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

	// * get table config from sequel.yml
	var tableConfig *ConfigTable
	if parser.Config != nil && parser.Config.Connections != nil {
		if dialectConfig, exists := parser.Config.Connections[dirName]; exists {
			if tc, exists := dialectConfig.Tables[tableName]; exists {
				tableConfig = tc
			}
		}
	}

	// * validate sequel.yml config if exists
	if tableConfig != nil {
		if err := parser.ValidateFields(tableConfig, table); err != nil {
			return err
		}
		if err := parser.ValidateAdditions(tableConfig.Additions, table); err != nil {
			return err
		}
	}

	// * collect required imports from addition config only
	var requiredImports []string

	// * add imports from addition config
	if tableConfig != nil {
		additionImports := ModelExtractAdditionImports(tableConfig.Additions)
		requiredImports = append(requiredImports, additionImports...)
	}

	// * parse existing structs
	existingStructs := ModelParseExistingStructs(existingContent)

	// * generate struct name in title case (singular form)
	structName := util.ToSingularTitleCase(tableName)

	// * generate main struct using tableConfig
	mainStruct := parser.GenerateStruct(structName, table, tableConfig)

	// * generate addition/contraction structs from config
	var additionStruct, contractionStruct string
	if tableConfig != nil {
		additionStruct = parser.GenerateAdditionStruct(structName, tableConfig.Additions)
		contractionStruct = parser.GenerateContractionStruct(structName, tableConfig.Fields, table)
	} else {
		// * default to empty structs if no config
		additionStruct = fmt.Sprintf("type %sAddition struct {\n}\n", structName)
		contractionStruct = fmt.Sprintf("type %sContraction struct {\n}\n", structName)
	}

	// * generate ModelAdded struct using reflection
	addedStruct := parser.GenerateAdded(structName, table, additionStruct, contractionStruct, tableConfig)

	// * generate ModelAdded struct first to use as base for Joined/Parented
	modelAddedBase := parser.GenerateAdded(structName, table, additionStruct, contractionStruct, tableConfig)

	// * generate ModelJoined struct with references
	joinedStruct := parser.GenerateJoined(structName, table, modelAddedBase)

	// * generate ModelParented struct with parent references
	parentedStruct := parser.GenerateParented(structName, table, modelAddedBase)

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

func ModelOrganizeStructs(baseName, mainStruct, additionStruct, contractionStruct, addedStruct, joinedStruct, parentedStruct string, existingStructs map[string]string) []string {
	var ordered []string

	// * main model
	if mainStruct != "" {
		ordered = append(ordered, mainStruct)
	}

	// * addition model
	if additionStruct != "" {
		ordered = append(ordered, additionStruct)
	}

	// * contraction model
	if contractionStruct != "" {
		ordered = append(ordered, contractionStruct)
	}

	// * added model
	if addedStruct != "" {
		ordered = append(ordered, addedStruct)
	}

	// * joined model
	if joinedStruct != "" {
		ordered = append(ordered, joinedStruct)
	}

	// * parented model
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
				// * check for JSON tag in the complete line
				if strings.Contains(trimmed, "`json:") {
					// * extract JSON tag for proper camel case
					start := strings.Index(trimmed, "`json:\"") + 7
					end := strings.Index(trimmed[start:], "\"")
					if end != -1 {
						jsonTag := trimmed[start : start+end]
						// * remove backticks from field type
						cleanFieldType := strings.TrimSuffix(fieldType, "`")
						fields[fieldName] = cleanFieldType + "|" + jsonTag
					} else {
						fields[fieldName] = fieldType
					}
				} else {
					fields[fieldName] = fieldType
				}
			}
		}
	}

	return fields
}

func ModelExtractAdditionImports(additions []*ConfigAddition) []string {
	var imports []string
	seen := make(map[string]bool)

	for _, addition := range additions {
		if addition.Package != nil && *addition.Package != "" && !seen[*addition.Package] {
			imports = append(imports, *addition.Package)
			seen[*addition.Package] = true
		}
	}

	return imports
}
