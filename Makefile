include .env
export

# ---- helpers ----

.PHONY: confirm
confirm:
	@echo -n "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]

.PHONY: help
help:
	@echo "Usage: "
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# ---- development ----

## db/migrations/up: apply all the database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_DNS)" up

## db/migrations/up: revert all the database migrations
.PHONY: db/migrations/down
db/migrations/down: confirm
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_DNS)" down

## db/migrations/status: display the current status of all database migrations
.PHONY: db/migrations/status
db/migrations/status:
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_DNS)" status

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo "Creating migration files for ${name}"
	goose create ${name} --dir ./migrations sql

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run $(MAIN_PACKAGE)

# ---- quality control ----

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor
