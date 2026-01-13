# QuickChat Notification Service Makefile

.PHONY: build run test clean docker docker-build docker-run lint fmt deps

# Variables
APP_NAME := notification-service
BUILD_DIR := ./build
DOCKER_IMAGE := quickchat/notification-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	$(GOCMD) run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Lint code
lint:
	@echo "Linting code..."
	$(GOLINT) run ./...

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -f docker/Dockerfile .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose -f docker/docker-compose.yml up -d

# Stop Docker Compose
docker-stop:
	@echo "Stopping services..."
	docker-compose -f docker/docker-compose.yml down

# View logs
docker-logs:
	docker-compose -f docker/docker-compose.yml logs -f notification-service

# Generate API docs (if using swag)
docs:
	@echo "Generating API documentation..."
	swag init -g cmd/server/main.go -o ./docs

# Database migrations
migrate-up:
	@echo "Running migrations..."
	mysql -h localhost -u quickchat -pquickchat_notification_password quickchat_notification < docker/init-db.sql

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose"
	@echo "  docker-logs    - View Docker logs"
	@echo "  migrate-up     - Run database migrations"
	@echo "  help           - Show this help"
