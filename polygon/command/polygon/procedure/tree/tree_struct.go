package tree

type Tree struct {
	Path       string
	Template   *Template
	Handlers   map[string]*StructuredHandler
	Procedures map[string]*StructuredProcedure
	Services   map[string]*StructuredService
}

type Template struct {
	MakefileValid bool
	SqlcValid     bool
	ClaudeValid   bool
}

type StructuredHandler struct {
	Name      string
	DependsOn []*DependencyTarget
}

type StructuredService struct {
	Name      string
	DependsOn []*DependencyTarget
}

type DependencyTarget struct {
	Structure Structure
	Name      string
}
