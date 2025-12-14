.PHONY: generate
generate: init vert mock

.PHONY: init
init:
	mkdir -p generate/convert
	mkdir -p .local/dist
	touch .local/dist/index.html


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