package testutil

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func findProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Walk up the directory tree to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod
			panic("Could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

var migrationPath = "file://" + filepath.Join(findProjectRoot(), "migrations")

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
