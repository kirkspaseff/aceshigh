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
DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):5432/aceshigh?sslmode=disable


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
	goose -dir migrations postgres "$(DB_URL)" up
 
migrate-down: ## Roll back the most recent migration
	goose -dir migrations postgres "$(DB_URL)" down

psql: ## Open a psql shell against the running database
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

