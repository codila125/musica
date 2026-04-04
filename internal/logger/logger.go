package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *Logger
	once     sync.Once
)

type Logger struct {
	file  *os.File
	mu    sync.Mutex
	level string
}

func Get() *Logger {
	once.Do(func() {
		instance = newLogger()
	})
	return instance
}

func newLogger() *Logger {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Logger{}
	}

	logDir := filepath.Join(home, ".config", "musica")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return &Logger{}
	}

	logFile := filepath.Join(logDir, "musica.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return &Logger{}
	}

	return &Logger{file: f}
}

func (l *Logger) SetLevel(level string) {
	l.level = level
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.write("DEBUG", format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.write("INFO", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.write("ERROR", format, args...)
}

func (l *Logger) write(level, format string, args ...interface{}) {
	if l.file == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, fmt.Sprintf(format, args...))

	l.mu.Lock()
	defer l.mu.Unlock()
	l.file.WriteString(msg)
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
