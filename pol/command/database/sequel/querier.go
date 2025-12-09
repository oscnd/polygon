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
		if err := QuerierGenerate(querierDir, singularTableName, table, connection, parser, dirName); err != nil {
			return fmt.Errorf("failed to generate queriers for table %s: %w", tableName, err)
		}
	}

	if *parser.App.Verbose() {
		fmt.Printf("Generated queriers for %s\n", dirName)
	}

	return nil
}

func QuerierGenerate(querierDir, entityName string, table *Table, connection *Connection, parser *Parser, dirName string) error {
	// * get table config for features
	tableConfig := QuerierGetTableConfig(parser, dirName, table)

	// * generate all querier types
	queriers := QuerierGenerateAllQueries(entityName, table, connection, tableConfig)

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
func QuerierGetTableConfig(parser *Parser, dirName string, table *Table) *QuerierTableConfig {
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
func QuerierGenerateAllQueries(entityName string, table *Table, connection *Connection, tableConfig *QuerierTableConfig) []string {
	var queries []string

	// create querier
	queries = append(queries, QuerierGenerateCreate(entityName, table, connection))

	// update querier
	queries = append(queries, QuerierGenerateUpdate(entityName, table, connection))

	// count querier
	queries = append(queries, QuerierGenerateCount(entityName, table, connection, tableConfig))

	// one querier
	queries = append(queries, QuerierGenerateOne(entityName, table, connection))

	// one querier with combinations
	fkRefs := QuerierGetForeignKeyReferences(table)
	parentTables := QuerierGetParentTableNames(fkRefs)

	for _, combination := range QuerierGenerateCombinations(parentTables) {
		if len(combination) > 0 {
			queries = append(queries, QuerierGenerateOneWith(entityName, table, connection, combination))
		}
	}

	// * one counted querier
	queries = append(queries, QuerierGenerateOneCounted(entityName, table, connection))

	// * many querier
	queries = append(queries, QuerierGenerateMany(entityName, table, connection, tableConfig))

	// * many queriers with combinations
	for _, combination := range QuerierGenerateCombinations(parentTables) {
		if len(combination) > 0 {
			queries = append(queries, QuerierGenerateManyWith(entityName, table, connection, tableConfig, combination))
		}
	}

	// * list querier
	queries = append(queries, QuerierGenerateList(entityName, table, connection, tableConfig))

	// * list with combinations
	for _, combination := range QuerierGenerateCombinations(parentTables) {
		if len(combination) > 0 {
			queries = append(queries, QuerierGenerateListWith(entityName, table, connection, tableConfig, combination))
		}
	}

	// * increase queriers
	for _, field := range tableConfig.IncreaseFields {
		queries = append(queries, QuerierGenerateIncrease(entityName, table, field))
	}

	// * delete  querier
	queries = append(queries, QuerierGenerateDelete(entityName, table, connection))

	return queries
}

// QuerierGenerateCombinations generates all combinations of parent tables for With queriers
func QuerierGenerateCombinations(parentTables []string) [][]string {
	var combinations [][]string

	// * add single parent combinations in order
	for _, table := range parentTables {
		combinations = append(combinations, []string{table})
	}

	// * add combinations of multiple parents
	if len(parentTables) > 1 {
		// * generate combinations of 2 or more parents
		for i := 1; i < (1 << uint(len(parentTables))); i++ {
			var combination []string
			for j, table := range parentTables {
				if i&(1<<uint(j)) != 0 {
					combination = append(combination, table)
				}
			}
			if len(combination) > 1 {
				combinations = append(combinations, combination)
			}
		}
	}

	return combinations
}

// QuerierGetParentTableNames extracts parent table names from foreign key references
func QuerierGetParentTableNames(fkRefs map[string]string) []string {
	var parents []string
	seen := make(map[string]bool)

	for _, refTable := range fkRefs {
		if !seen[refTable] {
			parents = append(parents, refTable)
			seen[refTable] = true
		}
	}

	return parents
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
