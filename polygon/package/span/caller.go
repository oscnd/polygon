package span

import (
	"fmt"
	"runtime"
	"strings"
)

type Caller struct {
	Name *string
	Line *int
}

func (r *Caller) String() string {
	return fmt.Sprintf("%s:%d", *r.Name, *r.Line)
}

func NewCaller(skip int) *Caller {
	pc, _, line, ok := runtime.Caller(skip + 1)
	if !ok {
		panic("no caller information")
	}
	name := runtime.FuncForPC(pc).Name()
	name = name[strings.LastIndex(name, "/")+1:]

	return &Caller{
		Name: &name,
		Line: &line,
	}
}
