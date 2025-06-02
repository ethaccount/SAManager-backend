.PHONY: migrate
migrate:
	go run cmd/migrate/main.go

.PHONY: run
run:
	go run cmd/server/main.go


# Start test database
test-db-up:
	docker compose -f docker-compose.test.yml up -d postgres_test

# Stop test database
test-db-down:
	docker compose -f docker-compose.test.yml down