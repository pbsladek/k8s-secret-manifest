.PHONY: all fmt vet lint test test-integration test-all build clean tidy

# Default target: format, vet, and test
all: fmt vet test

## build: compile the binary
build:
	go build -o k8s-secret-manifest .

## fmt: format all Go source files in place
fmt:
	gofmt -l -w .

## fmt-check: fail if any files need formatting (used in CI)
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files need gofmt:"; echo "$$unformatted"; exit 1; \
	fi

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint if available, otherwise fall back to go vet
lint:
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, running go vet instead"; \
		go vet ./...; \
	fi

## test: run all unit tests
test:
	go test ./... -v

## test-short: run tests without verbose output
test-short:
	go test ./...

## test-integration: compile the binary and run e2e integration tests
test-integration:
	go test -v -tags integration -count=1 ./e2e/

## test-all: run unit tests followed by integration tests
test-all: test test-integration

## cover: run tests and show coverage
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

## cover-html: open coverage report in browser
cover-html: cover
	go tool cover -html=coverage.out

## release-dry-run: run goreleaser in snapshot mode (no publish)
release-dry-run:
	goreleaser release --snapshot --clean

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## clean: remove build artefacts
clean:
	rm -f k8s-secret-manifest coverage.out
	rm -rf dist/

## help: list available targets
help:
	@grep -E '^##' Makefile | sed 's/## /  /'
