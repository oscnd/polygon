package sequel

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bsthun/gut"
	"go.scnd.dev/polygon/external/sqlc/config"
	"go.scnd.dev/polygon/external/sqlc/migrations"
	"go.scnd.dev/polygon/pol/index"
	"go.scnd.dev/polygon/polygon/util"
	"gopkg.in/yaml.v3"
)

type Parser struct {
	App         index.App
	Connections map[string]*Connection
	Config      *Config
	SqlcConfig  *config.Config
}

func NewParser(app index.App) (*Parser, error) {
	r := &Parser{
		App:         app,
		Connections: make(map[string]*Connection),
		Config:      nil,
		SqlcConfig:  nil,
	}

	// * parse all directories in sequel
	sequelDir := filepath.Join("sequel")
	entries, err := os.ReadDir(sequelDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("sequel directory not found: %s", sequelDir)
		}
		return nil, fmt.Errorf("failed to read sequel directory: %w", err)
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
			continue // skip directories without migration
		}

		// * create connection for this directory
		connection := &Connection{
			Dialect: gut.Ptr("postgres"), // default
			Tables:  make(map[string]*Table),
		}

		// * parse migrations for this directory into connection
		if err := r.ParseConnection(dirName, migrationDir, connection); err != nil {
			log.Printf("Warning: failed to parse directory %s: %v", dirName, err)
			continue
		}

		// * ensure dialect is set to postgres
		if connection.Dialect == nil {
			connection.Dialect = gut.Ptr("postgres")
		}

		// * store connection by directory name
		r.Connections[dirName] = connection
	}

	// * parse configurations once
	if err := r.ParseConfig(); err != nil {
		return nil, fmt.Errorf("failed to parse configurations: %w", err)
	}

	return r, nil
}

func (r *Parser) ParseConnection(dirName, migrationDir string, connection *Connection) error {
	// * find all sql files in migration directory
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasPrefix(name, ".") || migrations.IsDown(name) {
			continue
		}
		migrationFiles = append(migrationFiles, filepath.Join(migrationDir, name))
	}

	if len(migrationFiles) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationDir)
	}

	// * sort files for consistent order
	sort.Strings(migrationFiles)

	// * parse all migration files using sequel parser only
	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// * remove rollback statements
		cleanContent := migrations.RemoveRollbackStatements(string(content))

		// * parse with sequel parser to get table information
		ParseMigration(cleanContent, connection.Tables, make(map[string]*Function), make(map[string]*Trigger))
	}

	return nil
}

func (r *Parser) ParseConfig() error {
	// * parse sqlc configuration
	sqlcConfigPath := "sqlc.yml"
	sqlcConfigFile, err := os.Open(sqlcConfigPath)
	if err != nil {
		return fmt.Errorf("no sqlc configuration found: %w", err)
	}
	defer sqlcConfigFile.Close()

	if parsedConfig, parseErr := config.ParseConfig(sqlcConfigFile); parseErr != nil {
		return fmt.Errorf("failed to parse sqlc.yml: %w", parseErr)
	} else {
		r.SqlcConfig = &parsedConfig
	}

	// * parse sequel configuration
	configPath := filepath.Join(*r.App.Directory(), "sequel.yml")
	var configData []byte
	if configData, err = os.ReadFile(configPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read sequel.yml: %w", err)
		}
		// * file not found is not fatal, use empty config
		r.Config = &Config{Connections: make(map[string]*ConfigConnection)}
		return nil
	}

	if err := yaml.Unmarshal(configData, &r.Config); err != nil {
		return fmt.Errorf("failed to parse sequel.yml: %w", err)
	}

	// * revise sequel config based on connections
	if err := r.ReviseConfig(); err != nil {
		return fmt.Errorf("failed to revise sequel config: %w", err)
	}

	return nil
}

