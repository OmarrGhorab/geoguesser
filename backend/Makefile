.PHONY: dev run test lint tidy migrate-up migrate-down

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
