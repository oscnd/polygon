package sequel

import (
	"fmt"
	"strings"

	"go.scnd.dev/polygon/polygon/util"
)

func QuerierGenerateCreate(connection *Connection, entityName string, table *Table) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName) // plural form

	var columns []string
	var placeholders []string

	for _, column := range table.Columns {
		colName := *column.Name

		// * skip auto-increment primary key and timestamp fields
		if strings.Contains(strings.ToUpper(*column.Type), "SERIAL") ||
			colName == "id" ||
			colName == "created_at" ||
			colName == "updated_at" {
			continue
		}

		columns = append(columns, colName)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(columns)))
	}

	return fmt.Sprintf(`-- name: %sCreate :one
INSERT INTO %s (%s)
VALUES (%s)
RETURNING *;`,
		entityTitleCase,
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
}

// QuerierGenerateOne generates the basic One querier
func QuerierGenerateOne(connection *Connection, entityName string, table *Table) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %sOne :one
SELECT * FROM %s WHERE id = $1 LIMIT 1;`,
		entityTitleCase,
		tableName)
}

// QuerierGenerateOneCounted generates the OneCounted querier with child relation counts
func QuerierGenerateOneCounted(connection *Connection, entityName string, table *Table) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	childTables := QuerierGetChildTables(connection, tableName)
	var selectFields []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * add child table counts
	for _, childTable := range childTables {
		childEntityName := util.ToSingular(*childTable.Name)
		countField := fmt.Sprintf("%s_count", childEntityName)
		selectFields = append(selectFields, fmt.Sprintf(`(SELECT COALESCE(COUNT(*), 0)::BIGINT FROM %s WHERE %s.%s_id = %s.id) AS %s`,
			*childTable.Name, *childTable.Name, entityName, tableName, countField))
	}

	return fmt.Sprintf(`-- name: %sOneCounted :one
SELECT %s
FROM %s
WHERE %s.id = $1
LIMIT 1;`,
		entityTitleCase,
		strings.Join(selectFields, ",\n       "),
		tableName,
		tableName)
}

// QuerierGenerateManyCounted generates the ManyCounted querier with child relation counts
func QuerierGenerateManyCounted(connection *Connection, entityName string, table *Table) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	childTables := QuerierGetChildTables(connection, tableName)
	var selectFields []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * add child table counts
	for _, childTable := range childTables {
		childEntityName := util.ToSingular(*childTable.Name)
		countField := fmt.Sprintf("%s_count", childEntityName)
		selectFields = append(selectFields, fmt.Sprintf(`(SELECT COALESCE(COUNT(*), 0)::BIGINT FROM %s WHERE %s.%s_id = %s.id) AS %s`,
			*childTable.Name, *childTable.Name, entityName, tableName, countField))
	}

	return fmt.Sprintf(`-- name: %sManyCounted :many
SELECT %s
FROM %s
WHERE %s.id = ANY($1::BIGINT[]);`,
		entityTitleCase,
		strings.Join(selectFields, ",\n       "),
		tableName,
		tableName)
}

// QuerierGenerateMany generates the Many querier with IN filter
func QuerierGenerateMany(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %sMany :many
SELECT * FROM %s WHERE id = ANY($1::BIGINT[]);`,
		entityTitleCase,
		tableName)
}

// QuerierGenerateCount generates the Count querier with same conditions as List
func QuerierGenerateCount(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	fkRefs := QuerierGetForeignKeyReferences(table)
	var whereConditions []string

	// * add filter conditions for parent relations
	for columnName, refTable := range fkRefs {
		refEntityName := util.ToSingular(refTable)
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			refEntityName, tableName, columnName, refEntityName))
	}

	// * add filter conditions for fields with filter feature
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, tableName, field, field))
	}

	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	query := fmt.Sprintf(`-- name: %sCount :one
SELECT COALESCE(COUNT(*), 0)::BIGINT AS %s_count
FROM %s`,
		entityTitleCase,
		entityName,
		tableName)

	if whereClause != "" {
		query += "\n" + whereClause
	}

	query += ";"

	return query
}

