package erroring

import (
	"fmt"
	"runtime"
	"strings"
)

type Trace struct {
	Name *string
	Line *int
}

func (r *Trace) String() string {
	return fmt.Sprintf("%s:%d", *r.Name, *r.Line)
}

func NewTrace(skip int) *Trace {
	pc, _, line, ok := runtime.Caller(skip + 1)
	if !ok {
		panic("no caller information")
	}
	name := runtime.FuncForPC(pc).Name()
	name = name[strings.LastIndex(name, "/")+1:]

	return &Trace{
		Name: &name,
		Line: &line,
	}
}
