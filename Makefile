SHELL := /bin/bash

# Load .env if present.
ifneq (,$(wildcard .env))
    include .env
    export
endif

DATABASE_URL ?= postgres://timetrak:timetrak@localhost:5432/timetrak?sslmode=disable

.PHONY: run build test lint migrate-up migrate-down migrate-redo dev-seed db-up db-down fmt vet tidy

run:
	go run ./cmd/web

build:
	mkdir -p bin
	go build -o bin/web ./cmd/web
	go build -o bin/migrate ./cmd/migrate

test:
	# -p 1 serializes test packages so integration tests that TRUNCATE don't
	# race with each other against the shared Postgres database.
	go test -p 1 ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-redo:
	go run ./cmd/migrate redo

dev-seed:
	go run ./cmd/migrate seed

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
