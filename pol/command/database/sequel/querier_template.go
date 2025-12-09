package sequel

import (
	"fmt"
	"strings"

	"go.scnd.dev/polygon/polygon/util"
)

func QuerierGenerateCreate(entityName string, table *Table, connection *Connection) string {
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
func QuerierGenerateOne(entityName string, table *Table, connection *Connection) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %sOne :one
SELECT * FROM %s WHERE id = $1 LIMIT 1;`,
		entityTitleCase,
		tableName)
}

// QuerierGenerateOneCounted generates the OneCounted querier with child relation counts
func QuerierGenerateOneCounted(entityName string, table *Table, connection *Connection) string {
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

// QuerierGenerateMany generates the Many querier with IN filter
func QuerierGenerateMany(entityName string, table *Table, connection *Connection, tableConfig *QuerierTableConfig) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %sMany :many
SELECT * FROM %s WHERE id = ANY($1::BIGINT[]);`,
		entityTitleCase,
		tableName)
}

// QuerierGenerateCount generates the Count querier with same conditions as List
func QuerierGenerateCount(entityName string, table *Table, connection *Connection, tableConfig *QuerierTableConfig) string {
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
func QuerierGenerateIncrease(entityName string, table *Table, fieldName string) string {
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

func QuerierGenerateList(entityName string, table *Table, connection *Connection, tableConfig *QuerierTableConfig) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	// * get foreign key references
	fkRefs := QuerierGetForeignKeyReferences(table)

	// * get child tables for counts
	childTables := QuerierGetChildTables(connection, tableName)

	var selectFields []string
	var joinConditions []string
	var whereConditions []string
	var groupByFields []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", entityName))

	// * add parent table embeddings and joins
	for columnName, refTable := range fkRefs {
		refEntityName := util.ToSingular(refTable)
		selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", refEntityName))
		joinConditions = append(joinConditions, fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.id",
			refTable, tableName, columnName, refTable))

		// * add filter condition for parent relation
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			refEntityName, tableName, columnName, refEntityName))
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

	// * build WHERE clause
	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	// * build GROUP BY
	groupByFields = append(groupByFields, fmt.Sprintf("%s.id", tableName))
	for _, refTable := range fkRefs {
		groupByFields = append(groupByFields, fmt.Sprintf("%s.id", refTable))
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
	query.WriteString(fmt.Sprintf("-- name: %sList :many\n", entityTitleCase))
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

func QuerierGenerateUpdate(entityName string, table *Table, connection *Connection) string {
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

func QuerierGenerateDelete(entityName string, table *Table, connection *Connection) string {
	entityTitleCase := util.ToTitleCase(entityName)
	tableName := fmt.Sprintf("%ss", entityName)

	return fmt.Sprintf(`-- name: %sDelete :one
DELETE FROM %s WHERE id = $1 RETURNING *;`,
		entityTitleCase,
		tableName)
}

// QuerierGenerateOneWith generates One queriers with specific parent combinations
func QuerierGenerateOneWith(entityName string, table *Table, connection *Connection, parentTables []string) string {
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
func QuerierGenerateManyWith(entityName string, table *Table, connection *Connection, tableConfig *QuerierTableConfig, parentTables []string) string {
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
func QuerierGenerateListWith(entityName string, table *Table, connection *Connection, tableConfig *QuerierTableConfig, parentTables []string) string {
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
