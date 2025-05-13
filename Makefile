.PHONY: migrate
migrate:
	go run cmd/migrate/main.go

.PHONY: run
run:
	go run cmd/server/main.go