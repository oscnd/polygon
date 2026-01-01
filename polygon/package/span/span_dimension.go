package span

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
