# Binary name
BINARY_NAME=crobe

# Build flags for production (smaller binary)
LDFLAGS=-s -w
BUILD_FLAGS=-trimpath -ldflags="$(LDFLAGS)"

.PHONY: help schema build build-linux build-windows build-mac-intel build-mac-arm build-builder build-builder-linux build-builder-windows build-builder-mac-intel build-builder-mac-arm test test-coverage test-coverage-report test-e2e clean

## help: Show this help message
help:
	@echo "crobe Task Runner"
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

schema: ## Generate playbook.schema.json
	go run ./cmd/builder --schema > playbook.schema.json

## build: Build production binaries for all platforms
build: build-linux build-windows build-mac-intel build-mac-arm ## Build production binaries for all platforms

build-linux: ## Build production for Linux (amd64)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux ./cmd/probe

build-windows: ## Build production for Windows (amd64)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows.exe ./cmd/probe

build-mac-intel: ## Build production for Mac Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-mac-intel ./cmd/probe

build-mac-arm: ## Build production for Mac Arm (arm64)
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-mac-arm ./cmd/probe

## build-builder: Build builder binaries (with esbuild tags) for all platforms
build-builder: build-builder-linux build-builder-windows build-builder-mac-intel build-builder-mac-arm ## Build builder binaries for all platforms

build-builder-linux: ## Build builder for Linux (amd64)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-builder-linux ./cmd/builder

build-builder-windows: ## Build builder for Windows (amd64)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-builder-windows.exe ./cmd/builder

build-builder-mac-intel: ## Build builder for Mac Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-builder-mac-intel ./cmd/builder

build-builder-mac-arm: ## Build builder for Mac Arm (arm64)
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-builder-mac-arm ./cmd/builder

test: ## Run go tests
	go test -v ./...

test-coverage: ## Run go tests with coverage
	go test -cover ./...

test-coverage-report: ## Generate coverage.html and coverage.json for CI
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@PERCENTAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	COLOR="red"; \
	IS_BRIGHTGREEN=$$(echo "$$PERCENTAGE" | awk '{print ($$1 > 80)}'); \
	IS_YELLOW=$$(echo "$$PERCENTAGE" | awk '{print ($$1 > 50)}'); \
	if [ "$$IS_BRIGHTGREEN" = "1" ]; then COLOR="brightgreen"; \
	elif [ "$$IS_YELLOW" = "1" ]; then COLOR="yellow"; fi; \
	echo "{\"schemaVersion\": 1, \"label\": \"coverage\", \"message\": \"$$PERCENTAGE%\", \"color\": \"$$COLOR\"}" > coverage.json

test-e2e: ## Verify transpilation and running playbooks with compiled binaries
	@echo "Verifying builder with built binaries..."
	# 1. Preprocess raw playbook to baked version
	./$(BINARY_NAME)-builder-linux --preprocess --input test-e2e/test.playbook.raw.yaml --output test-e2e/test.playbook.baked.yaml
	# 2. Run the baked playbook using the probe agent
	./$(BINARY_NAME)-linux test-e2e/test.playbook.baked.yaml
	# 3. Test direct execution of raw playbook via builder
	./$(BINARY_NAME)-builder-linux test-e2e/test.playbook.raw.yaml
	# 4. Deep Audit of generated reports and file persistence
	./$(BINARY_NAME)-linux test-e2e/test.playbook.reportevaluate.yaml
	@echo "✅ E2E verification passed"


## clean: Remove all generated binaries
clean: ## Remove generated binaries
	rm -f $(BINARY_NAME)-*
	rm -rf reports
