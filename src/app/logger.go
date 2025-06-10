package app

import (
	"os"

	"github.com/rs/zerolog"
)

func InitLogger(levelStr string) zerolog.Logger {
	// Set global log level
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Add color and formatting
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: "2006-01-02 15:04:05",
	}

	logger := zerolog.New(output).With().
		Timestamp().
		Logger()

	return logger
}
