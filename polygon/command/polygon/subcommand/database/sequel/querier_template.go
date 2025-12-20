package sequel

import (
	"fmt"
	"strings"

	"go.scnd.dev/open/polygon/utility/form"
)

func QuerierGenerateCreate(connection *Connection, table *Table) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

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
		*table.Name,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
}

func QuerierGenerateOne(connection *Connection, table *Table) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	return fmt.Sprintf(`-- name: %sOne :one
SELECT * FROM %s WHERE %s LIMIT 1;`,
		entityTitleCase,
		*table.Name,
		QuerierGetPrimaryKeyWhereClause(table, 1))
}

func QuerierGenerateOneCounted(connection *Connection, table *Table) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	childTables := QuerierGetChildTables(connection, *table.Name)
	var selectFields []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", *table.Name))

	// * add child table counts
	for _, childTable := range childTables {
		countField := fmt.Sprintf("%s_count", *childTable.SingularName)
		selectFields = append(selectFields, fmt.Sprintf(`(SELECT COALESCE(COUNT(*), 0)::BIGINT FROM %s WHERE %s.%s_id = %s.id) AS %s`,
			*childTable.Name, *childTable.Name, *table.SingularName, *table.Name, countField))
	}

	return fmt.Sprintf(`-- name: %sOneCounted :one
SELECT %s
FROM %s
WHERE %s
LIMIT 1;`,
		entityTitleCase,
		strings.Join(selectFields, ",\n       "),
		*table.Name,
		QuerierGetPrimaryKeyWhereClauseWithTable(table, 1, *table.Name))
}

func QuerierGenerateManyCounted(connection *Connection, table *Table) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	childTables := QuerierGetChildTables(connection, *table.Name)
	var selectFields []string

	// * add main table fields
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", *table.Name))

	// * add child table counts
	for _, childTable := range childTables {
		countField := fmt.Sprintf("%s_count", *childTable.SingularName)
		selectFields = append(selectFields, fmt.Sprintf(`(SELECT COALESCE(COUNT(*), 0)::BIGINT FROM %s WHERE %s.%s_id = %s.id) AS %s`,
			*childTable.Name, *childTable.Name, *table.SingularName, *table.Name, countField))
	}

	return fmt.Sprintf(`-- name: %sManyCounted :many
SELECT %s
FROM %s
WHERE %s;`,
		entityTitleCase,
		strings.Join(selectFields, ",\n       "),
		*table.Name,
		QuerierGetPrimaryKeyWhereInClause(table, 1))
}

func QuerierGenerateMany(connection *Connection, table *Table, tableConfig *QuerierTableConfig) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	return fmt.Sprintf(`-- name: %sMany :many
SELECT * FROM %s WHERE %s;`,
		entityTitleCase,
		*table.Name,
		QuerierGetPrimaryKeyWhereInClause(table, 1))
}

func QuerierGenerateCount(connection *Connection, table *Table, tableConfig *QuerierTableConfig) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	fkRefs := QuerierGetForeignKeyReferences(table)
	var whereConditions []string

	// * add filter conditions for parent relations
	for columnName, refTable := range fkRefs {
		refEntityName := form.ToSingular(refTable)
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			refEntityName, *table.Name, columnName, refEntityName))
	}

	// * add filter conditions for fields with filter feature
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, *table.Name, field, field))
	}

	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	query := fmt.Sprintf(`-- name: %sCount :one
SELECT COALESCE(COUNT(*), 0)::BIGINT AS %s_count
FROM %s`,
		entityTitleCase,
		*table.SingularName,
		*table.Name)

	if whereClause != "" {
		query += "\n" + whereClause
	}

	query += ";"

	return query
}

func QuerierGenerateIncrease(connection *Connection, table *Table, fieldName string) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)
	fieldTitleCase := form.ToPascalCase(fieldName)

	return fmt.Sprintf(`-- name: %s%sIncrease :one
UPDATE %s
SET %s = COALESCE(%s, 0) + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE %s
RETURNING *;`,
		entityTitleCase,
		fieldTitleCase,
		*table.Name,
		fieldName,
		fieldName,
		QuerierGetPrimaryKeyWhereClauseWithTable(table, 1, *table.Name))
}

