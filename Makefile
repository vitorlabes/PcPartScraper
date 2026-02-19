.PHONY: run test build clean lint docker-up docker-down docker-logs

# Run the scraper locally
run:
	go run cmd/scraper/main.go

# Run the consumer locally
run-consumer:
	go run cmd/consumer/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Build binaries
build:
	go build -o bin/scraper cmd/scraper/main.go
	go build -o bin/consumer cmd/consumer/main.go

# Build scraper only
build-scraper:
	go build -o bin/scraper cmd/scraper/main.go

# Build consumer only
build-consumer:
	go build -o bin/consumer cmd/consumer/main.go

# Clean generated files
clean:
	rm -rf bin/
	rm -rf exports/
	rm -f coverage.out

# Run code linting
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Verify dependencies
deps:
	go mod tidy
	go mod verify

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# ==================== DOCKER ====================

# Start the entire infrastructure
docker-up:
	docker compose up -d

# Start only PostgreSQL and RabbitMQ
docker-up-infra:
	docker compose up -d postgres rabbitmq

# Run scraper in Docker (one-time execution)
docker-run-scraper:
	docker compose run --rm scraper

# View logs
docker-logs:
	docker compose logs -f

# View consumer logs
docker-logs-consumer:
	docker compose logs -f consumer

# View scraper logs
docker-logs-scraper:
	docker compose logs -f scraper

# Stop containers
docker-down:
	docker compose down

# Stop and remove volumes (WARNING: deletes database data)
docker-down-volumes:
	docker compose down -v

# Rebuild images
docker-rebuild:
	docker compose build --no-cache

# Container status
docker-status:
	docker compose ps

# Access database
db-shell:
	docker compose exec postgres psql -U $$POSTGRES_USER -d $$POSTGRES_DB

# Access RabbitMQ Management UI
rabbitmq-ui:
	@echo "RabbitMQ Management UI: http://localhost:15672"
	@echo "User: $$RABBITMQ_USER"
	@echo "Pass: $$RABBITMQ_PASSWORD"

# Access Prometheus
prometheus-ui:
	@echo "Prometheus: http://localhost:9090"

# Access Grafana
grafana-ui:
	@echo "Grafana: http://localhost:3000"
	@echo "User: $$GRAFANA_USER"
	@echo "Pass: $$GRAFANA_PASSWORD"

# View scraper metrics
metrics-scraper:
	curl http://localhost:2114/metrics

# View consumer metrics
metrics-consumer:
	curl http://localhost:2113/metrics

run-filtered:
	GPU_TARGETS=$(GPUS) CPU_TARGETS=$(CPUS) go run cmd/scraper/main.go