package testutil

import (
	"os"
	"path/filepath"

	"github.com/ethaccount/backend/src/utils"
	"github.com/joho/godotenv"
)

func GetEnv(key string) string {
	err := godotenv.Load(filepath.Join(utils.FindProjectRoot(), ".env"))
	if err != nil {
		panic("Error loading .env file")
	}

	return os.Getenv(key)
}
