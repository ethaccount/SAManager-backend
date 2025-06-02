package testutil

import (
	"os"
	"testing"

	"github.com/ethaccount/backend/src/domain"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	// Auto-migrate the Job table for testing
	if err := db.AutoMigrate(&domain.Job{}); err != nil {
		t.Fatalf("Failed to migrate Job table: %v", err)
	}

	return db
}

func CleanupTestDB(t *testing.T, db *gorm.DB) {
	// Clean up test data
	if err := db.Exec("DELETE FROM jobs").Error; err != nil {
		t.Logf("Warning: Failed to clean up test data: %v", err)
	}
}