// QuerierGenerateIncrease generates the Increase querier for numeric fields
func QuerierGenerateIncrease(connection *Connection, entityName string, table *Table, fieldName string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	fieldTitleCase := util.ToTitleCase(fieldName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %s%sIncrease :one
UPDATE %s
SET %s = COALESCE(%s, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;`,
		entityTitleCase,
		fieldTitleCase,
		tableName,
		fieldName,
		fieldName)
}

func QuerierGenerateList(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	// * List querier does NOT have joins, only main table
	var selectFields []string
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * build WHERE clause based on filter fields
	var whereConditions []string
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, tableName, field, field))
	}

	// * build WHERE clause
	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	// * build final query without ORDER BY
	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sList :many\n", entityTitleCase))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if whereClause != "" {
		query.WriteString("\n")
		query.WriteString(whereClause)
	}

	query.WriteString("\nLIMIT sqlc.narg('limit')::BIGINT")
	query.WriteString("\nOFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);")

	return query.String()
}

func QuerierGenerateUpdate(connection *Connection, entityName string, table *Table) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	var setConditions []string

	for _, column := range table.Columns {
		colName := *column.Name

		// * skip id and auto-managed timestamp fields
		if colName == "id" ||
			colName == "created_at" ||
			colName == "updated_at" {
			continue
		}

		// * use coalesce for partial updates
		setConditions = append(setConditions, fmt.Sprintf(`%s = COALESCE(sqlc.narg('%s'), %s)`,
			colName, colName, colName))
	}

	return fmt.Sprintf(`-- name: %sUpdate :one
UPDATE %s
SET %s
WHERE id = sqlc.narg('id')::BIGINT
RETURNING *;`,
		entityTitleCase,
		tableName,
		strings.Join(setConditions, ",\n    "))
}

func QuerierGenerateDelete(connection *Connection, entityName string, table *Table) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %sDelete :one
DELETE FROM %s WHERE id = $1 RETURNING *;`,
		entityTitleCase,
		tableName)
}

// QuerierGenerateOneWith generates One queriers with specific parent combinations
func QuerierGenerateOneWith(connection *Connection, entityName string, table *Table, parentTables []string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	fkRefs := QuerierGetForeignKeyReferences(table)
	var selectFields []string
	var joinConditions []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * add selected parent table embeddings and joins
	for _, parentTable := range parentTables {
		// * find the column that references this parent table
		for columnName, refTable := range fkRefs {
			if refTable == parentTable {
				refEntityName := util.ToSingular(refTable)
				selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", refEntityName))
				joinConditions = append(joinConditions, fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.id",
					refTable, tableName, columnName, refTable))
				break
			}
		}
	}

	// * build querier name with With suffix
	var withSuffix string
	if len(parentTables) > 0 {
		var parentNames []string
		for _, parent := range parentTables {
			parentNames = append(parentNames, util.ToTitleCase(util.ToSingular(parent)))
		}
		withSuffix = "With" + strings.Join(parentNames, "")
	} else {
		return "" // * skip if no parent tables
	}

	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %s%s%s :one\n", entityTitleCase, "One", withSuffix))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	query.WriteString(fmt.Sprintf("\nWHERE %s.id = $1\n", tableName))
	query.WriteString("LIMIT 1;")

	return query.String()
}

// QuerierGenerateManyWith generates Many queriers with specific parent combinations
func QuerierGenerateManyWith(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig, parentTables []string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	fkRefs := QuerierGetForeignKeyReferences(table)
	var selectFields []string
	var joinConditions []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * add selected parent table embeddings and joins
	for _, parentTable := range parentTables {
		// * find the column that references this parent table
		for columnName, refTable := range fkRefs {
			if refTable == parentTable {
				refEntityName := util.ToSingular(refTable)
				selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", refEntityName))
				joinConditions = append(joinConditions, fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.id",
					refTable, tableName, columnName, refTable))
				break
			}
		}
	}

	// * build querier name with With suffix
	var withSuffix string
	if len(parentTables) > 0 {
		var parentNames []string
		for _, parent := range parentTables {
			parentNames = append(parentNames, util.ToTitleCase(util.ToSingular(parent)))
		}
		withSuffix = "With" + strings.Join(parentNames, "")
	} else {
		return "" // * skip if no parent tables
	}

	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %s%s%s :many\n", entityTitleCase, "Many", withSuffix))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	query.WriteString(fmt.Sprintf("\nWHERE %s.id = ANY($1::BIGINT[])", tableName))
	query.WriteString(";")

	return query.String()
}

