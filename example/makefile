.PHONY: generate
generate: init sqlc vert mock

.PHONY: init
init:
	mkdir -p generate/convert
	mkdir -p .local/dist
	touch .local/dist/index.html

.PHONY: sqlc
sqlc:
	sqlc generate
	find ./generate/*sql -type f \( -name "models.go" -o -name "*.sql.go" -o -name "querier.go" \) -exec sed -i '' \
		-e 's/\"database\/sql\"//g' \
		-e 's/string/\*string/g' \
		-e 's/\([^:]\)bool/\1*bool/g' \
		-e 's/int64/\*uint64/g' \
		-e 's/int32/\*int32/g' \
		-e 's/float64/\*float64/g' \
		-e 's/time\.Time/\*time.Time/g' \
		-e 's/sql\.NullString/\*string/g' \
		-e 's/sql\.NullTime/\*time.Time/g' \
		-e 's/sql\.NullInt64/\*uint64/g' \
		-e 's/sql\.NullInt32/\*int32/g' \
		-e 's/sql\.NullFloat64/\*float64/g' \
		-e 's/sql\.NullBool/\*bool/g' \
		{} +
	find ./generate/psql -type f -exec sed -i '' -e 's/DBTX/PDBTX/g' -e 's/Querier/PQuerier/g' {} +

.PHONY: vert
vert:
	find ./generate/convert -type f -exec sed -i '' '/^\/\/go:build !goverter/d' {} +
	goverter gen -g 'skipCopySameType yes' ./helper/convert || true

.PHONY: mock
mock:
	mockery

.PHONY: build
build:
	make generate
	(cd ../frontend && bun run build && bun run build:embed)
	cp -r ../frontend/dist/ .local/dist/
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o .local/backend .