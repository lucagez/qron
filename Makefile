.PHONY: test gqlgen sqlc migrate_up recreate_local_env 

test: 
	@echo "Running TinyQ tests..."
	@go test ./...

gqlgen:
	@echo "Generating gqlgen graph..."
	@rm -rf ./graph/generated
	@go generate ./...

sqlc:
	@echo "Generating sqlc queries..."
	@docker run --rm -v $(shell pwd):/src -w /src kjconroy/sqlc generate

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