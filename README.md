# SAManager Backend

### test

```
go test ./src/repository -v
```

### database

```
migrate create -ext sql -dir migrations -seq create_jobs_table
migrate -database $DB_URL -path migrations up
migrate -database $DB_URL -path migrations down -all
```

docker

```
docker exec -it samanager_test_db psql -U postgres -d samanager_test
```

psql

```psql
<!-- list of relations -->
\dt

```

### swagger

Generate swagger docs

```
swag init -g cmd/server/main.go -o docs/swagger
```