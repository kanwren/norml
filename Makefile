.DEFAULT_GOAL := build

.PHONY: build
build:
	go build ./cmd/norml

.PHONY: install
install:
	go install ./cmd/norml

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	pre-commit run --all-files
	golangci-lint run ./...

.PHONY: test
test:
	go test ./...

.PHONY: check
check: build test lint

.PHONY: dist
dist:
	goreleaser release --snapshot --clean

.PHONY: release
release:
	goreleaser release --clean

.PHONY: clean
clean:
	rm -rf ./norml ./dist/
