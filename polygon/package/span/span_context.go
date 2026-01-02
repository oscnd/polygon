package span

type ContextKey struct {
	Name string
}

var (
	ContextKeySpan = ContextKey{
		Name: "polygon.span",
	}
)
