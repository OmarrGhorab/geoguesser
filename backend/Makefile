.PHONY: dev run test lint tidy migrate-up migrate-down migrate-create openapi-validate

DATABASE_URL ?= postgres://geoguess:geoguess@localhost:5432/geoguess?sslmode=disable

dev:
	air

run:
	go run ./cmd/api

test:
	go test ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

migrate-up:
	goose -dir ./migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir ./migrations postgres "$(DATABASE_URL)" down

migrate-create:
	@:$(if $(name),,$(error name is required. Usage: make migrate-create name=create_users))
	goose -dir ./migrations create $(name) sql

openapi-validate:
	npx -y @redocly/cli lint ./openapi/openapi.yaml
