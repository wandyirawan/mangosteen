// Package logger provides structured logging with hourly file rotation.
// Logs are written in JSON format for compatibility with Loki and other log aggregation systems.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// HourlyRotator handles hourly file rotation for log files.
type HourlyRotator struct {
	basePath    string
	currentFile *os.File
	currentHour time.Time
	mu          sync.RWMutex
	onRotate    func(oldFile string) // Callback when file is rotated
}

// NewHourlyRotator creates a new hourly file rotator.
func NewHourlyRotator(basePath string, onRotate func(oldFile string)) (*HourlyRotator, error) {
	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	r := &HourlyRotator{
		basePath: basePath,
		onRotate: onRotate,
	}

	// Open initial file
	now := time.Now().Truncate(time.Hour)
	if err := r.rotate(now); err != nil {
		return nil, err
	}

	return r, nil
}

// Write implements io.Writer interface.
func (r *HourlyRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Truncate(time.Hour)
	if now != r.currentHour {
		if err := r.rotate(now); err != nil {
			return 0, err
		}
	}

	return r.currentFile.Write(p)
}

// rotate switches to a new file for the given hour.
func (r *HourlyRotator) rotate(hour time.Time) error {
	// Close current file if exists
	if r.currentFile != nil {
		oldFile := r.currentFile.Name()
		r.currentFile.Close()
		r.currentFile = nil

		// Trigger callback for the closed file
		if r.onRotate != nil {
			go r.onRotate(oldFile)
		}
	}

	// Generate filename with hour precision
	filename := fmt.Sprintf("app-%s.log", hour.Format("2006-01-02-15"))
	filepath := filepath.Join(r.basePath, filename)

	// Open or create file
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	r.currentFile = f
	r.currentHour = hour

	return nil
}

// Close closes the current log file.
func (r *HourlyRotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentFile != nil {
		return r.currentFile.Close()
	}
	return nil
}

// GetCurrentFile returns the path of the currently active log file.
func (r *HourlyRotator) GetCurrentFile() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentFile != nil {
		return r.currentFile.Name()
	}
	return ""
}

// Config holds configuration for the logger.
type Config struct {
	BasePath string
	Level    string
	Console  bool // If true, also output to console
	OnRotate func(oldFile string)
}

// Logger wraps zerolog with file rotation capabilities.
type Logger struct {
	rotator *HourlyRotator
	logger  zerolog.Logger
}

// New creates a new logger with hourly file rotation.
func New(cfg Config) (*Logger, error) {
	rotator, err := NewHourlyRotator(cfg.BasePath, cfg.OnRotate)
	if err != nil {
		return nil, err
	}

	// Parse log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Setup outputs
	var writers []io.Writer
	writers = append(writers, rotator)

	if cfg.Console {
		// Pretty console output for development
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	}

	var baseLogger zerolog.Logger
	if len(writers) > 1 {
		baseLogger = zerolog.New(io.MultiWriter(writers...))
	} else {
		baseLogger = zerolog.New(rotator)
	}

	baseLogger = baseLogger.With().Timestamp().Logger()

	// Set as global logger
	log.Logger = baseLogger

	return &Logger{
		rotator: rotator,
		logger:  baseLogger,
	}, nil
}

// GetLogger returns the underlying zerolog instance.
func (l *Logger) GetLogger() zerolog.Logger {
	return l.logger
}

// Debug logs a debug message.
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info logs an info message.
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn logs a warning message.
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error logs an error message.
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// Close closes the logger and its underlying file.
func (l *Logger) Close() error {
	return l.rotator.Close()
}

// GetCurrentFile returns the path of the currently active log file.
func (l *Logger) GetCurrentFile() string {
	return l.rotator.GetCurrentFile()
}
