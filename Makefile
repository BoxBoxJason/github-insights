# Define the Go binary and output directory
GO ?= go
OUTPUT_DIR ?= ./bin
PROJECT_NAME ?= github-insights
MAIN_FILE ?= cmd/github-insights/*
DOCKERFILE ?= Containerfile
DOCKER_ENGINE ?= podman
GO_BUILD_FLAGS ?= -buildvcs=true

# Default target
.DEFAULT_GOAL := build

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

# Build target
build: deps
	@echo "Building the binary..."
	$(GO) build $(GO_BUILD_FLAGS) -o $(OUTPUT_DIR)/$(PROJECT_NAME) $(MAIN_FILE)

# Run target
run: build
	@echo "Running the binary..."
	$(OUTPUT_DIR)/$(PROJECT_NAME) -verbose

# Lint target
lint: deps
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4; }
	@echo "Running golangci-lint..."
	golangci-lint run ./...

# Dependency check target
dependency-check:
	@echo "Running dependency-check..."
	dependency-check --nvdApiKey $(NVD_API_KEY) --scan ./ --format ALL --out dependency-check/ --enableExperimental

# Test target
test: deps
	@command -v gotestsum >/dev/null 2>&1 || { echo "Installing gotestsum..."; go install gotest.tools/gotestsum@v1.13.0; }
	@mkdir -p codequality
	gotestsum --junitfile codequality/unit-tests.xml --format-icons octicons -- -coverprofile=codequality/coverage.out -covermode=atomic ./...
	@echo "Coverage report generated: codequality/coverage.html"


# Docker target
package:
	@echo "Building Docker image..."
	$(DOCKER_ENGINE) build -t $(PROJECT_NAME):dev -f $(DOCKERFILE) .

# Bundle results/*.json into frontend/data.js for the static viewer
frontend-data:
	@echo "Bundling results into frontend/data.js..."
	@sh frontend/generate.sh

# Generate the data bundle and print how to open the viewer
frontend: frontend-data
	@echo "Open frontend/index.html in your browser (e.g. xdg-open frontend/index.html)"

# Phony targets
.PHONY: deps build lint dependency-check test package frontend-data frontend