// QuerierGenerateListWith generates List queriers with specific parent combinations
func QuerierGenerateListWith(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig, parentTables []string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	fkRefs := QuerierGetForeignKeyReferences(table)
	childTables := QuerierGetChildTables(connection, tableName)

	var selectFields []string
	var joinConditions []string
	var whereConditions []string
	var groupByFields []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * add selected parent table embeddings and joins
	selectedParents := make(map[string]bool)
	for _, parentTable := range parentTables {
		selectedParents[parentTable] = true
		// * find the column that references this parent table
		for columnName, refTable := range fkRefs {
			if refTable == parentTable {
				refEntityName := util.ToSingular(refTable)
				selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", refEntityName))
				joinConditions = append(joinConditions, fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.id",
					refTable, tableName, columnName, refTable))
				break
			}
		}
	}

	// * add filter conditions for selected parent relations
	for columnName, refTable := range fkRefs {
		if selectedParents[refTable] {
			refEntityName := util.ToSingular(refTable)
			whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
				refEntityName, tableName, columnName, refEntityName))
		}
	}

	// * add child table counts
	for _, childTable := range childTables {
		childEntityName := util.ToSingular(*childTable.Name)
		countField := fmt.Sprintf("%s_count", childEntityName)
		selectFields = append(selectFields, fmt.Sprintf(`(SELECT COALESCE(COUNT(*), 0)::BIGINT FROM %s WHERE %s.%s_id = %s.id) AS %s`,
			*childTable.Name, *childTable.Name, entityName, tableName, countField))
	}

	// * add filter conditions for fields with filter feature
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, tableName, field, field))
	}

	// * build querier name with With suffix
	var withSuffix string
	if len(parentTables) > 0 {
		var parentNames []string
		for _, parent := range parentTables {
			parentNames = append(parentNames, util.ToTitleCase(util.ToSingular(parent)))
		}
		withSuffix = "With" + strings.Join(parentNames, "")
	} else {
		return "" // * skip if no parent tables
	}

	// * build WHERE clause
	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	// * build GROUP BY
	groupByFields = append(groupByFields, fmt.Sprintf("%s.id", tableName))
	for _, parentTable := range parentTables {
		groupByFields = append(groupByFields, fmt.Sprintf("%s.id", parentTable))
	}

	// * build ORDER BY with dynamic sorting using configurable sortable fields
	var orderByFields []string
	for _, field := range tableConfig.SortableFields {
		orderByFields = append(orderByFields, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') = '%s' AND COALESCE(sqlc.narg('order'), 'asc') = 'asc' THEN %s.%s END`,
			field, tableName, field))
		orderByFields = append(orderByFields, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') = '%s' AND sqlc.narg('order') = 'desc' THEN %s.%s END DESC`,
			field, tableName, field))
	}

	// * build final query
	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %s%s%s :many\n", entityTitleCase, "List", withSuffix))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	if whereClause != "" {
		query.WriteString("\n")
		query.WriteString(whereClause)
	}

	if len(groupByFields) > 1 { // * only add GROUP BY if we have joins
		query.WriteString("\nGROUP BY ")
		query.WriteString(strings.Join(groupByFields, ", "))
	}

	query.WriteString("\nORDER BY\n  ")
	query.WriteString(strings.Join(orderByFields, ",\n  "))
	query.WriteString("\nLIMIT sqlc.narg('limit')::BIGINT")
	query.WriteString("\nOFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);")

	return query.String()
}

// * generate One querier with join configuration
func QuerierGenerateOneWithJoin(connection *Connection, entityName string, table *Table, join *ConfigJoin, joinName string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	// * build SELECT fields and JOINs from join configuration
	selectFields, joinConditions, _ := QuerierBuildJoinsFromFields(connection, table, join)

	// * add main table fields
	selectFields = append([]string{fmt.Sprintf("sqlc.embed(%s)", entityName)}, selectFields...)

	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sOne%s :one\n", entityTitleCase, joinName))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	query.WriteString(fmt.Sprintf("\nWHERE %s.id = $1\n", tableName))
	query.WriteString("LIMIT 1;")

	return query.String()
}

// * generate Many querier with join configuration
func QuerierGenerateManyWithJoin(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig, join *ConfigJoin, joinName string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	// * build SELECT fields and JOINs from join configuration
	selectFields, joinConditions, _ := QuerierBuildJoinsFromFields(connection, table, join)

	// * add main table fields
	selectFields = append([]string{fmt.Sprintf("sqlc.embed(%s)", entityName)}, selectFields...)

	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sMany%s :many\n", entityTitleCase, joinName))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	query.WriteString(fmt.Sprintf("\nWHERE %s.id = ANY($1::BIGINT[])", tableName))
	query.WriteString(";")

	return query.String()
}

// * generate List querier with join configuration
func QuerierGenerateListWithJoin(connection *Connection, entityName string, table *Table, tableConfig *QuerierTableConfig, join *ConfigJoin, joinName string) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	// * build SELECT fields and JOINs from join configuration
	selectFields, joinConditions, groupByFields := QuerierBuildJoinsFromFields(connection, table, join)

	// * add main table fields
	selectFields = append([]string{fmt.Sprintf("sqlc.embed(%s)", entityName)}, selectFields...)

	// * build WHERE clause based on filter fields
	var whereConditions []string
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, tableName, field, field))
	}

	// * build WHERE clause
	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	// * build GROUP BY
	if len(groupByFields) == 0 {
		groupByFields = []string{fmt.Sprintf("%s.id", tableName)}
	}

	// * build final query without ORDER BY
	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sList%s :many\n", entityTitleCase, joinName))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", tableName))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	if whereClause != "" {
		query.WriteString("\n")
		query.WriteString(whereClause)
	}

	if len(groupByFields) > 1 { // * only add GROUP BY if we have joins
		query.WriteString("\nGROUP BY ")
		query.WriteString(strings.Join(groupByFields, ", "))
	}

	query.WriteString("\nLIMIT sqlc.narg('limit')::BIGINT")
	query.WriteString("\nOFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);")

	return query.String()
}

