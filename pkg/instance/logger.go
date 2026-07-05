package instance

import (
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Logger struct {
	mu   sync.Mutex
	file io.WriteCloser
	path string
}

func NewLogger(name, logsDir string) (*Logger, error) {
	if logsDir == "" {
		logsDir = "logs"
	}
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(logsDir, name+".log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f, path: path}, nil
}

func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Write(p)
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Path() string {
	return l.path
}
