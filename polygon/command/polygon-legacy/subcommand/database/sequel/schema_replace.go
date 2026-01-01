package sequel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReplaceGeneratedTypes performs the type replacements on generated Go files
func ReplaceGeneratedTypes(parser *Parser) error {
	// Get the output directory from sqlc config
	if parser.SqlcConfig == nil || len(parser.SqlcConfig.SQL) == 0 {
		return fmt.Errorf("no sqlc configuration found")
	}

	for _, sql := range parser.SqlcConfig.SQL {
		if sql.Gen.Go != nil && sql.Gen.Go.Out != "" {
			outputDir := sql.Gen.Go.Out

			// Walk through all generated Go files in the output directory
			err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Only process .go files that are models, querier, or .sql.go files
				if !strings.HasSuffix(path, ".go") {
					return nil
				}

				filename := filepath.Base(path)
				if filename != "models.go" &&
					!strings.HasSuffix(filename, ".sql.go") &&
					filename != "querier.go" {
					return nil
				}

				return ReplaceFileGeneratedTypes(path)
			})

			if err != nil {
				return fmt.Errorf("error processing directory %s: %w", outputDir, err)
			}

			interfaceFileTemplate := "package %s\n\nimport (\n\t\"context\"\n\t\"database/sql\"\n)\n\ntype Database interface {\n\tP() Querier\n\tPtx(context context.Context, opts *sql.TxOptions) (DatabaseTx, Querier)\n}\n\ntype DatabaseTx interface {\n\tCommit() error\n\tRollback() error\n}\n"
			interfaceFileContent := fmt.Sprintf(interfaceFileTemplate, filepath.Base(outputDir))
			if err := os.WriteFile(filepath.Join(outputDir, "interface.go"), []byte(interfaceFileContent), 0644); err != nil {
				return fmt.Errorf("failed to write interface.go: %w", err)
			}
		}
	}

	return nil
}

// ReplaceFileGeneratedTypes performs type replacements on a single file
func ReplaceFileGeneratedTypes(filePath string) error {
	// Read the entire file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	fileContent := string(content)

	// Check if we need time import (check before replacements)
	hasTimeType := strings.Contains(fileContent, "time.Time") || strings.Contains(fileContent, "sql.NullTime")
	hasTimeImport := strings.Contains(fileContent, `"time"`)

	// Apply replacements
	fileContent = strings.ReplaceAll(fileContent, `"database/sql"`, "")
	fileContent = strings.ReplaceAll(fileContent, "string", "*string")
	fileContent = strings.ReplaceAll(fileContent, " bool,", " *bool,")
	fileContent = strings.ReplaceAll(fileContent, " bool)", " *bool)")
	fileContent = strings.ReplaceAll(fileContent, " bool\n", " *bool\n")
	fileContent = strings.ReplaceAll(fileContent, " bool ", " *bool ")
	fileContent = strings.ReplaceAll(fileContent, "\tbool:", "\t*bool:")
	fileContent = strings.ReplaceAll(fileContent, "int64", "*uint64")
	fileContent = strings.ReplaceAll(fileContent, "int32", "*int32")
	fileContent = strings.ReplaceAll(fileContent, "float64", "*float64")
	fileContent = strings.ReplaceAll(fileContent, "time.Time", "*time.Time")
	fileContent = strings.ReplaceAll(fileContent, "sql.NullString", "*string")
	fileContent = strings.ReplaceAll(fileContent, "sql.NullTime", "*time.Time")
	fileContent = strings.ReplaceAll(fileContent, "sql.NullInt64", "*uint64")
	fileContent = strings.ReplaceAll(fileContent, "sql.NullInt32", "*int32")
	fileContent = strings.ReplaceAll(fileContent, "sql.NullFloat64", "*float64")
	fileContent = strings.ReplaceAll(fileContent, "sql.NullBool", "*bool")

	// Add time import if needed
	if hasTimeType && !hasTimeImport {
		// Find the import block and add time import
		importStart := strings.Index(fileContent, "import (")
		if importStart >= 0 {
			importEnd := strings.Index(fileContent[importStart:], ")")
			if importEnd >= 0 {
				importEnd += importStart
				// Find the last newline before closing parenthesis to maintain formatting
				lastNewline := strings.LastIndex(fileContent[:importEnd], "\n")
				if lastNewline >= 0 {
					// Insert time import before the closing parenthesis
					fileContent = fileContent[:lastNewline] + "\n\t\"time\"" + fileContent[lastNewline:]
				} else {
					// No newline found, just add before closing parenthesis
					fileContent = fileContent[:importEnd] + "\n\t\"time\"" + fileContent[importEnd:]
				}
			}
		}
	}

	// Write the modified content back
	lines := strings.Split(fileContent, "\n")
	return WriteFileReplaced(filePath, lines)
}

// WriteFileReplaced writes lines to a file
func WriteFileReplaced(filePath string, lines []string) error {
	// Create a temporary file
	tempPath := filePath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Write all lines
	for _, line := range lines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
	}

	// Replace the original file
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	return nil
}
