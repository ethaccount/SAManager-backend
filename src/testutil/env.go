package testutil

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func GetEnv(key string) string {
	err := godotenv.Load(filepath.Join(findProjectRoot(), ".env"))
	if err != nil {
		panic("Error loading .env file")
	}

	return os.Getenv(key)
}
