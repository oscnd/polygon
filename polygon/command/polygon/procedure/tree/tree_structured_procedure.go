package tree

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon"
)

const (
	StructuredProcedureDirectoryName = "procedure"
	StructuredProcedureInitializer   = "procedure.go"
)

type StructuredProcedure struct {
	Name      string
	DependsOn []*DependencyTarget
}

func ParseStructuredProcedure(ctx context.Context) (map[string]*StructuredProcedure, error) {
	_, ctx = polygon.With(ctx)

	parser := new(StructuredProcedureParser)
	return parser.Parse(ctx)
}

type StructuredProcedureParser struct {
	Procedures map[string]*StructuredProcedure
}

func (r *StructuredProcedureParser) Parse(ctx context.Context) (map[string]*StructuredProcedure, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	// * reset procedures map
	r.Procedures = make(map[string]*StructuredProcedure)

	// * read the contents of the root procedure directory
	entries, err := os.ReadDir(StructuredProcedureDirectoryName)
	if err != nil {
		if os.IsNotExist(err) {
			return r.Procedures, nil
		}

		return nil, s.Error("failed to read procedure directory", err)
	}

	for _, entry := range entries {
		if err := r.ParseEntry(ctx, entry); err != nil {
			return nil, err
		}
	}

	return r.Procedures, nil
}

func (r *StructuredProcedureParser) ParseEntry(ctx context.Context, entry os.DirEntry) error {
	// * start span
	s, ctx := polygon.With(ctx)
	s.Variable("name", entry.Name())
	defer s.End()

	// * construct path
	path := filepath.Join(StructuredProcedureDirectoryName, entry.Name())

	// * check for file in root procedure
	if !entry.IsDir() {
		return s.Error("file is not in structural procedure tree", nil)
	}

	// * check for initializer
	procPath := filepath.Join(path, StructuredProcedureInitializer)
	if _, err := os.Stat(procPath); err != nil {
		if os.IsNotExist(err) {
			return s.Error("procedure is missing initializer file", nil)
		}
		return s.Error("failed to stat procedure initializer file", err)
	}

	// * check contents of the procedure subdirectory
	subEntries, err := os.ReadDir(path)
	if err != nil {
		return s.Error("failed to read procedure subdirectory", err)
	}

	for _, subEntry := range subEntries {
		subName := subEntry.Name()
		s.Variable("subentry", subName)

		// * skip initializer
		if subName == StructuredProcedureInitializer {
			continue
		}

		// * check if sub-entry is a directory
		if subEntry.IsDir() {
			return s.Error("subentry is directory", nil)
		}

		// * check for the file prefix
		if !strings.HasPrefix(subName, "proc_") {
			return s.Error("subentry file does not have 'proc_' prefix", nil)
		}
	}

	// * assign procedure
	r.Procedures[entry.Name()] = &StructuredProcedure{
		Name:      entry.Name(),
		DependsOn: []*DependencyTarget{},
	}

	return nil
}
