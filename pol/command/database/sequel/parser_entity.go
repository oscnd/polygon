package sequel

import (
	"fmt"
	"strings"
)

type Config struct {
	Connections map[string]*ConfigConnection `yaml:"connections"`
}

type ConfigConnection struct {
	Dialect *string                 `yaml:"dialect"`
	Tables  map[string]*ConfigTable `yaml:"tables"`
}

type ConfigTable struct {
	Fields    []*ConfigField    `yaml:"fields"`
	Additions []*ConfigAddition `yaml:"additions"`
	Joins     []*ConfigJoin     `yaml:"joins,omitempty"`
}

// Field method to retrieve field by name
func (r *ConfigTable) Field(name string) *ConfigField {
	for _, field := range r.Fields {
		if field.Name != nil && *field.Name == name {
			return field
		}
	}
	return nil
}

type ConfigField struct {
	Name    *string   `yaml:"name"`
	Include *string   `yaml:"include"`
	Feature []*string `yaml:"feature,omitempty"`
}

type ConfigAddition struct {
	Name    *string `yaml:"name"`
	Package *string `yaml:"package"`
	Type    *string `yaml:"type"`
}

type ConfigJoin struct {
	Type   *string   `yaml:"type"`   // Must be "parented"
	Table  *string   `yaml:"table"`  // Originate table
	Fields []*string `yaml:"fields"` // Fields with .notation for parent refs
}

type Connection struct {
	Dialect *string
	Tables  map[string]*Table
}

type Table struct {
	Name        *string
	Columns     []*Column
	Indexes     []*Index
	Constraints []*Constraint
}

type Column struct {
	Name        *string
	Type        *string
	Nullable    *bool
	Default     *string
	Constraints []*string
}

type Index struct {
	Name    *string
	Columns []*string
	Unique  *bool
	Type    *string
}

type Constraint struct {
	Name       *string
	Type       *string
	Columns    []*string
	References *string
}

func (r *Table) GenerateStatement() string {
	builder := new(strings.Builder)
	builder.WriteString("CREATE TABLE ")
	builder.WriteString(*r.Name)
	builder.WriteString(" (\n")

	// * track which constraints have been processed inline
	inlineProcessed := make(map[string]bool)

	// * collect remaining constraints that need to be added at table level
	var remainingConstraints []*Constraint

	// * write columns with inline constraints where appropriate
	for i, column := range r.Columns {
		builder.WriteString("    ")
		builder.WriteString(*column.Name)
		builder.WriteString(" ")
		builder.WriteString(*column.Type)

		if column.Nullable != nil && !*column.Nullable {
			builder.WriteString(" NOT NULL")
		} else {
			builder.WriteString(" NULL")
		}

		if column.Default != nil && *column.Default != "" {
			builder.WriteString(" DEFAULT ")
			builder.WriteString(*column.Default)
		}

		// * check for single-column constraints that can be inlined
		for _, constraint := range r.Constraints {
			if len(constraint.Columns) == 1 && *constraint.Columns[0] == *column.Name {
				constraintKey := fmt.Sprintf("%s_%s", *constraint.Type, *column.Name)
				if *constraint.Type == "UNIQUE" {
					builder.WriteString(" UNIQUE")
					inlineProcessed[constraintKey] = true
				} else if *constraint.Type == "FOREIGN KEY" && *constraint.References != "" {
					builder.WriteString(" REFERENCES ")
					builder.WriteString(*constraint.References)
					inlineProcessed[constraintKey] = true
				}
			}
		}

		if i < len(r.Columns)-1 || len(r.Constraints) > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}

	// * collect remaining constraints that not inlined
	for _, constraint := range r.Constraints {
		constraintKey := ""
		if len(constraint.Columns) == 1 {
			constraintKey = fmt.Sprintf("%s_%s", *constraint.Type, *constraint.Columns[0])
		}

		// * skip constraints that already inlined
		if !inlineProcessed[constraintKey] {
			remainingConstraints = append(remainingConstraints, constraint)
		}
	}

	// * write remaining table-level constraints
	for i, constraint := range remainingConstraints {
		switch *constraint.Type {
		case "PRIMARY KEY":
			builder.WriteString("    PRIMARY KEY (")
			for j, col := range constraint.Columns {
				if j > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(*col)
			}
			builder.WriteString(")")
		case "FOREIGN KEY":
			builder.WriteString("    FOREIGN KEY (")
			for j, col := range constraint.Columns {
				if j > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(*col)
			}
			builder.WriteString(") REFERENCES ")
			builder.WriteString(*constraint.References)
		case "UNIQUE":
			builder.WriteString("    UNIQUE (")
			for j, col := range constraint.Columns {
				if j > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(*col)
			}
			builder.WriteString(")")
		}

		// * add comma between constraints
		if i < len(remainingConstraints)-1 {
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}

	builder.WriteString(");")
	return builder.String()
}

type Function struct {
	Name       *string
	Parameters []*string
	Returns    *string
	Body       *string
	Language   *string
}

func (r *Function) GenerateStatement() string {
	return *r.Body
}

type Trigger struct {
	Name       *string
	Table      *string
	Before     *bool
	After      *bool
	InsteadOf  *bool
	Events     []*string
	Function   *string
	ForEachRow *bool
}

func (t *Trigger) GenerateStatement() string {
	return *t.Function
}
