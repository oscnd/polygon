# Backend

## Guideline

- If text contains dynamic pattern, `#variableName#` (starts end with `#`)
  will be used as example `#entityName#IdRequest`
- The project is based on Polygon architecture.

## General

- Use pointer as basis, use `&` for normal variable and use `go.scnd.dev/open/polygon/utility/value` with
  `gut.Ptr("")` or `gut.Ptr(uint64(50))` for specific type to get pointer of a value declared inline.
- Use `r` as receiver name for all struct, e.g., `func (r *Service) UserCreate(...) ...`
- Comment format: `// * lowercase compact action`
- Use camel case for json tags
- The project structure use one declaration per file as basis, e.g., one endpoint / procedure per file, except for
  main struct and constructor which has `type Handler` / `func Handle`, `type Service struct` /  `func New` in one file.
- Dependencies will be injected from `./generate/polygon/index/interface.go` which have all interface that can be used
  across modules, in some edit, it requires `make generate` to regenerate the file.
- For any unfamiliar package, use `go doc` to check usage and examples.
