package sample

import (
	"testing"

	"go.scnd.dev/open/polygon/utility/code"
)

func TestNewParser(t *testing.T) {
	// Test creating parser with example module
	examplePath := "../example"
	parser, err := code.NewParser(examplePath)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	if parser.Module == nil {
		t.Fatal("Module should not be nil")
	}

	if parser.Module.Path == nil {
		t.Fatal("Module path should not be nil")
	}

	if parser.Module.Name == nil {
		t.Fatal("Module name should not be nil")
	}

	expectedName := "example"
	if *parser.Module.Name != expectedName {
		t.Errorf("Expected module name %s, got %s", expectedName, *parser.Module.Name)
	}
}

func TestParseModule(t *testing.T) {
	// Test parsing the example module
	examplePath := "../example"
	parser, err := code.NewParser(examplePath)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	err = parser.ParseModule()
	if err != nil {
		t.Fatalf("Failed to parse module: %v", err)
	}

	if len(parser.Module.Packages) == 0 {
		t.Fatal("Expected at least one package")
	}

	// Print packages found for debugging
	for _, pkg := range parser.Module.Packages {
		if pkg.PackageName != nil {
			t.Logf("Found package: %s", *pkg.PackageName)
		}
		if pkg.Files != nil {
			t.Logf("  Files: %d", len(pkg.Files))
		}
	}
}

func TestParseCurrentPackage(t *testing.T) {
	// Test parsing current package (sample)
	currentPath := "."
	parser, err := code.NewParser(currentPath)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	err = parser.ParseModule()
	if err != nil {
		t.Fatalf("Failed to parse current module: %v", err)
	}

	if len(parser.Module.Packages) == 0 {
		t.Fatal("Expected at least one package")
	}

	// Find the sample package
	var samplePkg *code.Package
	for _, pkg := range parser.Module.Packages {
		if pkg.PackageName != nil && *pkg.PackageName == "sample" {
			samplePkg = pkg
			break
		}
	}

	if samplePkg == nil {
		t.Fatal("Sample package not found")
	}

	if len(samplePkg.Files) == 0 {
		t.Fatal("Expected at least one file in sample package")
	}

	// Check for expected structs and interfaces
	var userStruct *code.Struct
	var repositoryInterface *code.Interface
	var serviceStruct *code.Struct
	var functionCount int

	for _, file := range samplePkg.Files {
		// Check structs
		for _, strct := range file.Structs {
			if strct.Name != nil {
				t.Logf("Found struct: %s", *strct.Name)
				if *strct.Name == "User" {
					userStruct = strct
				}
				if *strct.Name == "Service" {
					serviceStruct = strct
				}
			}
		}

		// Check interfaces
		for _, iface := range file.Interfaces {
			if iface.Name != nil {
				t.Logf("Found interface: %s", *iface.Name)
				if *iface.Name == "Repository" {
					repositoryInterface = iface
				}
			}
		}

		// Count functions
		functionCount += len(file.Functions)
		for _, fn := range file.Functions {
			if fn.Name != nil {
				t.Logf("Found function/method: %s", *fn.Name)
			}
		}
	}

	// Validate expected results
	if userStruct == nil {
		t.Error("User struct not found")
	} else {
		if len(userStruct.Fields) == 0 {
			t.Error("User struct should have fields")
		}
		// Check for expected fields
		foundID := false
		foundName := false
		for _, field := range userStruct.Fields {
			if field.Name != nil {
				if *field.Name == "ID" {
					foundID = true
				}
				if *field.Name == "Name" {
					foundName = true
				}
			}
		}
		if !foundID {
			t.Error("User struct missing ID field")
		}
		if !foundName {
			t.Error("User struct missing Name field")
		}
	}

	if repositoryInterface == nil {
		t.Error("Repository interface not found")
	} else {
		if len(repositoryInterface.Methods) == 0 {
			t.Error("Repository interface should have methods")
		}
		// Check for expected methods
		methodNames := make(map[string]bool)
		for _, method := range repositoryInterface.Methods {
			if method.Name != nil {
				methodNames[*method.Name] = true
			}
		}
		if !methodNames["Get"] {
			t.Error("Repository interface missing Get method")
		}
		if !methodNames["Create"] {
			t.Error("Repository interface missing Create method")
		}
	}

	if serviceStruct == nil {
		t.Error("Service struct not found")
	}

	if functionCount == 0 {
		t.Error("Expected at least one function/method")
	}
}

func TestParseFileDirectly(t *testing.T) {
	// Test parsing a specific file directly
	pkg := &code.Package{
		Path:    &[]string{"."}[0],
		Package: &[]string{"sample"}[0],
	}

	file, err := code.ParsePackageFile(pkg, "types.go")
	if err != nil {
		t.Fatalf("Failed to parse types.go: %v", err)
	}

	if file.Name == nil {
		t.Error("File name should not be nil")
	}

	if *file.Name != "types.go" {
		t.Errorf("Expected filename types.go, got %s", *file.Name)
	}

	if len(file.Structs) == 0 {
		t.Error("Expected at least one struct in types.go")
	}

	if len(file.Interfaces) == 0 {
		t.Error("Expected at least one interface in types.go")
	}
}
