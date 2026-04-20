SHELL := /bin/bash

# Load .env if present.
ifneq (,$(wildcard .env))
    include .env
    export
endif

DATABASE_URL ?= postgres://timetrak:timetrak@localhost:5432/timetrak?sslmode=disable

.PHONY: run build test lint migrate-up migrate-down migrate-redo dev-seed backfill-rate-snapshots check-rate-snapshots db-up db-down fmt vet tidy

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

# Populate time_entries.rate_rule_id / hourly_rate_minor / currency_code for
# closed entries missing a snapshot. Idempotent. Use DRY_RUN=1 to probe without
# writing.
backfill-rate-snapshots:
ifeq ($(DRY_RUN),1)
	go run ./cmd/migrate backfill-rate-snapshots --dry-run
else
	go run ./cmd/migrate backfill-rate-snapshots
endif

# Deploy gate: fails non-zero when any closed billable entry has a NULL rate
# snapshot. Intended to run in CI / pre-deploy after backfill-rate-snapshots.
check-rate-snapshots:
	go run ./cmd/migrate check-rate-snapshots

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
