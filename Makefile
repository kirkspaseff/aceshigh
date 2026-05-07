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


up:
	docker compose up -d
	@echo "Waiting for postgres to be ready..."
	@until docker compose exec -T postgres pg_isready -U $(POSTGRES_USER) -d $(POSTGRES_DB) >/dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Postgres is ready at $(POSTGRES_HOST):$(POSTGRES_PORT)"

down: ## Stop postgres (data preserved in the named volume)
	docker compose down


migrate-up:
	goose -dir migrations postgres "$(GOOSE_DB_URL)" up
 
migrate-down:
	goose -dir migrations postgres "$(GOOSE_DB_URL)" down

.PHONY: migrate-status
migrate-status:
	goose -dir migrations postgres "$(GOOSE_DB_URL)" status

psql: ## Open a psql shell against the running database
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

.PHONY: extract-events
extract-events:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) events > data/events.csv
	@echo "Wrote data/events.csv"
 
.PHONY: extract-aircraft
extract-aircraft:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) aircraft > data/aircraft.csv
	@echo "Wrote data/aircraft.csv"

.PHONY: extract-narratives
extract-narratives:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) narratives > data/narratives.csv
	@echo "Wrote data/narratives.csv"
 
.PHONY: extract-findings
extract-findings:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) Findings > data/findings.csv
	@echo "Wrote data/findings.csv"
 
.PHONY: extract-events-sequence
extract-events-sequence:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) Events_Sequence > data/events_sequence.csv
	@echo "Wrote data/events_sequence.csv"

.PHONY: build-loader
build-loader:
	@mkdir -p bin
	go build -o bin/loader ./cmd/loader
 
.PHONY: extract-countries
extract-countries:
	@mkdir -p data
	mdb-export -q '"' $(MDB) country > data/countries.csv
	@echo "Wrote data/countries.csv"
 
.PHONY: extract-us-states
extract-us-states:
	@mkdir -p data
	mdb-export -q '"' $(MDB) states > data/us_states.csv
	@echo "Wrote data/us_states.csv"
 
.PHONY: extract-code-lookups
extract-code-lookups:
	@mkdir -p data
	mdb-export -q '"' $(MDB) ct_iaids > data/code_lookups.csv
	@echo "Wrote data/code_lookups.csv"
 
.PHONY: extract-lookups
extract-lookups: extract-countries extract-us-states extract-code-lookups
 
.PHONY: extract-all
extract-all: extract-events extract-aircraft extract-narratives extract-findings extract-events-sequence extract-lookups ## Export all currently-supported tables
.PHONY: load-events
load-events: build-loader 
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --source data/events.csv

.PHONY: load-aircraft
load-aircraft: build-loader 
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table aircraft --source data/aircraft.csv

.PHONY: load-narratives
load-narratives: build-loader
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table narratives --source data/narratives.csv
 
.PHONY: load-findings
load-findings: build-loader
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table findings --source data/findings.csv
 
.PHONY: load-events-sequence
load-events-sequence: build-loader
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table events_sequence --source data/events_sequence.csv
 
.PHONY: load-countries
load-countries: build-loader
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table countries --source data/countries.csv
 
.PHONY: load-us-states
load-us-states: build-loader 
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table us_states --source data/us_states.csv
 
.PHONY: load-code-lookups
load-code-lookups: build-loader 
	DATABASE_URL="$(GOOSE_DB_URL)" ./bin/loader --table code_lookups --source data/code_lookups.csv
 
.PHONY: load-lookups
load-lookups: load-countries load-us-states load-code-lookups 
 
.PHONY: load-all
load-all: load-lookups load-events load-aircraft load-narratives load-findings load-events-sequence