func (r *Parser) ReviseConfig() error {
	// * ensure sequel config structure exists
	if r.Config == nil {
		r.Config = &Config{Connections: make(map[string]*ConfigConnection)}
	}

	if r.Config.Connections == nil {
		r.Config.Connections = make(map[string]*ConfigConnection)
	}

	// * ensure each connection has a dialect
	for _, connection := range r.Config.Connections {
		if connection.Dialect == nil {
			connection.Dialect = gut.Ptr("postgres")
		}
		// ensure tables map exists
		if connection.Tables == nil {
			connection.Tables = make(map[string]*ConfigTable)
		}
	}

	// * add missing tables and fields from connections
	updated := false
	for connName, connection := range r.Connections {
		// * find or create config for this connection
		connectionConfig, exists := r.Config.Connections[connName]
		if !exists {
			connectionConfig = &ConfigConnection{
				Dialect: gut.Ptr("postgres"),
				Tables:  make(map[string]*ConfigTable),
			}
			r.Config.Connections[connName] = connectionConfig
			updated = true
		}

		// * add tables from this connection to its config
		for _, table := range connection.Tables {
			tableConfig, tableExists := connectionConfig.Tables[*table.Name]
			if !tableExists {
				tableConfig = &ConfigTable{
					Fields:    make(map[string]*ConfigField),
					Additions: []*ConfigAddition{},
				}
				connectionConfig.Tables[*table.Name] = tableConfig
				updated = true
			}

			// * ensure fields map exists
			if tableConfig.Fields == nil {
				tableConfig.Fields = make(map[string]*ConfigField)
			}

			// * add missing fields with include: "base"
			for _, column := range table.Columns {
				if _, fieldExists := tableConfig.Fields[*column.Name]; !fieldExists {
					tableConfig.Fields[*column.Name] = &ConfigField{
						Include: gut.Ptr("base"),
					}
					updated = true
				} else {
					// * update existing nil includes to "base"
					if tableConfig.Fields[*column.Name].Include == nil {
						tableConfig.Fields[*column.Name].Include = gut.Ptr("base")
						updated = true
					}
				}
			}
		}
	}

	// * write back config if updated
	if updated {
		configData, err := yaml.Marshal(r.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal sequel.yml: %w", err)
		}

		configPath := filepath.Join(*r.App.Directory(), "sequel.yml")
		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to write sequel.yml: %w", err)
		}

		if *r.App.Verbose() {
			log.Printf("updated sequel configuration with connection mapping")
		}
	}

	return nil
}

func (r *Parser) ShouldIncludeField(include *string) bool {
	if include == nil {
		return true
	}
	return *include != "none"
}

func (r *Parser) IsIncludeEqual(include *string, target string) bool {
	if include == nil {
		return target == "base"
	}
	return *include == target
}

func (r *Parser) ValidateFields(tableConfig *ConfigTable, table *Table) error {
	for fieldName, fieldConfig := range tableConfig.Fields {
		if fieldConfig.Include != nil && r.ShouldIncludeField(fieldConfig.Include) {
			// * check if field exists in database schema
			found := false
			for _, col := range table.Columns {
				if *col.Name == fieldName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("field '%s' from sequel.yml not found in table '%s' database schema", fieldName, *table.Name)
			}
		}
	}
	return nil
}

func (r *Parser) ValidateAdditions(additions []*ConfigAddition, table *Table) error {
	for _, addition := range additions {
		if addition.Name != nil {
			// * check if addition conflicts with existing columns
			for _, col := range table.Columns {
				if *col.Name == *addition.Name {
					return fmt.Errorf("addition '%s' conflicts with existing column in table '%s'", *addition.Name, *table.Name)
				}
			}
		}
	}
	return nil
}

func (r *Parser) GenerateStruct(name string, table *Table, tableConfig *ConfigTable) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, col := range table.Columns {
		// * check if field should be included based on config
		shouldInclude := true
		if tableConfig != nil && tableConfig.Fields != nil {
			if fieldConfig, exists := tableConfig.Fields[*col.Name]; exists {
				shouldInclude = r.ShouldIncludeField(fieldConfig.Include)
			}
		}

		if shouldInclude {
			// * convert SQL type to Go type
			goType := r.SqlToGoType(*col.Type, !*col.Nullable, *col.Name, "")

			// * generate json tag
			jsonTag := util.ToCamelCase(*col.Name)

			// * generate field
			if !*col.Nullable {
				builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\" validate:\"required\"`\n", util.ToTitleCase(*col.Name), goType, jsonTag))
			} else {
				builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n", util.ToTitleCase(*col.Name), goType, jsonTag))
			}
		}
	}

	builder.WriteString("}\n")
	return builder.String()
}

