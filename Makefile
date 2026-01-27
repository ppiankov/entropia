.PHONY: build test fmt lint clean install run-laksa run-batch help

BINARY=entropia
BUILD_DIR=bin
MAIN_PATH=./cmd/entropia

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(MAIN_PATH)

install: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -w .
	go mod tidy

lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -rf test-reports

run-laksa: build ## Scan laksa Wikipedia article (test)
	@echo "Scanning laksa-origin artifact..."
	./$(BUILD_DIR)/$(BINARY) scan https://en.wikipedia.org/wiki/Laksa --json test-laksa.json --md test-laksa.md

run-uk-law: build ## Scan UK common-law marriage article (test)
	@echo "Scanning UK common-law marriage artifact..."
	./$(BUILD_DIR)/$(BINARY) scan https://en.wikipedia.org/wiki/Common-law_marriage --json test-uk-law.json --md test-uk-law.md

run-batch: build ## Run batch mode test
	@echo "Creating test batch file..."
	@echo "https://en.wikipedia.org/wiki/Laksa" > test-batch.txt
	@echo "https://en.wikipedia.org/wiki/Common-law_marriage" >> test-batch.txt
	@echo "https://en.wikipedia.org/wiki/List_of_common_misconceptions" >> test-batch.txt
	./$(BUILD_DIR)/$(BINARY) batch test-batch.txt --concurrency 3 --output-dir ./test-reports

.DEFAULT_GOAL := help