// * build JOINs from fields configuration
func QuerierBuildJoinsFromFields(connection *Connection, table *Table, join *ConfigJoin) ([]string, []string, []string) {
	var selectFields []string
	var joinConditions []string
	var groupByFields []string

	if join == nil || len(join.Fields) == 0 {
		return selectFields, joinConditions, groupByFields
	}

	// * track processed joins by source_table.source_column -> dest_table.dest_alias
	processedJoins := make(map[string]string)
	// * track embed names for each table to handle duplicates
	embedNamesByTable := make(map[string][]string)

	// * process all field paths and build joins
	for _, fieldPtr := range join.Fields {
		if fieldPtr == nil {
			continue
		}

		fieldPath := *fieldPtr
		if fieldPath == "" {
			continue
		}

		parts := strings.Split(fieldPath, ".")
		if len(parts) < 1 {
			continue
		}

		// * build join chain for this field path
		currentTable := table
		currentTableName := *table.Name
		pathTables := []string{} // track table names in the path

		// * process each part of the path (starting from first part)
		for i := 0; i < len(parts); i++ {
			columnName := parts[i]

			// * find foreign key constraint from current table using the column name
			var foundConstraint *Constraint
			for _, constraint := range currentTable.Constraints {
				if *constraint.Type == "FOREIGN KEY" && len(constraint.Columns) == 1 {
					if *constraint.Columns[0] == columnName {
						foundConstraint = constraint
						break
					}
				}
			}

			// * if no FK found by column name, try by referenced table name
			if foundConstraint == nil {
				for _, constraint := range currentTable.Constraints {
					if *constraint.Type == "FOREIGN KEY" && len(constraint.Columns) == 1 {
						referencedTable := *constraint.References
						if parenIndex := strings.Index(referencedTable, "("); parenIndex != -1 {
							referencedTable = strings.TrimSpace(referencedTable[:parenIndex])
						}

						if referencedTable == columnName {
							foundConstraint = constraint
							break
						}
					}
				}
			}

			if foundConstraint == nil {
				continue
			}

			// * get next table
			referencedTable := *foundConstraint.References
			if parenIndex := strings.Index(referencedTable, "("); parenIndex != -1 {
				referencedTable = strings.TrimSpace(referencedTable[:parenIndex])
			}

			nextTable, exists := connection.Tables[referencedTable]
			if !exists {
				continue
			}

			// * create join key based on source table and column
			joinColumn := *foundConstraint.Columns[0]
			joinKey := fmt.Sprintf("%s.%s", currentTableName, joinColumn)

			// * check if we've already processed this join
			if existingAlias, exists := processedJoins[joinKey]; exists {
				// * join already exists, reuse the alias
				currentTable = nextTable
				currentTableName = existingAlias
				// * still need to track table for path
				pathTables = append(pathTables, referencedTable)
				continue
			}

			// * add table to path
			pathTables = append(pathTables, referencedTable)

			// * build alias from table names (plural, underscore-separated)
			alias := strings.Join(pathTables, "_")

			// * ensure alias is unique (in case of conflicts)
			if _, exists := processedJoins[joinKey]; exists {
				counter := 1
				originalAlias := alias
				for {
					found := false
					for _, existingAlias := range processedJoins {
						if existingAlias == alias {
							found = true
							break
						}
					}
					if !found {
						break
					}
					alias = fmt.Sprintf("%s_%d", originalAlias, counter)
					counter++
				}
			}

			// * build join condition
			joinCondition := fmt.Sprintf("LEFT JOIN %s %s ON %s.%s = %s.id",
				referencedTable, alias, currentTableName, joinColumn, alias)

			// * record this join
			joinConditions = append(joinConditions, joinCondition)
			processedJoins[joinKey] = alias

			// * track this embed name (use alias directly)
			embedNamesByTable[referencedTable] = append(embedNamesByTable[referencedTable], alias)

			// * add to group by
			groupByFields = append(groupByFields, fmt.Sprintf("%s.id", alias))

			// * move to next table
			currentTable = nextTable
			currentTableName = alias
		}
	}

	// * generate select fields using aliases directly
	seenAliases := make(map[string]bool)
	for _, aliases := range embedNamesByTable {
		for _, alias := range aliases {
			if !seenAliases[alias] {
				selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", alias))
				seenAliases[alias] = true
			}
		}
	}

	return selectFields, joinConditions, groupByFields
}
