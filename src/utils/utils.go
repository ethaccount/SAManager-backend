package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

func FindProjectRoot() string {
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
