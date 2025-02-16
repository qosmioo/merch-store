DOCKER_COMPOSE_FILE = docker-compose.yml
SERVICE_NAME = merch-store-service

.PHONY: all build up down test

all: build up

build:
	@echo "Building Docker containers..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) build

up:
	@echo "Starting Docker containers..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

down:
	@echo "Stopping Docker containers..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

test:
	@echo "Running tests..."
	go test ./... -v

logs:
	@echo "Showing logs for $(SERVICE_NAME)..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f $(SERVICE_NAME)

clean:
	@echo "Cleaning up..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v
	docker system prune -f