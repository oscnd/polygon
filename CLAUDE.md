# Backend

## Guideline

- If any variable should apply in the name, it will have `#variableName#` in the name, e.g., `#entityName#IdRequest`

## Code
- Only comment as `// * short lowercase description`, do not add additional comment when editing.
- Use `r` as receiver name for all struct, e.g., `func (r *Service) UserCreate(...)`

## Tree

- **example**: Root of example Polygon project, used for test functionality
- **external**: Cloned external dependencies
- **pol**: The command line tool for Polygon
- **polygon**: The main Polygon library
