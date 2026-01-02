package tree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon/package/span"
)

const (
	StructuredProcedureDirectoryName = "procedure"
	StructuredProcedureInitializer   = "procedure.go"
)

type StructuredProcedure struct {
	Name      string
	DependsOn []*DependencyTarget
}

type StructuredProcedureParser struct {
	Procedures map[string]*StructuredProcedure
}

func (r *StructuredProcedureParser) Parse() (map[string]*StructuredProcedure, error) {
	// * reset procedures map
	r.Procedures = make(map[string]*StructuredProcedure)

	// * read the contents of the root procedure directory
	entries, err := os.ReadDir(StructuredProcedureDirectoryName)
	if err != nil {
		if os.IsNotExist(err) {
			return r.Procedures, nil
		}

		return nil, span.NewError(nil, "failed to read procedure directory", err)
	}

	for _, entry := range entries {
		if err := r.ParseEntry(entry); err != nil {
			return nil, err
		}
	}

	return r.Procedures, nil
}

func (r *StructuredProcedureParser) ParseEntry(entry os.DirEntry) error {
	path := filepath.Join(StructuredProcedureDirectoryName, entry.Name())

	// * check for file in root procedure
	if !entry.IsDir() {
		return span.NewError(nil, fmt.Sprintf("file is not in structural procedure tree", entry.Name()), nil)
	}

	// * check for initializer
	procPath := filepath.Join(path, StructuredProcedureInitializer)
	if _, err := os.Stat(procPath); err != nil {
		if os.IsNotExist(err) {
			return span.NewError(nil, fmt.Sprintf("procedure \"%s\" is missing initializer file", entry.Name()), nil)
		}
		return span.NewError(nil, fmt.Sprintf("failed to access initializer file of procedure \"%s\"", entry.Name()), err)
	}

	// * check contents of the procedure subdirectory
	subEntries, err := os.ReadDir(path)
	if err != nil {
		return span.NewError(nil, fmt.Sprintf("failed to read contents of procedure \"%s\" directory", entry.Name()), err)
	}

	for _, subEntry := range subEntries {
		subName := subEntry.Name()

		// * skip initializer
		if subName == StructuredProcedureInitializer {
			continue
		}

		// * check if sub-entry is a directory
		if subEntry.IsDir() {
			return fmt.Errorf("directory \"%s\" is not in a procedure \"%s\" structural tree", subName, entry.Name())
		}

		// * check for the file prefix
		if !strings.HasPrefix(subName, "proc_") {
			return fmt.Errorf("file \"%s\" is not in a procedure \"%s\" structural tree", subName, entry.Name())
		}
	}

	// * assign procedure
	r.Procedures[entry.Name()] = &StructuredProcedure{
		Name:      entry.Name(),
		DependsOn: []*DependencyTarget{},
	}

	return nil
}
