package inter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon/command/polygon/index"
	"go.scnd.dev/open/polygon/utility/code"
	"go.scnd.dev/open/polygon/utility/form"
	"gopkg.in/yaml.v3"
)

// Config represents the interface.yml configuration
type Config struct {
	Scans []ScanConfig `yaml:"scans"`
}

// ScanConfig represents a single scan configuration
type ScanConfig struct {
	ScanDir               string `yaml:"scan_dir"`
	ScanReceiverType      string `yaml:"scan_receiver_type"`
	GenerateInterfaceName string `yaml:"generate_interface_name"`
	Recursive             bool   `yaml:"recursive"`
}

// MethodInfo stores information about a receiver method
type MethodInfo struct {
	Name          string
	Receiver      string
	Params        []string
	Returns       []string
	FullSignature string
}

// InterfaceInfo stores information about a generated interface
type InterfaceInfo struct {
	Name    string
	Methods []MethodInfo
	Imports map[string]string // map of alias -> import path
}

// Generator handles interface generation
type Generator struct {
	App        index.App
	Config     *Config
	Interfaces []InterfaceInfo
}

// InterfaceNewGenerator creates a new interface generator
func InterfaceNewGenerator(app index.App) (*Generator, error) {
	g := &Generator{
		App:        app,
		Config:     nil,
		Interfaces: make([]InterfaceInfo, 0),
	}

	// * load interface.yml configuration
	configPath := filepath.Join(*app.Directory(), "interface.yml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load interface.yml: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse interface.yml: %w", err)
	}
	g.Config = cfg

	return g, nil
}

// InterfaceGenerate is the main entry point for interface generation
func InterfaceGenerate(app index.App) error {
	// * create generator
	generator, err := InterfaceNewGenerator(app)
	if err != nil {
		return err
	}

	// * scan and collect interfaces
	if err := generator.InterfaceScan(); err != nil {
		return err
	}

	if len(generator.Interfaces) == 0 {
		log.Printf("no interfaces found")
		return nil
	}

	// * generate files
	if err := generator.InterfaceFile(); err != nil {
		return err
	}

	if err := generator.InterfaceBindFile(); err != nil {
		return err
	}

	log.Printf("successfully generated %d interfaces", len(generator.Interfaces))
	return nil
}

// InterfaceScan scans directories and collects interface information
func (r *Generator) InterfaceScan() error {
	for _, scanCfg := range r.Config.Scans {
		log.Printf("scanning %s for %s receivers...", scanCfg.ScanDir, scanCfg.ScanReceiverType)

		scanInterfaces, err := r.InterfaceScanDirectory(scanCfg)
		if err != nil {
			log.Printf("error scanning %s: %v", scanCfg.ScanDir, err)
			continue
		}

		r.Interfaces = append(r.Interfaces, scanInterfaces...)
		log.Printf("found %d interfaces in %s", len(scanInterfaces), scanCfg.ScanDir)
	}

	return nil
}

// InterfaceScanDirectory scans a directory for receiver methods based on scan configuration
func (r *Generator) InterfaceScanDirectory(scanCfg ScanConfig) ([]InterfaceInfo, error) {
	// * group methods and paths by directory name (packageName)
	packageMethods := make(map[string][]MethodInfo)
	packagePaths := make(map[string]string)

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// * skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// * extract directory name as packageName
		dirPath := filepath.Dir(path)
		packageName := filepath.Base(dirPath)

		// * skip if not recursive and not in the immediate directory
		if !scanCfg.Recursive {
			scanDirAbs, _ := filepath.Abs(scanCfg.ScanDir)
			dirPathAbs, _ := filepath.Abs(dirPath)
			if scanDirAbs != dirPathAbs {
				return nil
			}
		}

		// * store the package path
		packagePaths[packageName] = dirPath

		// * parse the Go file for methods
		methods, err := r.InterfaceExtractReceiverMethods(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// * collect methods for this package
		packageMethods[packageName] = append(packageMethods[packageName], methods...)

		return nil
	}

	if err := filepath.Walk(scanCfg.ScanDir, walkFunc); err != nil {
		return nil, err
	}

	// * create interfaces for each package
	var interfaces []InterfaceInfo
	for packageName, allMethods := range packageMethods {
		// * filter methods for specified receiver type
		var filteredMethods []MethodInfo
		for _, method := range allMethods {
			if method.Receiver == scanCfg.ScanReceiverType {
				filteredMethods = append(filteredMethods, method)
			}
		}

		if len(filteredMethods) > 0 {
			// * create interface name by replacing {{ structName }} with package name
			structName := form.ToPascalCase(packageName)
			interfaceName := strings.ReplaceAll(scanCfg.GenerateInterfaceName, "{{ structName }}", structName)

			// * extract required imports from the package directory
			imports := r.InterfaceExtractImportsFromFiles(packagePaths[packageName])

			interfaces = append(interfaces, InterfaceInfo{
				Name:    interfaceName,
				Methods: filteredMethods,
				Imports: imports,
			})
		}
	}

	return interfaces, nil
}

