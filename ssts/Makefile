# System Stress Testing Suite (SSTS) - Makefile

# Variables
BINARY_NAME=ssts
DOCKER_IMAGE=ssts/ssts
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Default target
.PHONY: all
all: test build

# Build the binary
.PHONY: build
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ssts

# Build for multiple platforms
.PHONY: build-all
build-all:
	# Build for Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/ssts
	# Build for macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/ssts
	# Build for Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/ssts

# Test
.PHONY: test
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Test with coverage report
.PHONY: test-coverage
test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Benchmark tests
.PHONY: bench
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Clean
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

# Run
.PHONY: run
run:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ssts
	./$(BINARY_NAME) server

# Run with development configuration
.PHONY: run-dev
run-dev:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ssts
	./$(BINARY_NAME) server --config ssts.yaml --debug

# Dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
.PHONY: deps-update
deps-update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Lint
.PHONY: lint
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
.PHONY: fmt
fmt:
	gofmt -s -w .
	goimports -w .

# Web frontend
.PHONY: web-install
web-install:
	cd web && npm install

.PHONY: web-build
web-build:
	cd web && npm run build

.PHONY: web-dev
web-dev:
	cd web && npm start

.PHONY: web-test
web-test:
	cd web && npm test

# Docker
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE):latest

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Docker Compose
.PHONY: up
up:
	docker-compose up -d

.PHONY: down
down:
	docker-compose down

.PHONY: logs
logs:
	docker-compose logs -f

# Database migrations
.PHONY: migrate
migrate:
	./$(BINARY_NAME) migrate --config ssts.yaml

# Generate API documentation
.PHONY: docs
docs:
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/ssts/main.go -o docs/swagger

# Example tests
.PHONY: test-cpu
test-cpu:
	./$(BINARY_NAME) run examples/cpu-stress-test.yaml

.PHONY: test-memory
test-memory:
	./$(BINARY_NAME) run examples/memory-stress-test.yaml

.PHONY: test-io
test-io:
	./$(BINARY_NAME) run examples/io-stress-test.yaml

.PHONY: test-comprehensive
test-comprehensive:
	./$(BINARY_NAME) run examples/comprehensive-test.yaml

# Development helpers
.PHONY: dev-setup
dev-setup: deps web-install
	@echo "Development environment setup complete"

.PHONY: dev-run
dev-run:
	@echo "Starting development environment..."
	make -j2 run-dev web-dev

# CI/CD helpers
.PHONY: ci-test
ci-test: deps lint test

.PHONY: ci-build
ci-build: build-all docker-build

# Release
.PHONY: release
release: clean ci-test ci-build
	@echo "Release $(VERSION) built successfully"

# Install locally
.PHONY: install
install: build
	cp $(BINARY_NAME) /usr/local/bin/

# Uninstall
.PHONY: uninstall
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

# Generate configuration template
.PHONY: config-template
config-template:
	./$(BINARY_NAME) config --template > ssts-template.yaml

# Validate configuration
.PHONY: config-validate
config-validate:
	./$(BINARY_NAME) validate --config ssts.yaml

# Database setup for development
.PHONY: db-setup
db-setup:
	docker run -d --name ssts-postgres -e POSTGRES_PASSWORD=password -p 5432:5432 postgres:14
	docker run -d --name ssts-influxdb -p 8086:8086 influxdb:2.0

.PHONY: db-teardown
db-teardown:
	docker stop ssts-postgres ssts-influxdb || true
	docker rm ssts-postgres ssts-influxdb || true

# Security scan
.PHONY: security
security:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

# Performance profiling
.PHONY: profile-cpu
profile-cpu:
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	go tool pprof cpu.prof

.PHONY: profile-mem
profile-mem:
	$(GOTEST) -memprofile=mem.prof -bench=. ./...
	go tool pprof mem.prof

# DevOps and Infrastructure targets
.PHONY: deploy-dev deploy-staging deploy-prod
deploy-dev:
	./scripts/deploy.sh -e dev up

deploy-staging:
	./scripts/deploy.sh -e staging -m k8s up

deploy-prod:
	./scripts/deploy.sh -e prod -m k8s up

.PHONY: health-check backup restore
health-check:
	./scripts/health-check.sh

backup:
	./scripts/deploy.sh backup

restore:
	./scripts/deploy.sh restore

# Kubernetes targets
.PHONY: k8s-deploy-dev k8s-deploy-prod k8s-status k8s-logs
k8s-deploy-dev:
	kubectl apply -k k8s/overlays/dev

k8s-deploy-prod:
	kubectl apply -k k8s/overlays/prod

k8s-status:
	kubectl get all -n ssts-prod