func QuerierGenerateList(connection *Connection, table *Table, tableConfig *QuerierTableConfig) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	// * List querier does NOT have joins, only main table
	var selectFields []string
	selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", *table.Name))

	// * build WHERE clause based on filter fields
	var whereConditions []string
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, *table.Name, field, field))
	}

	// * build WHERE clause
	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	// * build ORDER BY clause based on sort fields with dynamic sorting
	var orderByClause string
	if len(tableConfig.SortableFields) > 0 {
		var orderConditions []string
		for _, field := range tableConfig.SortableFields {
			// * add CASE statements for dynamic sorting with configurable direction
			orderConditions = append(orderConditions, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') = '%s' AND COALESCE(sqlc.narg('order'), 'asc') = 'asc' THEN %s.%s END`,
				field, *table.Name, field))
			orderConditions = append(orderConditions, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') = '%s' AND sqlc.narg('order') = 'desc' THEN %s.%s END DESC`,
				field, *table.Name, field))
		}

		// * add default sort by first sortable field if no sort parameter provided
		if len(tableConfig.SortableFields) > 0 {
			firstField := tableConfig.SortableFields[0]
			orderConditions = append(orderConditions, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') IS NULL THEN %s.%s END`,
				*table.Name, firstField))
		}

		orderByClause = "ORDER BY " + strings.Join(orderConditions, ",\n      ")
	}

	// * build final query
	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sList :many\n", entityTitleCase))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", *table.Name))

	if whereClause != "" {
		query.WriteString("\n")
		query.WriteString(whereClause)
	}

	if orderByClause != "" {
		query.WriteString("\n")
		query.WriteString(orderByClause)
	}

	query.WriteString("\nLIMIT sqlc.narg('limit')::BIGINT")
	query.WriteString("\nOFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);")

	return query.String()
}

func QuerierGenerateUpdate(connection *Connection, table *Table) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

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

	// * build where clause with primary key
	whereClause := QuerierGetPrimaryKeyWhereClauseForUpdate(table)

	return fmt.Sprintf(`-- name: %sUpdate :one
UPDATE %s
SET %s
WHERE %s
RETURNING *;`,
		entityTitleCase,
		*table.Name,
		strings.Join(setConditions, ",\n    "),
		whereClause)
}

func QuerierGenerateDelete(connection *Connection, table *Table) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	return fmt.Sprintf(`-- name: %sDelete :one
DELETE FROM %s WHERE %s RETURNING *;`,
		entityTitleCase,
		*table.Name,
		QuerierGetPrimaryKeyWhereClause(table, 1))
}

func QuerierGenerateOneWithJoin(connection *Connection, table *Table, join *ConfigJoin, joinName string) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	// * build SELECT fields and JOINs from join configuration
	selectFields, joinConditions, _ := QuerierBuildJoinsFromFields(connection, table, join)

	// * add main table fields
	selectFields = append([]string{fmt.Sprintf("sqlc.embed(%s)", *table.Name)}, selectFields...)

	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sOne%s :one\n", entityTitleCase, joinName))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", *table.Name))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	query.WriteString(fmt.Sprintf("\nWHERE %s\n", QuerierGetPrimaryKeyWhereClauseWithTable(table, 1, *table.Name)))
	query.WriteString("LIMIT 1;")

	return query.String()
}

func QuerierGenerateManyWithJoin(connection *Connection, table *Table, _ *QuerierTableConfig, join *ConfigJoin, joinName string) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	// * build SELECT fields and JOINs from join configuration
	selectFields, joinConditions, _ := QuerierBuildJoinsFromFields(connection, table, join)

	// * add main table fields
	selectFields = append([]string{fmt.Sprintf("sqlc.embed(%s)", *table.Name)}, selectFields...)

	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sMany%s :many\n", entityTitleCase, joinName))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", *table.Name))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	query.WriteString(fmt.Sprintf("\nWHERE %s", QuerierGetPrimaryKeyWhereInClause(table, 1)))
	query.WriteString(";")

	return query.String()
}

