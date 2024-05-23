package logger

import (
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
)

var (
	sugar *zap.SugaredLogger
	once  sync.Once
)

func GetLogger() *zap.SugaredLogger {
	once.Do(func() {
		var logger *zap.Logger
		var err error

		// By default, use production configuration - JSON logging and
		// debug logs are omitted.
		//
		// For local development set the DEBUG environment variable to any
		// non-empty value, this will enable terminal-formatted output
		// and debug level logs will be included.
		if os.Getenv("DEBUG") != "" {
			logger, err = zap.NewDevelopment()
		} else {
			logger, err = zap.NewProduction()
		}

		if err != nil {
			log.Panic(err) // Use stdlib log in case the zap logger fails to initialise
		}

		sugar = logger.Sugar()
	})

	return sugar
}