func (r *Parser) GenerateAdditionStruct(structName string, additions []*ConfigAddition) string {
	if len(additions) == 0 {
		return fmt.Sprintf("type %sAddition struct {\n}\n", structName)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %sAddition struct {\n", structName))

	for _, addition := range additions {
		if addition.Name != nil {
			fieldType := r.convertAdditionType(addition)
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n",
				util.ToTitleCase(*addition.Name), fieldType, util.ToCamelCase(*addition.Name)))
		}
	}

	builder.WriteString("}\n")
	return builder.String()
}

func (r *Parser) GenerateContractionStruct(structName string, fields map[string]*ConfigField, table *Table) string {
	// * find excluded fields (include: false or "none")
	var excludedFields []string
	for fieldName, fieldConfig := range fields {
		if !r.ShouldIncludeField(fieldConfig.Include) {
			excludedFields = append(excludedFields, fieldName)
		}
	}

	if len(excludedFields) == 0 {
		return fmt.Sprintf("type %sContraction struct {\n}\n", structName)
	}

	// * generate struct with excluded fields only, using original types
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %sContraction struct {\n", structName))

	for _, field := range excludedFields {
		// * find the column to get original type
		var goType = "any"
		for _, col := range table.Columns {
			if *col.Name == field {
				goType = r.SqlToGoType(*col.Type, !*col.Nullable, *col.Name, "")
				break
			}
		}
		builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n",
			util.ToTitleCase(field), goType, util.ToCamelCase(field)))
	}
	builder.WriteString("}\n")
	return builder.String()
}

func (r *Parser) GenerateAdded(baseName string, table *Table, additionStruct, contractionStruct string, tableConfig *ConfigTable) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %sAdded struct {\n", baseName))

	// * parse addition and contraction structs to get fields
	additionFields := ModelParseStructFields(additionStruct)
	contractionFields := ModelParseStructFields(contractionStruct)

	// * add main struct fields filtered by tableConfig
	for _, col := range table.Columns {
		// * check if field should be included based on config
		shouldInclude := true
		if tableConfig != nil && tableConfig.Fields != nil {
			if fieldConfig, exists := tableConfig.Fields[*col.Name]; exists {
				shouldInclude = r.ShouldIncludeField(fieldConfig.Include)
			}
		}

		if shouldInclude {
			goType := r.SqlToGoType(*col.Type, !*col.Nullable, *col.Name, "")
			jsonTag := util.ToCamelCase(*col.Name)
			if !*col.Nullable {
				builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\" validate:\"required\"`\n", util.ToTitleCase(*col.Name), goType, jsonTag))
			} else {
				builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n", util.ToTitleCase(*col.Name), goType, jsonTag))
			}
		}
	}

	// * add fields from addition that aren't in contraction
	for fieldName, fieldType := range additionFields {
		if _, exists := contractionFields[fieldName]; !exists {
			// * check if we have JSON tag stored from parsing
			jsonTag := util.ToCamelCase(fieldName)
			cleanFieldType := fieldType
			if strings.Contains(fieldType, "|") {
				parts := strings.Split(fieldType, "|")
				if len(parts) == 2 {
					cleanFieldType = parts[0]
					jsonTag = parts[1]
				}
			} else if strings.Contains(fieldType, "`json:") {
				// * fallback: parse JSON tag from field type
				start := strings.Index(fieldType, "`json:\"") + 7
				end := strings.Index(fieldType[start:], "\"")
				if end != -1 {
					jsonTag = fieldType[start : start+end]
				}
				// * remove backticks from field type
				cleanFieldType = strings.Fields(fieldType)[1]
			}
			builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n", fieldName, cleanFieldType, jsonTag))
		}
	}

	builder.WriteString("}\n")
	return builder.String()
}

func (r *Parser) GenerateJoined(baseName string, table *Table, modelAddedBase string) string {
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
	relationships := r.GetRelationships(baseName, "child")
	for _, rel := range relationships {
		builder.WriteString(fmt.Sprintf("    %s\n", rel))
	}

	builder.WriteString("}\n")
	return builder.String()
}

