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

func NewCaller() *Caller {
	// * find outer package caller
	skip := 1
	for {
		pc, _, _, ok := runtime.Caller(skip)
		if !ok {
			panic("no caller information")
		}
		name := runtime.FuncForPC(pc).Name()
		if !strings.HasPrefix(name, "go.scnd.dev/open/polygon/span.") {
			break
		}
		skip++
	}

	pc, _, line, ok := runtime.Caller(skip)
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
