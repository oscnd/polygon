package erroring

import "context"

type Dimension struct {
	Context    context.Context `json:"context,omitempty"`
	Scope      *string         `json:"scope,omitempty"`
	Type       *string         `json:"type,omitempty"`
	Arguments  map[string]any  `json:"arguments,omitempty"`
	Parameters map[string]any  `json:"parameters,omitempty"`
}

func NewDimension(context context.Context, scope string, arguments map[string]any) *Dimension {
	return &Dimension{
		Context:    context,
		Scope:      &scope,
		Type:       nil,
		Arguments:  arguments,
		Parameters: make(map[string]any),
	}
}

func (r *Dimension) WithType(dimensionType string) *Dimension {
	r.Type = &dimensionType
	return r
}

type DimensionType string

const (
	DimensionTypeTimeout    DimensionType = "timeout"
	DimensionTypeValidation DimensionType = "validation"
	DimensionTypeFatal      DimensionType = "fatal"
	DimensionTypeOperation  DimensionType = "operation"
	DimensionTypeOverflow   DimensionType = "overflow"
)

type DimensionScope string

const (
	DimensionScopeConfig    DimensionScope = "config"
	DimensionScopeLibrary   DimensionScope = "library"
	DimensionScopeDatabase  DimensionScope = "database"
	DimensionScopeProcedure DimensionScope = "procedure"
	DimensionScopeExternal  DimensionScope = "external"
)

type DimensionPlace string

const (
	DimensionPlacePolygon DimensionPlace = "polygon"
)
