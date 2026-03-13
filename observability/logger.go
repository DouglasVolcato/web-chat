package observability

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const logDirectory = "logs"

var (
	requestLogger      *log.Logger
	requestEntryLogger *log.Logger
	requestExitLogger  *log.Logger
	requestOnce        sync.Once
	requestEntryOnce   sync.Once
	requestExitOnce    sync.Once
	sqlLogger          *log.Logger
	sqlOnce            sync.Once
	errorLogger        *log.Logger
	errorOnce          sync.Once
)

// DebugLoggingEnabled indicates whether debug logging should run based on APP_ENV.
func DebugLoggingEnabled() bool {
	return LogsEnabled()
}

// LogsEnabled indicates whether application logs should be emitted.
func LogsEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "dev")
}

// ConfigureRuntimeLogging disables the default Go loggers outside of development.
func ConfigureRuntimeLogging() {
	if LogsEnabled() {
		return
	}

	log.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func ensureLogDir() error {
	return os.MkdirAll(logDirectory, 0o755)
}

func buildLogger(fileName string) (*log.Logger, error) {
	if !LogsEnabled() {
		return log.New(io.Discard, "", 0), nil
	}

	if err := ensureLogDir(); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(filepath.Join(logDirectory, fileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	writer := io.MultiWriter(logFile, os.Stdout)
	return log.New(writer, "", log.LstdFlags|log.Lmicroseconds), nil
}

func resolveLogger(target **log.Logger, initOnce *sync.Once, fileName string) *log.Logger {
	initOnce.Do(func() {
		var err error
		*target, err = buildLogger(fileName)
		if err != nil {
			log.Printf("failed to create %s logger: %v", fileName, err)
			*target = log.Default()
		}
	})

	return *target
}

// RequestLogger returns a logger dedicated to request logging.
func RequestLogger() *log.Logger {
	return resolveLogger(&requestLogger, &requestOnce, "requests.log")
}

// SQLLogger returns a logger dedicated to SQL statement logging.
func SQLLogger() *log.Logger {
	return resolveLogger(&sqlLogger, &sqlOnce, "sql.log")
}

// RequestEntryLogger returns a logger dedicated to inbound request payloads.
func RequestEntryLogger() *log.Logger {
	return resolveLogger(&requestEntryLogger, &requestEntryOnce, "requests_in.log")
}

// RequestExitLogger returns a logger dedicated to outbound responses.
func RequestExitLogger() *log.Logger {
	return resolveLogger(&requestExitLogger, &requestExitOnce, "requests_out.log")
}

// ErrorLogger returns a logger dedicated to error tracking.
func ErrorLogger() *log.Logger {
	return resolveLogger(&errorLogger, &errorOnce, "errors.log")
}
