.PHONY: help dev test coverage lint check build docker run infra-up infra-down sonar clean

help:
	@echo "Job Service Go — available targets:"
	@echo "  make dev          Run with hot reload (requires 'air' or 'realize')"
	@echo "  make test         Run unit tests"
	@echo "  make coverage     Run tests with coverage"
	@echo "  make lint         Run go vet + gofmt"
	@echo "  make check        Run lint + test + coverage gate"
	@echo "  make build        Build the binary"
	@echo "  make docker       Build Docker image"
	@echo "  make run          Run the application"
	@echo "  make infra-up     Start PG + Redis + RabbitMQ via docker compose"
	@echo "  make infra-down   Stop the dev infrastructure"
	@echo "  make sonar        Run SonarQube scan"
	@echo "  make clean        Remove build artifacts"

dev:
	GOTOOLCHAIN=go1.25.0 go run ./cmd/jobservice

test:
	GOTOOLCHAIN=go1.25.0 go test ./... -count=1

coverage:
	GOTOOLCHAIN=go1.25.0 go test ./... -coverprofile=coverage.out -covermode=atomic
	@echo ""
	@echo "Coverage report:"
	@GOTOOLCHAIN=go1.25.0 go tool cover -func=coverage.out | tail -1

lint:
	go vet ./...
	@! gofmt -l . | grep -v vendor/ | grep . && echo "✅ gofmt clean"

check: lint test
	@echo "✅ All checks passed"

build:
	GOTOOLCHAIN=go1.25.0 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/jobservice ./cmd/jobservice

docker:
	docker build -t job-service-go:latest .

run:
	GOTOOLCHAIN=go1.25.0 go run ./cmd/jobservice

infra-up:
	docker compose -f docker-compose.infra.yml up -d

infra-down:
	docker compose -f docker-compose.infra.yml down

sonar:
	./scripts/sonar-scan.sh "job-service-go" "job-service-go"

clean:
	rm -rf bin/ coverage.out
