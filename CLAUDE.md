# Backend

## Guideline

- If any variable should apply in the name, it will have `#variableName#` in the name, e.g., `#entityName#IdRequest`

## Tree

- **example**: Root of example Polygon project, used for test functionality
- **external**: Cloned external dependencies
- **pol**: The command line tool for Polygon
- **polygon**: The main Polygon library

## Current Development

- Currently, the main focus is on developing `polygon` command line tool, test by (cd example && go run ../polygon/command/polygon -d polygon <subcommand>)
- Install command globally using `go install ./polygon/command/polygon` and use `polygon <subcommand>` anywhere