// InterfaceExtractImportsFromFiles analyzes all Go files in package directories and extracts required imports
func (r *Generator) InterfaceExtractImportsFromFiles(packagePath string) map[string]string {
	imports := make(map[string]string)

	// * first, build a mapping of import paths to actual package names
	importPathToPkgName := make(map[string]string)

	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// * only process .go files (skip test files)
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// * parse the Go file to get both imports and package name
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly|parser.ImportsOnly)
		if err != nil {
			return nil // skip files that can't be parsed
		}

		// * extract package name
		if node.Name != nil {
			pkgName := node.Name.Name
			importPathToPkgName[pkgName] = pkgName
		}

		// * extract all imports and map to their actual package names
		for _, imp := range node.Imports {
			// * remove quotes from import path
			importPath := strings.Trim(imp.Path.Value, `"`)

			// * determine the actual package name from the imported package
			// * we need to read the package's source files to get the real package name
			pkgName := r.getPackageNameFromImportPath(importPath)

			// * determine alias (for imports with explicit aliases)
			var alias string
			if imp.Name != nil {
				// * explicit alias (e.g., `import alias "path"`)
				alias = imp.Name.Name
			} else {
				// * use the actual package name
				alias = pkgName
			}

			imports[alias] = importPath
		}

		return nil
	})

	if err != nil {
		// * if we can't walk the path, return empty imports
		return make(map[string]string)
	}

	return imports
}

// getPackageNameFromImportPath retrieves the actual package name from an import path
// by reading the package's source files
func (r *Generator) getPackageNameFromImportPath(importPath string) string {
	// * check if it's a standard library package
	if !strings.Contains(importPath, ".") {
		parts := strings.Split(importPath, "/")
		return parts[len(parts)-1]
	}

	// * for local packages, try to find the package directory
	// * build the absolute path relative to the current working directory
	pkgDir := filepath.Join(r.App.ProjectRoot(), importPath)

	// * walk the directory to find a .go file and extract the package name
	var pkgName string
	filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || pkgName != "" {
			return nil
		}

		// * only process .go files (skip test files)
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// * parse just the package clause
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil {
			return nil
		}

		if node.Name != nil {
			pkgName = node.Name.Name
			return filepath.SkipAll // we found it, stop walking
		}

		return nil
	})

	// * fallback to last part of path if we couldn't determine the package name
	if pkgName == "" {
		parts := strings.Split(importPath, "/")
		pkgName = parts[len(parts)-1]
	}

	return pkgName
}

