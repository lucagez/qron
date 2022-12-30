.PHONY: test gqlgen sqlc migrate_up recreate_local_env connect_local

test: 
	@echo "Running TinyQ tests..."
	@TZ=UTC go test ./... -race -count=1 -timeout=30s

gqlgen:
	@echo "Generating gqlgen graph..."
	@rm -rf ./graph/generated
	@go generate ./...

sqlc:
	@echo "Generating sqlc queries..."
	# @docker run --rm -v $(shell pwd):/src -w /src kjconroy/sqlc generate
	@sqlc-dev --experimental generate

migrate_up:
	@echo "Migrate local db..."
	@goose -dir ./migrations postgres "postgresql://postgres:password@localhost:5435/postgres?sslmode=disable" up

recreate_local_env:
	@echo "Recreating local environment..."
	@docker compose down -v
	@docker compose up -d
	@sleep 1
	@make migrate_up
	@echo "Done 🎉"

connect_local:
	@echo "COnnecting to local db..."
	@psql -d postgres://postgres:password@localhost:5435/postgres
