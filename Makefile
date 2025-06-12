-include .env
export

.PHONY: run # .PHONY is to make always executes the commands regardless of whether a file named "run" exists.
run:
	trap 'exit 0' INT; go run cmd/samanager/main.go

build:
	go build -o bin/samanager cmd/samanager/main.go

migrate-up:
	migrate -database $(DB_URL) -path migrations up

migrate-down:
	migrate -database $(DB_URL) -path migrations down

db-up:
	docker compose up -d

db-down:
	docker compose down

test-db-up:
	docker compose -f docker-compose.test.yml up -d

test-db-down:
	docker compose -f docker-compose.test.yml down