k8s-logs:
	kubectl logs -f deployment/ssts-app -n ssts-prod

# Monitoring setup
.PHONY: monitoring-setup monitoring-dashboards
monitoring-setup:
	docker-compose up -d prometheus grafana
	@echo "Monitoring available at:"
	@echo "  Grafana: http://localhost:3000 (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"

monitoring-dashboards:
	@echo "Setting up Grafana dashboards..."
	# Import dashboard configurations here

# Performance and load testing
.PHONY: perf-test load-test stress-test
perf-test:
	./$(BINARY_NAME) run examples/comprehensive-test.yaml

load-test:
	@echo "Running load tests..."
	# Add load testing framework integration

stress-test:
	@echo "Running stress tests..."
	./$(BINARY_NAME) run examples/cpu-stress-test.yaml
	./$(BINARY_NAME) run examples/memory-stress-test.yaml
	./$(BINARY_NAME) run examples/io-stress-test.yaml

# Cleanup and maintenance
.PHONY: clean-docker clean-k8s clean-all
clean-docker:
	./scripts/deploy.sh clean

clean-k8s:
	kubectl delete namespace ssts-dev ssts-staging ssts-prod --ignore-not-found=true

clean-all: clean clean-docker
	rm -rf data/ logs/ backups/

# Development environment setup
.PHONY: dev-full-setup quick-start
dev-full-setup: deps web-install
	mkdir -p data logs grafana/dashboards grafana/datasources
	chmod +x scripts/*.sh
	@echo "Full development environment ready!"

quick-start: dev-full-setup docker-build up health-check
	@echo ""
	@echo "🎉 SSTS is running!"
	@echo ""
	@echo "Available services:"
	@echo "  SSTS UI:        http://localhost:8080"
	@echo "  Grafana:        http://localhost:3000 (admin/admin)"
	@echo "  Prometheus:     http://localhost:9090"
	@echo ""

# Comprehensive testing
.PHONY: test-all test-integration test-e2e
test-all: test test-integration test-e2e security

test-integration:
	docker-compose -f docker-compose.test.yml up -d postgres redis influxdb
	sleep 10
	$(GOTEST) -v -tags=integration ./...
	docker-compose -f docker-compose.test.yml down -v

test-e2e:
	@echo "Running end-to-end tests..."
	# Add E2E testing framework

# Help
.PHONY: help
help:
	@echo "SSTS - System Stress Testing Suite"
	@echo ""
	@echo "Build Targets:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms"
	@echo "  docker-build   - Build Docker image"
	@echo ""
	@echo "Testing Targets:"
	@echo "  test           - Run unit tests"
	@echo "  test-all       - Run all tests (unit, integration, e2e)"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  bench          - Run benchmark tests"
	@echo "  perf-test      - Run performance tests"
	@echo "  load-test      - Run load tests"
	@echo "  stress-test    - Run stress tests"
	@echo ""
	@echo "Development Targets:"
	@echo "  run            - Build and run server"
	@echo "  run-dev        - Run in development mode"
	@echo "  dev-setup      - Set up development environment"
	@echo "  quick-start    - Complete setup and start"
	@echo "  web-dev        - Start web development server"
	@echo ""
	@echo "Docker Targets:"
	@echo "  up             - Start with docker-compose"
	@echo "  down           - Stop docker-compose"
	@echo "  logs           - View docker-compose logs"
	@echo "  docker-run     - Run Docker container"
	@echo ""
	@echo "Kubernetes Targets:"
	@echo "  k8s-deploy-dev  - Deploy to Kubernetes dev"
	@echo "  k8s-deploy-prod - Deploy to Kubernetes prod"
	@echo "  k8s-status      - Show Kubernetes status"
	@echo "  k8s-logs        - Show Kubernetes logs"
	@echo ""
	@echo "Deployment Targets:"
	@echo "  deploy-dev      - Deploy to development"
	@echo "  deploy-staging  - Deploy to staging"
	@echo "  deploy-prod     - Deploy to production"
	@echo ""
	@echo "Operations Targets:"
	@echo "  health-check    - Run health checks"
	@echo "  backup          - Backup databases"
	@echo "  restore         - Restore databases"
	@echo "  monitoring-setup - Set up monitoring stack"
	@echo ""
	@echo "Maintenance Targets:"
	@echo "  clean          - Clean build artifacts"
	@echo "  clean-docker   - Clean Docker resources"
	@echo "  clean-all      - Clean everything"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  security       - Run security scan"
	@echo ""
	@echo "Utility Targets:"
	@echo "  install        - Install binary locally"
	@echo "  config-validate - Validate configuration"
	@echo "  help           - Show this help message"