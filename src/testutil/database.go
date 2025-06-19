package testutil

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethaccount/backend/src/utils"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var migrationPath = "file://" + filepath.Join(utils.FindProjectRoot(), "migrations")

func SetupTestDB(t *testing.T) *gorm.DB {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("Error loading .env file")
	}

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		t.Fatalf("TEST_DB_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	migration, err := migrate.New(
		migrationPath,
		dsn)
	if err != nil {
		log.Fatalf("failed to create migrate: %v", err)
	}

	migration.Up()
	return db
}

func CleanupTestDB(t *testing.T, db *gorm.DB) {
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		t.Fatalf("TEST_DB_URL is not set")
	}

	migration, err := migrate.New(
		migrationPath,
		dsn)
	if err != nil {
		log.Fatalf("failed to create migrate: %v", err)
	}

	migration.Down()
}
