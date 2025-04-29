package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type Level uint8

// "enum" that represents the log levels
const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

// convert the log level to a string
func (l Level) String() string {
	switch l {
	case 0:
		return "INFO"
	case 1:
		return "ERROR"
	case 2:
		return "FATAL"
	case 3:
		return "OFF"
	default:
		return ""
	}
}

type Logger struct {
	out      io.Writer  // where the log will be written
	minLevel Level      // a logger with minLevel = 0 will log everything
	mu       sync.Mutex // manage concurrent logs to the logger writter
}

func New(out io.Writer, level Level) *Logger {
	return &Logger{
		out: out,
		minLevel: level,
	}
}

// methods for writing log entries at different levels

func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

// logs the error and exit the process
func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelOff, err.Error(), properties)
	os.Exit(1)
}

// internal method for writing the log entry
func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	// ex: prevent a logger with Level = FATAL from logging INFO messages
	if level < l.minLevel {
		return 0, nil
	}

	// format the log message in a json format
	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	// add stack track for Error and Fatal Levels
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	// hold the formatted json. if theres any error on the conversion, sets the
	// message of the logger to be a plain-text
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshall log message: " + err.Error())
	}

	// control concurrent acess to the io.Writer
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.out.Write(append(line, '\n'))
}

// satisfies the io.Writer interface. Writes a log entry at the Error Level with no properties
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)
}