func QuerierGenerateListWithJoin(connection *Connection, table *Table, tableConfig *QuerierTableConfig, join *ConfigJoin, joinName string) string {
	entityTitleCase := form.ToPascalCase(*table.SingularName)

	// * build SELECT fields and JOINs from join configuration
	selectFields, joinConditions, groupByFields := QuerierBuildJoinsFromFields(connection, table, join)

	// * add main table fields
	selectFields = append([]string{fmt.Sprintf("sqlc.embed(%s)", *table.Name)}, selectFields...)

	// * build WHERE clause based on filter fields
	var whereConditions []string
	for _, field := range tableConfig.FilterFields {
		whereConditions = append(whereConditions, fmt.Sprintf(`(sqlc.narg('%s_ids')::BIGINT[] IS NULL OR %s.%s = ANY(sqlc.narg('%s_ids')::BIGINT[]))`,
			field, *table.Name, field, field))
	}

	// * build WHERE clause
	var whereClause string
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, "\n  AND ")
	}

	// * build ORDER BY clause based on sort fields
	var orderByClause string
	if len(tableConfig.SortableFields) > 0 {
		var orderConditions []string
		for _, field := range tableConfig.SortableFields {
			// * add CASE statements for dynamic sorting with configurable direction
			orderConditions = append(orderConditions, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') = '%s' AND COALESCE(sqlc.narg('order'), 'asc') = 'asc' THEN %s.%s END`,
				field, *table.Name, field))
			orderConditions = append(orderConditions, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') = '%s' AND sqlc.narg('order') = 'desc' THEN %s.%s END DESC`,
				field, *table.Name, field))
		}

		// * add default sort by first sortable field if no sort parameter provided
		if len(tableConfig.SortableFields) > 0 {
			firstField := tableConfig.SortableFields[0]
			orderConditions = append(orderConditions, fmt.Sprintf(`CASE WHEN sqlc.narg('sort') IS NULL THEN %s.%s END`,
				*table.Name, firstField))
		}

		orderByClause = "ORDER BY " + strings.Join(orderConditions, ",\n      ")
	}

	// * build GROUP BY
	if len(groupByFields) == 0 {
		groupByFields = []string{fmt.Sprintf("%s.id", *table.Name)}
	}

	// * build final query
	var query strings.Builder
	query.WriteString(fmt.Sprintf("-- name: %sList%s :many\n", entityTitleCase, joinName))
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(selectFields, ",\n       "))
	query.WriteString(fmt.Sprintf("\nFROM %s", *table.Name))

	if len(joinConditions) > 0 {
		query.WriteString("\n")
		query.WriteString(strings.Join(joinConditions, "\n"))
	}

	if whereClause != "" {
		query.WriteString("\n")
		query.WriteString(whereClause)
	}

	if len(groupByFields) > 1 {
		query.WriteString("\nGROUP BY ")
		query.WriteString(strings.Join(groupByFields, ", "))
	}

	if orderByClause != "" {
		query.WriteString("\n")
		query.WriteString(orderByClause)
	}

	query.WriteString("\nLIMIT sqlc.narg('limit')::BIGINT")
	query.WriteString("\nOFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);")

	return query.String()
}

func QuerierBuildJoinsFromFields(connection *Connection, table *Table, join *ConfigJoin) ([]string, []string, []string) {
	var selectFields []string
	var groupByFields []string

	if join == nil || len(join.Fields) == 0 {
		return selectFields, []string{}, groupByFields
	}

	// * track processed joins by source_table.source_column -> dest_table.dest_alias
	processedJoins := make(map[string]string)
	// * track embed names for each table to handle duplicates
	embedNamesByTable := make(map[string][]string)
	// * track joins by table name to group them
	joinsByTable := make(map[string][]string)

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
		var pathTables []string

		// * process each part of the path
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
				pathTables = append(pathTables, referencedTable)
				continue
			}

			// * add table to path
			pathTables = append(pathTables, referencedTable)

			// * build alias
			alias := "joined_" + strings.Join(pathTables, "_")

			// * ensure alias is unique
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

			// * group joins by referenced table
			joinsByTable[referencedTable] = append(joinsByTable[referencedTable], joinCondition)

			// * record this join
			processedJoins[joinKey] = alias

			// * track this embed name
			embedNamesByTable[referencedTable] = append(embedNamesByTable[referencedTable], alias)

			// * add to group by
			groupByFields = append(groupByFields, fmt.Sprintf("%s.id", alias))

			// * move to next table
			currentTable = nextTable
			currentTableName = alias
		}
	}

	// * build join conditions
	var joinConditions []string
	for _, tableJoins := range joinsByTable {
		joinConditions = append(joinConditions, tableJoins...)
	}

	// * generate select fields using aliases for sqlc.embed
	for _, aliases := range embedNamesByTable {
		if len(aliases) > 0 {
			for _, alias := range aliases {
				selectFields = append(selectFields, fmt.Sprintf("sqlc.embed(%s)", alias))
			}
		}
	}

	return selectFields, joinConditions, groupByFields
}
