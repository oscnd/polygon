package code

type Package struct {
	Path          *string          `json:"path"`        // relative path from module root
	DirectoryName *string          `json:"dirName"`     // last part of path
	Package       *string          `json:"package"`     // full package name
	PackageName   *string          `json:"packageName"` // package name from ast
	Files         map[string]*File `json:"files"`       // key: file name with .go extension
	Module        *Module          `json:"module"`      // back reference to parent module
}

func (r *Package) Interfaces() []*Interface {
	interfaces := make([]*Interface, 0)
	for _, file := range r.Files {
		interfaces = append(interfaces, file.Interfaces...)
	}
	return interfaces
}

func (r *Package) Structs() []*Struct {
	structs := make([]*Struct, 0)
	for _, file := range r.Files {
		structs = append(structs, file.Structs...)
	}
	return structs
}

func (r *Package) Receivers() []*Receiver {
	receivers := make([]*Receiver, 0)
	for _, file := range r.Files {
		receivers = append(receivers, file.Receivers...)
	}
	return receivers
}

func (r *Package) Functions() []*Function {
	functions := make([]*Function, 0)
	for _, file := range r.Files {
		functions = append(functions, file.Functions...)
	}
	return functions
}

func (r *Package) EntityByName(name string) (*Interface, *Struct, *Receiver, *Function) {
	for _, iface := range r.Interfaces() {
		if iface.Name != nil && *iface.Name == name {
			return iface, nil, nil, nil
		}
	}
	for _, strct := range r.Structs() {
		if strct.Name != nil && *strct.Name == name {
			return nil, strct, nil, nil
		}
	}
	for _, recv := range r.Receivers() {
		if recv.Name != nil && *recv.Name == name {
			return nil, nil, recv, nil
		}
	}
	for _, fn := range r.Functions() {
		if fn.Name != nil && *fn.Name == name {
			return nil, nil, nil, fn
		}
	}
	return nil, nil, nil, nil
}

func (r *Package) FileByName(name string) (*File, bool) {
	file, exists := r.Files[name]
	return file, exists
}
