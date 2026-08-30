SHELL := /bin/bash

include .env
export

MIGRATIONS_PATH := ./migrations
MIGRATE := migrate

.PHONY: migrate-up migrate-down migrate-version migrate-create
.PHONY: docker-up docker-down docker-logs docker-build

migrate-up:
	@test -n "$(GOREST_DATABASE_URL)" || (echo "GOREST_DATABASE_URL is not set"; exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(GOREST_DATABASE_URL)" up

migrate-down:
	@test -n "$(GOREST_DATABASE_URL)" || (echo "GOREST_DATABASE_URL is not set"; exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(GOREST_DATABASE_URL)" down 1

migrate-version:
	@test -n "$(GOREST_DATABASE_URL)" || (echo "GOREST_DATABASE_URL is not set"; exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(GOREST_DATABASE_URL)" version

migrate-create:
	@test - "$(GOREST_DATABASE_URL)" || (echo "GOREST_DATABASE_URL is not set"; exit 1)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_PATH) $(name)

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-build:
	docker compose build
