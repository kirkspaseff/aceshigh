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
include .env
DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):5432/aceshigh?sslmode=disable

up:
	docker compose up -d
	@echo "Waiting for postgres to be ready..."
	@until docker compose exec -T postgres pg_isready -U $(POSTGRES_USER) -d $(POSTGRES_DB) >/dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Postgres is ready at $(POSTGRES_HOST):$(POSTGRES_PORT)"

down:
	docker compose down

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up
 
migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir migrations postgres "$(DATABASE_URL)" status

psql:
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

extract-events:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) events > data/events.csv
	@echo "Wrote data/events.csv"
 
extract-aircraft:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) aircraft > data/aircraft.csv
	@echo "Wrote data/aircraft.csv"

extract-narratives:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) narratives > data/narratives.csv
	@echo "Wrote data/narratives.csv"
 
extract-findings:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) Findings > data/findings.csv
	@echo "Wrote data/findings.csv"
 
extract-events-sequence:
	@mkdir -p data
	mdb-export -q '"' -T '%Y-%m-%d %H:%M:%S' $(MDB) Events_Sequence > data/events_sequence.csv
	@echo "Wrote data/events_sequence.csv"
 
extract-countries:
	@mkdir -p data
	mdb-export -q '"' $(MDB) country > data/countries.csv
	@echo "Wrote data/countries.csv"
 
extract-us-states:
	@mkdir -p data
	mdb-export -q '"' $(MDB) states > data/us_states.csv
	@echo "Wrote data/us_states.csv"
 
extract-code-lookups:
	@mkdir -p data
	mdb-export -q '"' $(MDB) ct_iaids > data/code_lookups.csv
	@echo "Wrote data/code_lookups.csv"
 
extract-lookups: extract-countries extract-us-states extract-code-lookups
 
extract-all: extract-events extract-aircraft extract-narratives extract-findings extract-events-sequence extract-lookups

build-loader:
	@mkdir -p bin
	go build -o bin/loader ./cmd/loader

load-events: build-loader 
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table events --source data/events.csv

load-aircraft: build-loader 
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table aircraft --source data/aircraft.csv

load-narratives: build-loader
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table narratives --source data/narratives.csv
 
load-findings: build-loader
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table findings --source data/findings.csv
 
load-events-sequence: build-loader
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table events_sequence --source data/events_sequence.csv
 
load-countries: build-loader
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table countries --source data/countries.csv
 
load-us-states: build-loader 
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table us_states --source data/us_states.csv
 
load-code-lookups: build-loader 
	DATABASE_URL="$(DATABASE_URL)" ./bin/loader --table code_lookups --source data/code_lookups.csv
 
load-lookups: load-countries load-us-states load-code-lookups 
 
.PHONY: load-all
load-all: load-lookups load-events load-aircraft load-narratives load-findings load-events-sequence
