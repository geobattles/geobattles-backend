.PHONY: check build setup dev dev-maps db-up db-status db-stop db-down

# The default target is safe without a database, API key, or .env file.
check:
	./scripts/check.sh

# Build only after the same static checks; .local/ is ignored.
build: check
	mkdir -p .local/bin
	GOTOOLCHAIN=local GOWORK=off go build -mod=readonly -o .local/bin/geobattles-backend .

setup:
	GOTOOLCHAIN=local GOWORK=off go run -mod=readonly ./cmd/setup-env

dev: build
	exec .local/bin/geobattles-backend -dev

dev-maps: build
	exec .local/bin/geobattles-backend -dev -allow-paid-maps

db-up:
	env -u DB_PASSWORD docker compose -f compose.dev.yml --env-file .env up -d --wait database

db-status:
	env -u DB_PASSWORD docker compose -f compose.dev.yml --env-file .env ps

db-stop:
	env -u DB_PASSWORD docker compose -f compose.dev.yml --env-file .env stop database

# Keep the disposable database volume; deleting data is a separate owner action.
db-down:
	env -u DB_PASSWORD docker compose -f compose.dev.yml --env-file .env down
