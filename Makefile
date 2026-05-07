# Aces High — development workflow
#
# Common targets:
#   make up              start postgres
#   make down            stop postgres (data preserved)
#   make reset           nuke postgres + volume, start fresh, apply migrations
#   make migrate-up      apply all pending migrations
#   make migrate-down    roll back the most recent migration
#   make migrate-status  show migration state
#   make migrate-create name=add_foo  scaffold a new migration file
#   make psql            open a psql shell against the running db
#   make logs            tail postgres logs
#   make install-tools   install goose CLI
#
# The DB connection is read from .env (copy from .env.example on first setup).
include .env
# Goose uses its own DSN format for postgres. We build it once here.
GOOSE_DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):5432/aceshigh?sslmode=disable


up: ## Start postgres in the background and wait until healthy
	docker compose up -d
	@echo "Waiting for postgres to be ready..."
	@until docker compose exec -T postgres pg_isready -U $(POSTGRES_USER) -d $(POSTGRES_DB) >/dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Postgres is ready at $(POSTGRES_HOST):$(POSTGRES_PORT)"

down: ## Stop postgres (data preserved in the named volume)
	docker compose down


migrate-up: ## Apply all pending migrations
	goose -dir migrations postgres "$(GOOSE_DB_URL)" up
 
migrate-down: ## Roll back the most recent migration
	goose -dir migrations postgres "$(GOOSE_DB_URL)" down

.PHONY: migrate-status
migrate-status: ## Show which migrations have been applied
	goose -dir migrations postgres "$(GOOSE_DB_URL)" status

psql: ## Open a psql shell against the running database
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

.PHONY: extract-events
extract-events: ## Export events from MDB to data/events.csv (override MDB=path/to/avall.mdb)
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) events > data/events.csv
	@echo "Wrote data/events.csv"
 
.PHONY: extract-aircraft
extract-aircraft: ## Export aircraft from MDB to data/aircraft.csv
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) aircraft > data/aircraft.csv
	@echo "Wrote data/aircraft.csv"

.PHONY: extract-narratives
extract-narratives: ## Export narratives from MDB to data/narratives.csv
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) narratives > data/narratives.csv
	@echo "Wrote data/narratives.csv"
 
.PHONY: extract-findings
extract-findings: ## Export findings from MDB to data/findings.csv
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) Findings > data/findings.csv
	@echo "Wrote data/findings.csv"
 
.PHONY: extract-events-sequence
extract-events-sequence: ## Export Events_Sequence from MDB to data/events_sequence.csv
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) Events_Sequence > data/events_sequence.csv
	@echo "Wrote data/events_sequence.csv"
.PHONY: build-loader
build-loader: ## Compile the loader binary
	@mkdir -p bin
	go build -o bin/loader ./cmd/loader
 
.PHONY: load-events
load-events: build-loader 
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --source data/events.csv

.PHONY: load-aircraft
load-aircraft: build-loader 
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table aircraft --source data/aircraft.csv

.PHONY: load-narratives
load-narratives: build-loader ## Load narratives.csv (aircraft must be loaded first)
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table narratives --source data/narratives.csv
 
.PHONY: load-findings
load-findings: build-loader ## Load findings.csv (aircraft must be loaded first)
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table findings --source data/findings.csv
 
.PHONY: load-events-sequence
load-events-sequence: build-loader ## Load events_sequence.csv (aircraft must be loaded first)
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table events_sequence --source data/events_sequence.csv
 
.PHONY: load-all
load-all: load-events load-aircraft load-narratives load-findings load-events-sequence ## Load all tables in dependency order

