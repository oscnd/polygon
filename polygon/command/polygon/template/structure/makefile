include .env

.PHONY: generate
generate: generate-polygon

.PHONY: generate-polygon
generate-polygon:
	polygon generate

.PHONY: clean
clean:
	git clean -fdX

.PHONY: build
build:
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o .local/backend .

.PHONY: build-windows
build-windows:
	env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o .local/backend.exe .