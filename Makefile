.PHONY: help build run test clean docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make build       - Build the agent"
	@echo "  make run         - Run the agent"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make docker-up   - Start Docker containers"
	@echo "  make docker-down - Stop Docker containers"
	@echo "  make docker-logs - Show Docker logs"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"

build:
	@go build -o bin/agent ./cmd/agent

run: build
	@./bin/agent

test:
	@go test -v ./...

clean:
	@rm -rf bin/

docker-up:
	@docker-compose up -d

docker-down:
	@docker-compose down

docker-logs:
	@docker-compose logs -f agent

fmt:
	@go fmt ./...

lint:
	@golangci-lint run