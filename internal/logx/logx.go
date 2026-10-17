// Package logx writes a dated UTF-8 log next to the executable.
package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codex-app-proxy/internal/config"
)

// maxFileSize caps a single log file before it is rotated to "<name>.1".
const maxFileSize = 2 << 20 // 2 MiB

// Logger is the default process logger.
type Logger struct {
	mu   sync.Mutex
	path string
	mode config.LogMode
	echo bool
}

var std = &Logger{mode: config.LogErrors, echo: true}

// Init points the default logger at dir using a date-stamped file name.
func Init(dir string) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.path = filepath.Join(dir, "proxy-"+time.Now().Format("2006-01-02")+".log")
}

// SetMode changes how much is persisted.
func SetMode(mode config.LogMode) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.mode = mode
}

// Mode returns the active log mode.
func Mode() config.LogMode {
	std.mu.Lock()
	defer std.mu.Unlock()
	return std.mode
}

// Path returns the active log file path, empty when logging is disabled.
func Path() string {
	std.mu.Lock()
	defer std.mu.Unlock()
	return std.path
}

// SetEcho controls whether lines are also printed to stdout.
func SetEcho(enabled bool) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.echo = enabled
}

// Info records an informational line; persisted only in "always" mode.
func Info(format string, args ...any) {
	std.write("INFO", fmt.Sprintf(format, args...))
}

// Error records a failure; persisted in "errors" and "always" mode.
func Error(format string, args ...any) {
	std.write("ERROR", fmt.Sprintf(format, args...))
}

func (l *Logger) write(level, line string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.echo {
		fmt.Printf("[%s] %s\n", level, line)
	}

	if l.mode == config.LogOff || l.path == "" {
		return
	}
	if l.mode == config.LogErrors && level != "ERROR" {
		return
	}

	l.rotateLocked()

	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	stamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(file, "%s [%s] %s\n", stamp, level, line)
}

func (l *Logger) rotateLocked() {
	info, err := os.Stat(l.path)
	if err != nil || info.Size() < maxFileSize {
		return
	}
	backup := l.path + ".1"
	_ = os.Remove(backup)
	_ = os.Rename(l.path, backup)
}
