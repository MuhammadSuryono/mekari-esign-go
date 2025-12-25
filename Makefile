.PHONY: build build-service build-windows run test clean tidy dev install-service

# Application name
APP_NAME=mekari-esign
VERSION?=1.0.0

# GitHub config (for auto-update)
GITHUB_OWNER?=muhammadsuryono
GITHUB_REPO?=mekari-esign-go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

# Build directory
BUILD_DIR=./bin
DIST_DIR=./dist

# Main package paths
MAIN_PATH=./cmd/main.go
SERVICE_PATH=./cmd/service/main.go

# Linker flags for version injection
LDFLAGS=-ldflags "-X mekari-esign/updater.Version=$(VERSION) -s -w"

# Build the application (Linux/Mac - development)
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build the service version (with Windows service support)
build-service:
	@echo "Building $(APP_NAME) service..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(SERVICE_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for Windows
build-windows:
	@echo "Building $(APP_NAME) for Windows..."
	@mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/windows/$(APP_NAME).exe $(SERVICE_PATH)
	@echo "Build complete: $(BUILD_DIR)/windows/$(APP_NAME).exe"

# Build release package for Windows
release-windows: build-windows
	@echo "Creating Windows release package..."
	@mkdir -p $(DIST_DIR)
	@rm -f $(DIST_DIR)/$(APP_NAME)-windows-amd64.zip
	cd $(BUILD_DIR)/windows && zip -r ../../$(DIST_DIR)/$(APP_NAME)-windows-amd64.zip $(APP_NAME).exe
	cp config.example.yml $(DIST_DIR)/config.yml
	cd $(DIST_DIR) && zip -u $(APP_NAME)-windows-amd64.zip config.yml && rm config.yml
	@echo "Release package: $(DIST_DIR)/$(APP_NAME)-windows-amd64.zip"
	@sha256sum $(DIST_DIR)/$(APP_NAME)-windows-amd64.zip > $(DIST_DIR)/$(APP_NAME)-windows-amd64.zip.sha256

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(APP_NAME)

# Run service version
run-service: build-service
	@echo "Running $(APP_NAME) service..."
	$(BUILD_DIR)/$(APP_NAME)

# Run in development mode (with hot reload using air if installed)
dev:
	@echo "Running in development mode..."
	$(GORUN) $(MAIN_PATH)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf ./embedded
	rm -rf ./tools

# Download dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Download all dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Generate swagger docs (if using swag)
swagger:
	@echo "Generating swagger docs..."
	swag init -g cmd/main.go -o docs

# Check for updates (development helper)
check-update:
	@echo "Checking for updates..."
	$(GORUN) $(SERVICE_PATH) -update

# Show version
version:
	@echo "Version: $(VERSION)"

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application (development)"
	@echo "  build-service   - Build the service version"
	@echo "  build-windows   - Build for Windows"
	@echo "  release-windows - Create Windows release package"
	@echo "  run             - Build and run the application"
	@echo "  run-service     - Build and run the service version"
	@echo "  dev             - Run in development mode"
	@echo "  test            - Run tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  tidy            - Tidy dependencies"
	@echo "  deps            - Download dependencies"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  swagger         - Generate swagger docs"
	@echo "  check-update    - Check for updates"
	@echo "  version         - Show version"
	@echo "  help            - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION         - Version number (default: 1.0.0)"
	@echo "  GITHUB_OWNER    - GitHub username for auto-update"
	@echo "  GITHUB_REPO     - GitHub repository name"

