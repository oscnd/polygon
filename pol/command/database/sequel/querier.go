package sequel

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.scnd.dev/polygon/polygon/util"
)

func Querier(parser *Parser, dirName string) error {
	// * get connection for this directory
	connection, exists := parser.Connections[dirName]
	if !exists {
		return fmt.Errorf("connection not found for directory: %s", dirName)
	}

	// * generate querier directory structure
	querierDir := filepath.Join("generate", "polygon", "sequel", dirName)
	if err := os.MkdirAll(querierDir, 0755); err != nil {
		return fmt.Errorf("failed to create querier directory: %w", err)
	}

	// * generate queriers for each table
	for _, tableName := range SortedTableKeys(connection.Tables) {
		table := connection.Tables[tableName]
		singularTableName := util.ToSingular(tableName)

		// * generate all required queriers for this table
		if err := QuerierGenerate(connection, parser, table, dirName, querierDir, singularTableName); err != nil {
			return fmt.Errorf("failed to generate queriers for table %s: %w", tableName, err)
		}
	}

	if *parser.App.Verbose() {
		fmt.Printf("Generated queriers for %s\n", dirName)
	}

	return nil
}

func QuerierGenerate(connection *Connection, parser *Parser, table *Table, dirName, querierDir, entityName string) error {
	// * get table config for features
	tableConfig := QuerierGetTableConfig(connection, parser, dirName, table)

	// * generate all querier types
	queriers := QuerierGenerateAllQueries(connection, parser, table, dirName, tableConfig, entityName)

	builder := new(strings.Builder)

	// * add header comment
	builder.WriteString("-- POLYGON GENERATED\n")
	builder.WriteString(fmt.Sprintf("-- table: %s\n\n", entityName))

	// * generate each querier and concatenate
	for _, querier := range queriers {
		builder.WriteString(querier)
		builder.WriteString("\n\n")
	}

	// * write single file for the table
	tableName := fmt.Sprintf("%ss", entityName) // plural form
	filename := fmt.Sprintf("%s.sql", tableName)
	path := filepath.Join(querierDir, filename)

	if err := os.WriteFile(path, []byte(builder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write querier file %s: %w", filename, err)
	}

	return nil
}

// QuerierTableConfig holds table-specific configuration for querier generation
type QuerierTableConfig struct {
	SortableFields []string
	FilterFields   []string
	IncreaseFields []string
}

// QuerierGetTableConfig extracts field features from table config
func QuerierGetTableConfig(connection *Connection, parser *Parser, dirName string, table *Table) *QuerierTableConfig {
	config := &QuerierTableConfig{
		SortableFields: []string{},
		FilterFields:   []string{},
		IncreaseFields: []string{},
	}

	// * extract features from table config
	tableName := *table.Name
	for _, column := range table.Columns {
		colName := *column.Name
		if columnFeatures := QuerierGetColumnFeatures(parser, dirName, tableName, colName); len(columnFeatures) > 0 {
			if slices.Contains(columnFeatures, "sort") {
				config.SortableFields = append(config.SortableFields, colName)
			}
			if slices.Contains(columnFeatures, "filter") {
				config.FilterFields = append(config.FilterFields, colName)
			}
			if slices.Contains(columnFeatures, "increase") {
				config.IncreaseFields = append(config.IncreaseFields, colName)
			}
		}
	}

	return config
}

// QuerierGetColumnFeatures gets features for a column from table config
func QuerierGetColumnFeatures(parser *Parser, dirName, tableName, columnName string) []string {
	// * get table config from parser
	if parser.Config == nil || parser.Config.Connections == nil {
		return []string{}
	}

	if dialectConfig, exists := parser.Config.Connections[dirName]; exists {
		if tableConfig, exists := dialectConfig.Tables[tableName]; exists {
			if fieldConfig := tableConfig.Field(columnName); fieldConfig != nil && fieldConfig.Include != nil {
				// * return the feature array from config
				if fieldConfig.Feature != nil {
					features := make([]string, 0, len(fieldConfig.Feature))
					for _, feature := range fieldConfig.Feature {
						if feature != nil {
							features = append(features, *feature)
						}
					}
					return features
				}
			}
		}
	}

	return []string{}
}

// QuerierGenerateAllQueries generates all querier types for a table in the specified order
func QuerierGenerateAllQueries(connection *Connection, parser *Parser, table *Table, dirName string, tableConfig *QuerierTableConfig, entityName string) []string {
	var queries []string

	// * check if table has join configuration
	joins := QuerierGetJoinConfigurations(parser, dirName, *table.Name)

	// * generate basic queriers
	queries = append(queries, QuerierGenerateCreate(connection, entityName, table))
	queries = append(queries, QuerierGenerateUpdate(connection, entityName, table))
	queries = append(queries, QuerierGenerateCount(connection, entityName, table, tableConfig))
	queries = append(queries, QuerierGenerateOne(connection, entityName, table))
	queries = append(queries, QuerierGenerateOneCounted(connection, entityName, table))
	queries = append(queries, QuerierGenerateMany(connection, entityName, table, tableConfig))
	queries = append(queries, QuerierGenerateManyCounted(connection, entityName, table))
	queries = append(queries, QuerierGenerateList(connection, entityName, table, tableConfig))

	// * generate "With" queriers only if join configuration exists
	if len(joins) > 0 {
		for _, join := range joins {
			joinName := QuerierBuildJoinName(join, connection, table)
			if joinName != "" {
				queries = append(queries, QuerierGenerateOneWithJoin(connection, entityName, table, join, joinName))
				queries = append(queries, QuerierGenerateManyWithJoin(connection, entityName, table, tableConfig, join, joinName))
				queries = append(queries, QuerierGenerateListWithJoin(connection, entityName, table, tableConfig, join, joinName))
			}
		}
	}

	// * increase queriers
	for _, field := range tableConfig.IncreaseFields {
		queries = append(queries, QuerierGenerateIncrease(connection, entityName, table, field))
	}

	// * delete querier
	queries = append(queries, QuerierGenerateDelete(connection, entityName, table))

	return queries
}

// QuerierGetForeignKeyReferences returns a map of column names to their referenced tables
func QuerierGetForeignKeyReferences(table *Table) map[string]string {
	references := make(map[string]string)
	for _, constraint := range table.Constraints {
		if *constraint.Type == "FOREIGN KEY" && len(constraint.Columns) == 1 {
			references[*constraint.Columns[0]] = *constraint.References
		}
	}
	return references
}

func QuerierGetChildTables(connection *Connection, tableName string) []*Table {
	var children []*Table
	for _, table := range connection.Tables {
		for _, constraint := range table.Constraints {
			if *constraint.Type == "FOREIGN KEY" && *constraint.References == tableName {
				children = append(children, table)
				break
			}
		}
	}
	return children
}

// QuerierGetPrimaryKeyColumns returns the primary key columns for a table
func QuerierGetPrimaryKeyColumns(table *Table) []string {
	var pkColumns []string
	for _, constraint := range table.Constraints {
		if *constraint.Type == "PRIMARY KEY" {
			for _, col := range constraint.Columns {
				pkColumns = append(pkColumns, *col)
			}
		}
	}

	// If no explicit primary key constraint, check if there's an 'id' column
	if len(pkColumns) == 0 {
		for _, column := range table.Columns {
			if *column.Name == "id" {
				pkColumns = append(pkColumns, "id")
				break
			}
		}
	}

	return pkColumns
}

// QuerierGetPrimaryKeyWhereClause returns the WHERE clause for the primary key
func QuerierGetPrimaryKeyWhereClause(table *Table, paramIndex int) string {
	pkColumns := QuerierGetPrimaryKeyColumns(table)
	if len(pkColumns) == 0 {
		// Fallback to id if no primary key found
		return fmt.Sprintf("id = $%d", paramIndex)
	}

	if len(pkColumns) == 1 {
		return fmt.Sprintf("%s = $%d", pkColumns[0], paramIndex)
	}

	// For composite primary keys, use a tuple
	var conditions []string
	for i, col := range pkColumns {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", col, paramIndex+i))
	}
	return "(" + strings.Join(conditions, " AND ") + ")"
}

// QuerierGetPrimaryKeyWhereInClause returns the WHERE IN clause for the primary key
func QuerierGetPrimaryKeyWhereInClause(table *Table, paramIndex int) string {
	pkColumns := QuerierGetPrimaryKeyColumns(table)
	if len(pkColumns) == 0 {
		// Fallback to id if no primary key found
		return fmt.Sprintf("id = ANY($%d::BIGINT[])", paramIndex)
	}

	if len(pkColumns) == 1 {
		return fmt.Sprintf("%s = ANY($%d::BIGINT[])", pkColumns[0], paramIndex)
	}

	// For composite keys, we can't use ANY, need to use multiple conditions
	// This is a limitation - composite keys with IN clauses need special handling
	return fmt.Sprintf("(%s) = ANY($%d::BIGINT[])", strings.Join(pkColumns, ", "), paramIndex)
}

// QuerierGetPrimaryKeyWhereClauseForUpdate returns the WHERE clause for the Update query
func QuerierGetPrimaryKeyWhereClauseForUpdate(table *Table) string {
	pkColumns := QuerierGetPrimaryKeyColumns(table)
	tableName := *table.Name
	if len(pkColumns) == 0 {
		// Fallback to id if no primary key found
		return fmt.Sprintf("%s.id = sqlc.narg('id')::BIGINT", tableName)
	}

	if len(pkColumns) == 1 {
		return fmt.Sprintf("%s.%s = sqlc.narg('id')::BIGINT", tableName, pkColumns[0])
	}

	// For composite primary keys, use named parameters with proper mapping
	var conditions []string
	for i, col := range pkColumns {
		paramName := "id"
		if i > 0 {
			paramName = col
		}
		conditions = append(conditions, fmt.Sprintf("%s.%s = sqlc.narg('%s')::BIGINT", tableName, col, paramName))
	}
	return strings.Join(conditions, " AND ")
}

// QuerierGetPrimaryKeyWhereClauseWithTable returns the WHERE clause with table prefix
func QuerierGetPrimaryKeyWhereClauseWithTable(table *Table, paramIndex int, tableName string) string {
	pkColumns := QuerierGetPrimaryKeyColumns(table)
	if len(pkColumns) == 0 {
		// Fallback to id if no primary key found
		return fmt.Sprintf("%s.id = $%d", tableName, paramIndex)
	}

	if len(pkColumns) == 1 {
		return fmt.Sprintf("%s.%s = $%d", tableName, pkColumns[0], paramIndex)
	}

	// For composite primary keys, use a tuple with qualified column names
	var conditions []string
	for i, col := range pkColumns {
		conditions = append(conditions, fmt.Sprintf("%s.%s = $%d", tableName, col, paramIndex+i))
	}
	return "(" + strings.Join(conditions, " AND ") + ")"
}

// * get join configurations for a table
func QuerierGetJoinConfigurations(parser *Parser, dirName, tableName string) []*ConfigJoin {
	// * get table config from parser
	if parser.Config == nil || parser.Config.Connections == nil {
		return nil
	}

	if dialectConfig, exists := parser.Config.Connections[dirName]; exists {
		if tableConfig, exists := dialectConfig.Tables[tableName]; exists {
			return tableConfig.Joins
		}
	}

	return nil
}

// * build join name from join configuration
func QuerierBuildJoinName(join *ConfigJoin, connection *Connection, table *Table) string {
	if join == nil || len(join.Fields) == 0 {
		return ""
	}

	// * collect root-level foreign keys (fields without dots)
	rootPaths := make(map[string][]string)

	for _, fieldPtr := range join.Fields {
		if fieldPtr == nil {
			continue
		}

		fieldPath := *fieldPtr
		if fieldPath == "" {
			continue
		}

		parts := strings.Split(fieldPath, ".")
		if len(parts) == 0 {
			continue
		}

		// * get root column name
		rootColumn := parts[0]

		// * track the full path for this root
		if _, exists := rootPaths[rootColumn]; !exists {
			rootPaths[rootColumn] = []string{}
		}
		rootPaths[rootColumn] = append(rootPaths[rootColumn], fieldPath)
	}

	// * build name parts from each root path chain
	var nameParts []string

	for _, paths := range rootPaths {
		// * find the longest path for this root
		longestPath := ""
		for _, path := range paths {
			if len(path) > len(longestPath) {
				longestPath = path
			}
		}

		// * convert path to table names
		pathTables := QuerierPathToTableNames(longestPath, table, connection)

		// * join table names (plural form)
		if len(pathTables) > 0 {
			nameParts = append(nameParts, strings.Join(pathTables, ""))
		}
	}

	if len(nameParts) == 0 {
		return ""
	}

	return "With" + strings.Join(nameParts, "And")
}

// * convert field path to table names in title case plural form
func QuerierPathToTableNames(fieldPath string, originTable *Table, connection *Connection) []string {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 0 {
		return nil
	}

	var tableNames []string
	currentTable := originTable

	for _, columnName := range parts {
		// * find FK constraint
		var foundConstraint *Constraint
		for _, constraint := range currentTable.Constraints {
			if *constraint.Type == "FOREIGN KEY" && len(constraint.Columns) == 1 {
				if *constraint.Columns[0] == columnName {
					foundConstraint = constraint
					break
				}
			}
		}

		if foundConstraint == nil {
			break
		}

		// * get referenced table
		referencedTable := *foundConstraint.References
		if parenIndex := strings.Index(referencedTable, "("); parenIndex != -1 {
			referencedTable = strings.TrimSpace(referencedTable[:parenIndex])
		}

		nextTable, exists := connection.Tables[referencedTable]
		if !exists {
			break
		}

		// * add table name in title case (keep plural)
		tableNames = append(tableNames, util.ToTitleCase(referencedTable))
		currentTable = nextTable
	}

	return tableNames
}

// * check if table has join configuration
func QuerierHasJoinConfiguration(parser *Parser, dirName, tableName string) bool {
	joins := QuerierGetJoinConfigurations(parser, dirName, tableName)
	return len(joins) > 0
}
