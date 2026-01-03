package code

type Module struct {
	Path         *string             // absolute path
	Name         *string             // module name
	Packages     map[string]*Package // key: relative path
	PackageNames map[string]string   // key: package name -> value: relative path
}