// InterfaceExtractReceiverMethods parses a Go file and extracts all receiver methods
func (r *Generator) InterfaceExtractReceiverMethods(filePath string) ([]MethodInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var methods []MethodInfo

	// * walk through AST to find function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			return true
		}

		// * extract receiver information
		recv := funcDecl.Recv.List[0]
		recvType := ""
		switch t := recv.Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				recvType = "*" + ident.Name
			}
		case *ast.Ident:
			recvType = t.Name
		}

		// * extract parameters
		var params []string
		for _, param := range funcDecl.Type.Params.List {
			paramType := code.ExprToString(param.Type)
			if len(param.Names) > 1 {
				// * multiple parameters with same type
				for _, name := range param.Names {
					params = append(params, fmt.Sprintf("%s %s", name.Name, paramType))
				}
			} else if len(param.Names) == 1 {
				params = append(params, fmt.Sprintf("%s %s", param.Names[0].Name, paramType))
			} else {
				params = append(params, paramType)
			}
		}

		// * extract return values
		var returns []string
		if funcDecl.Type.Results != nil {
			for _, result := range funcDecl.Type.Results.List {
				resultType := code.ExprToString(result.Type)
				if len(result.Names) > 1 {
					// * multiple named return values with same type
					for _, name := range result.Names {
						returns = append(returns, fmt.Sprintf("%s %s", name.Name, resultType))
					}
				} else if len(result.Names) == 1 {
					returns = append(returns, fmt.Sprintf("%s %s", result.Names[0].Name, resultType))
				} else {
					returns = append(returns, resultType)
				}
			}
		}

		// * build full signature
		signature := fmt.Sprintf("%s(%s)", funcDecl.Name.Name, strings.Join(params, ", "))
		if len(returns) > 0 {
			signature += " (" + strings.Join(returns, ", ") + ")"
		}

		methods = append(methods, MethodInfo{
			Name:          funcDecl.Name.Name,
			Receiver:      recvType,
			Params:        params,
			Returns:       returns,
			FullSignature: signature,
		})

		return true
	})

	return methods, nil
}

// InterfaceFile generates the interface.go file
func (r *Generator) InterfaceFile() error {
	log.Printf("generating interface.go...")

	generateDir := filepath.Join("generate", "polygon", "index")
	if err := os.MkdirAll(generateDir, 0755); err != nil {
		return fmt.Errorf("failed to create generate directory: %w", err)
	}

	outputPath := filepath.Join(generateDir, "interface.go")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// * write package declaration and imports
	_, _ = outputFile.WriteString("package index\n\n")

	// * extract types used in interface signatures and determine required imports
	requiredImports := r.InterfaceExtractRequiredImports()

	// * write merged imports
	if len(requiredImports) > 0 {
		_, _ = outputFile.WriteString("import (\n")
		for _, importPath := range requiredImports {
			_, _ = outputFile.WriteString(fmt.Sprintf("\t\"%s\"\n", importPath))
		}
		_, _ = outputFile.WriteString(")\n\n")
	}

	// * generate interface definitions (without individual imports)
	for _, interfaceInfo := range r.Interfaces {
		interfaceCode := r.InterfaceCode(interfaceInfo)
		_, _ = outputFile.WriteString(interfaceCode)
		_, _ = outputFile.WriteString("\n")
	}

	return nil
}

// InterfaceBindFile generates the bind.go file with Binder struct and methods
func (r *Generator) InterfaceBindFile() error {
	log.Printf("generating bind.go...")

	generateDir := filepath.Join("generate", "polygon", "index")
	outputPath := filepath.Join(generateDir, "bind.go")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// * write package declaration
	_, _ = outputFile.WriteString("package index\n\n")

	// * write Binder struct
	_, _ = outputFile.WriteString("type Binder struct {\n")
	for _, interfaceInfo := range r.Interfaces {
		// * convert interface name to field name (camelCase)
		fieldName := form.ToCamelCase(interfaceInfo.Name)
		_, _ = outputFile.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, interfaceInfo.Name))
	}
	_, _ = outputFile.WriteString("}\n\n")

	// * write Bind methods for each interface
	for _, interfaceInfo := range r.Interfaces {
		// * convert interface name to method name
		methodName := "Bind" + interfaceInfo.Name
		// * convert interface name to field name (camelCase)
		fieldName := form.ToCamelCase(interfaceInfo.Name)

		_, _ = outputFile.WriteString(fmt.Sprintf("func (r *Binder) %s(impl %s) {\n", methodName, interfaceInfo.Name))
		_, _ = outputFile.WriteString(fmt.Sprintf("\tr.%s = impl\n", fieldName))
		_, _ = outputFile.WriteString("}\n\n")
	}

	// * write Get methods for each interface
	for _, interfaceInfo := range r.Interfaces {
		// * convert interface name to method name
		methodName := "Get" + interfaceInfo.Name
		// * convert interface name to field name (camelCase)
		fieldName := form.ToCamelCase(interfaceInfo.Name)

		_, _ = outputFile.WriteString(fmt.Sprintf("func (r *Binder) %s() %s {\n", methodName, interfaceInfo.Name))
		_, _ = outputFile.WriteString(fmt.Sprintf("\treturn r.%s\n", fieldName))
		_, _ = outputFile.WriteString("}\n\n")
	}

	_, _ = outputFile.WriteString("func Bind() *Binder {\n")
	_, _ = outputFile.WriteString("\treturn new(Binder)\n")
	_, _ = outputFile.WriteString("}\n")

	return nil
}

// InterfaceCode creates Go interface code without individual import blocks
func (r *Generator) InterfaceCode(info InterfaceInfo) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("// %s interface defines methods for %s\n", info.Name, info.Name))
	builder.WriteString(fmt.Sprintf("type %s interface {\n", info.Name))

	for _, method := range info.Methods {
		builder.WriteString(fmt.Sprintf("\t%s\n", method.FullSignature))
	}

	builder.WriteString("}\n\n")

	return builder.String()
}

// InterfaceExtractRequiredImports analyzes interface method signatures and returns only the imports that are actually used
func (r *Generator) InterfaceExtractRequiredImports() []string {
	// * collect all imports from all interfaces
	allImports := make(map[string]string) // import path -> alias

	for _, interfaceInfo := range r.Interfaces {
		for alias, importPath := range interfaceInfo.Imports {
			allImports[importPath] = alias
		}
	}

	// * extract types used in interface signatures
	usedTypes := r.InterfaceExtractTypesFromInterfaces()

	// * filter imports based on used types
	requiredImports := r.InterfaceFilterImportsByTypes(allImports, usedTypes)

	return requiredImports
}

// InterfaceExtractTypesFromInterfaces extracts all types used in interface method signatures
func (r *Generator) InterfaceExtractTypesFromInterfaces() map[string]bool {
	usedTypes := make(map[string]bool)

	for _, interfaceInfo := range r.Interfaces {
		for _, method := range interfaceInfo.Methods {
			// * extract types from parameters and return values
			allTypes := append(method.Params, method.Returns...)

			for _, typeStr := range allTypes {
				// * split by space to get the type part (skip parameter name)
				parts := strings.Fields(typeStr)
				if len(parts) == 0 {
					continue
				}
				typ := parts[len(parts)-1]

				// * handle built-in types that don't need imports
				if code.IsBuiltinType(typ) {
					continue
				}

				usedTypes[typ] = true
			}
		}
	}

	return usedTypes
}

// InterfaceFilterImportsByTypes filters imports to only include those needed for the used types
func (r *Generator) InterfaceFilterImportsByTypes(allImports map[string]string, usedTypes map[string]bool) []string {
	// * create a mapping from type prefixes to import paths
	typeToImport := make(map[string]string)
	for importPath, alias := range allImports {
		// * skip self-import (generate/index)
		if strings.Contains(importPath, "generate/index") {
			continue
		}

		// * determine the package name from import path
		parts := strings.Split(importPath, "/")
		packageName := parts[len(parts)-1]
		if alias != packageName {
			packageName = alias
		}
		typeToImport[packageName] = importPath
	}

	// * find which imports are needed
	neededImports := make(map[string]bool)
	for typ := range usedTypes {
		// * clean up the type name (remove *, [], etc.)
		cleanType := typ
		if strings.HasPrefix(cleanType, "*") {
			cleanType = cleanType[1:]
		}
		if strings.HasPrefix(cleanType, "[]") {
			cleanType = cleanType[2:]
		}
		if strings.HasPrefix(cleanType, "[]*") {
			cleanType = cleanType[3:]
		}
		if strings.HasPrefix(cleanType, "chan ") {
			cleanType = cleanType[5:]
		}
		if strings.HasPrefix(cleanType, "chan *") {
			cleanType = cleanType[6:]
		}

		// * handle types with package prefix (e.g., "payload.Chat")
		if dotIndex := strings.Index(cleanType, "."); dotIndex != -1 {
			packageName := cleanType[:dotIndex]
			// * skip index package since it's the same package
			if packageName == "index" {
				continue
			}
			if importPath, exists := typeToImport[packageName]; exists {
				neededImports[importPath] = true
			}
		} else {
			// * skip index package since it's the same package
			if cleanType == "index" {
				continue
			}
			// * handle unqualified types
			if importPath, exists := typeToImport[cleanType]; exists {
				neededImports[importPath] = true
			}
		}
	}

	// * convert to slice
	var result []string
	for importPath := range neededImports {
		result = append(result, importPath)
	}

	return result
}
