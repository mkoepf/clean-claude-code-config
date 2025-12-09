.PHONY: build test test-unit test-safety test-e2e test-all clean help

# Default target
all: build

# Build the binary
build:
	go build -o cccc ./cmd/ccc

# Run unit tests
test: test-unit

test-unit:
	go test ./internal/... ./cmd/...

# Run safety tests (verifies tool never deletes existing projects)
test-safety:
	go test -v -tags=safety ./test/safety/...

# Run E2E tests in Docker
test-e2e:
	docker build -t cccc-test -f test/Dockerfile .
	docker run --rm cccc-test

# Run all tests
test-all: test-unit test-safety
	@echo "All local tests passed"

# Run code quality checks
quality:
	./scripts/code_quality.sh

# Clean build artifacts
clean:
	rm -f cccc
	go clean

# Install the binary
install: build
	cp cccc $(GOPATH)/bin/ 2>/dev/null || cp cccc ~/go/bin/

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the cccc binary"
	@echo "  test        - Run unit tests (alias for test-unit)"
	@echo "  test-unit   - Run unit tests"
	@echo "  test-safety - Run safety tests"
	@echo "  test-e2e    - Run E2E tests in Docker"
	@echo "  test-all    - Run all local tests (unit + safety)"
	@echo "  quality     - Run code quality checks"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  help        - Show this help message"
