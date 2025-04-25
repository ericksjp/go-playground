include .env
export

migrate-up:
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_DNS)" up
migrate-down:
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_DNS)" down
migrate-status:
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_DNS)" status

run:
	go run $(MAIN_PACKAGE)

migrate-and-run: migrate-up run
