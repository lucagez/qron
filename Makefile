.PHONY: test gqlgen sqlc migrate_up recreate_local_env connect_local generate_migration httpdev build_httpdev release tag

test: 
	@echo "Running qron tests..."
	@TZ=UTC go test ./... -race -count=1 -timeout=30s

httpdev: 
	@echo "Starting httpdev..."
	@TZ=UTC go run cmd/httpdev/main.go 

build_httpdev: 
	@echo "Building httpdev..."
	@go build -o build/qron ./cmd/httpdev/...

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

generate_migration:
	@echo "Generating migration..."
	@goose -dir ./migrations postgres create $(name) sql

recreate_local_env:
	@echo "Recreating local environment..."
	@docker compose down -v
	@docker compose up -d
	@sleep 1
	@make migrate_up
	@echo "Done 🎉"

connect_local:
	@echo "Connecting to local db..."
	@psql -d postgres://postgres:password@localhost:5435/postgres

release:
	@echo "Releasing..."
	@goreleaser build --snapshot --rm-dist

tag:
	@echo "Tagging..."
	@git tag -a v$(version) -m "Release v$(version)"
	@git push origin v$(version)
