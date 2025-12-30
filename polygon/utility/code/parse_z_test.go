package code

import (
	"testing"
)

func TestParserWithExampleModule(t *testing.T) {
	// Test parsing the example module
	examplePath := "../../../example"
	parser, err := NewParser(examplePath)
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

	t.Logf("Successfully created parser for module: %s", *parser.Module.Name)
}

func TestParseExampleModule(t *testing.T) {
	// Test parsing the example module completely
	examplePath := "../../../example"
	parser, err := NewParser(examplePath)
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

	t.Logf("Found %d packages in module", len(parser.Module.Packages))

	// Print packages found for debugging
	for _, pkg := range parser.Module.Packages {
		if pkg.PackageName != nil {
			t.Logf("Found package: %s (path: %s)", *pkg.PackageName, safeString(pkg.Path))
		}
		if pkg.Files != nil {
			t.Logf("  Files: %d", len(pkg.Files))
		}
	}
}

func TestParseSamplePackage(t *testing.T) {
	// Test parsing the sample package specifically
	examplePath := "../../../example"
	parser, err := NewParser(examplePath)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	err = parser.ParseModule()
	if err != nil {
		t.Fatalf("Failed to parse module: %v", err)
	}

	// Find the sample package
	var samplePkg *Package
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

	t.Logf("Found sample package with %d files", len(samplePkg.Files))

	// Check for expected structs and interfaces
	var userStruct *Struct
	var repositoryInterface *Interface
	var serviceStruct *Struct
	var functionCount int

	for _, file := range samplePkg.Files {
		t.Logf("Processing file: %s", safeString(file.Name))

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

		// Count functions and methods
		functionCount += len(file.Functions)
		for _, fn := range file.Functions {
			if fn.Name != nil {
				t.Logf("Found function/method: %s", *fn.Name)
			}
		}

		// Check receivers (methods with receivers)
		for _, recv := range file.Receivers {
			if recv.Method != nil && recv.Method.Name != nil {
				t.Logf("Found method with receiver: %s", *recv.Method.Name)
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
		foundEmail := false
		for _, field := range userStruct.Fields {
			if field.Name != nil {
				t.Logf("User field: %s (type: %s)", *field.Name, safeString(field.Type))
				if *field.Name == "ID" {
					foundID = true
				}
				if *field.Name == "Name" {
					foundName = true
				}
				if *field.Name == "Email" {
					foundEmail = true
				}
			}
		}
		if !foundID {
			t.Error("User struct missing ID field")
		}
		if !foundName {
			t.Error("User struct missing Name field")
		}
		if !foundEmail {
			t.Error("User struct missing Email field")
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
				t.Logf("Repository method: %s", *method.Name)
				methodNames[*method.Name] = true
			}
		}
		if !methodNames["Get"] {
			t.Error("Repository interface missing Get method")
		}
		if !methodNames["Create"] {
			t.Error("Repository interface missing Create method")
		}
		if !methodNames["List"] {
			t.Error("Repository interface missing List method")
		}
	}

	if serviceStruct == nil {
		t.Error("Service struct not found")
	}

	if functionCount == 0 {
		t.Error("Expected at least one function/method")
	}

	t.Logf("Validation complete: %d functions/methods found", functionCount)
}

func TestParseFileDirectly(t *testing.T) {
	// Test parsing a specific file directly from sample package
	samplePath := "../../../example/sample"
	pkg := &Package{
		Path:    &samplePath,
		Package: &[]string{"sample"}[0],
	}

	file, err := ParsePackageFile(pkg, samplePath+"/types.go")
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

	t.Logf("Successfully parsed types.go: %d structs, %d interfaces", len(file.Structs), len(file.Interfaces))
}

func TestParseServiceFile(t *testing.T) {
	// Test parsing service.go specifically
	samplePath := "../../../example/sample"
	pkg := &Package{
		Path:    &samplePath,
		Package: &[]string{"sample"}[0],
	}

	file, err := ParsePackageFile(pkg, samplePath+"/service.go")
	if err != nil {
		t.Fatalf("Failed to parse service.go: %v", err)
	}

	if len(file.Structs) == 0 {
		t.Error("Expected at least one struct in service.go")
	}

	if len(file.Functions) == 0 {
		t.Error("Expected at least one function in service.go")
	}

	// Check for receiver methods
	if len(file.Receivers) == 0 {
		t.Error("Expected at least one receiver method in service.go")
	}

	t.Logf("Successfully parsed service.go: %d structs, %d functions, %d receivers",
		len(file.Structs), len(file.Functions), len(file.Receivers))
}

// Helper function to safely dereference string pointers
func safeString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