func (r *Parser) GenerateParented(baseName string, table *Table, modelAddedBase string) string {
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
	relationships := r.GetRelationships(baseName, "parent")
	for _, rel := range relationships {
		builder.WriteString(fmt.Sprintf("    %s\n", rel))
	}

	builder.WriteString("}\n")
	return builder.String()
}

func (r *Parser) GetRelationships(tableName, relationshipType string) []string {
	var relationships []string

	// * convert connections to table map for compatibility
	tables := make(map[string]*Table)
	for _, connection := range r.Connections {
		for name, table := range connection.Tables {
			tables[name] = table
		}
	}

	// * iterate through all tables to find foreign key relationships
	for _, table := range tables {
		for _, constraint := range table.Constraints {
			if constraint.Type != nil && *constraint.Type == "FOREIGN KEY" {
				// * extract referenced table name (remove column references if present)
				referencedTable := *constraint.References
				if parenIndex := strings.Index(referencedTable, "("); parenIndex != -1 {
					referencedTable = strings.TrimSpace(referencedTable[:parenIndex])
				}

				// * normalize table names for comparison
				targetStructName := util.ToSingularTitleCase(tableName)
				currentStructName := util.ToSingularTitleCase(*table.Name)
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
								suffix := util.ToTitleCase(*constraint.Columns[0])
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
									suffix := util.ToTitleCase(*constraint.Columns[0])
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

func (r *Parser) SqlToGoType(sqlType string, notNull bool, columnName string, tableName string) string {
	sqlType = strings.ToLower(sqlType)

	// * check for JSON/JSONB columns and look for sqlc overrides
	if strings.Contains(sqlType, "json") || strings.Contains(sqlType, "jsonb") {
		// * check sqlc config for overrides
		if r.SqlcConfig != nil && r.SqlcConfig.SQL != nil {
			for _, sql := range r.SqlcConfig.SQL {
				if sql.Gen.Go != nil && len(sql.Gen.Go.Overrides) > 0 {
					for _, override := range sql.Gen.Go.Overrides {
						// * check if this override matches our column
						if override.Column != "" {
							// * format: "table.column"
							if strings.Contains(override.Column, ".") {
								parts := strings.Split(override.Column, ".")
								if len(parts) == 2 {
									overrideTable := strings.TrimSpace(parts[0])
									overrideColumn := strings.TrimSpace(parts[1])
									if overrideTable == tableName && overrideColumn == columnName {
										// * found matching override, construct type from override
										goType := ""
										if override.GoType.Package != "" && override.GoType.Path != "" {
											// * imported type
											if override.GoType.Package != "" {
												goType = override.GoType.Package + "." + override.GoType.Name
											} else {
												pathParts := strings.Split(override.GoType.Path, "/")
												packageName := pathParts[len(pathParts)-1]
												goType = packageName + "." + override.GoType.Name
											}
										} else {
											// * built-in type
											goType = override.GoType.Name
										}
										if override.GoType.Pointer {
											goType = "*" + goType
										}
										if goType != "" {
											return goType
										}
									}
								}
							}
						}
					}
				}
			}
		}
		// * no override found, use default
		return "any"
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
	case strings.Contains(sqlType, "uuid"):
		return "*string"
	default:
		return "*string"
	}
}

func (r *Parser) convertAdditionType(addition *ConfigAddition) string {
	// * if package specified, use qualified type name
	if addition.Package != nil && *addition.Package != "" {
		// * extract package name from path
		pathParts := strings.Split(*addition.Package, "/")
		packageName := pathParts[len(pathParts)-1]
		return "*" + packageName + "." + *addition.Type
	}

	// * default to pointer type
	if addition.Type != nil {
		switch strings.ToLower(*addition.Type) {
		case "string":
			return "*string"
		case "int", "int64":
			return "*int64"
		case "uint64":
			return "*uint64"
		case "bool":
			return "*bool"
		case "time", "timestamp":
			return "*time.Time"
		default:
			return "*" + *addition.Type
		}
	}

	return "*string"
}
