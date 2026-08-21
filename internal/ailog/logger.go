package ailog

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const filePrefix = "ai-"

type Event struct {
	Timestamp  time.Time `json:"timestamp"`
	RequestID  string    `json:"requestId"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	Endpoint   string    `json:"endpoint"`
	DurationMS int64     `json:"durationMs"`
	HTTPStatus int       `json:"httpStatus,omitempty"`
	Success    bool      `json:"success"`
	Stage      string    `json:"stage"`
	Request    string    `json:"request"`
	Response   string    `json:"response,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type Logger struct {
	dir       string
	retention time.Duration
	events    chan Event
	done      chan struct{}
	mu        sync.RWMutex
	closed    bool
}

func New(dir string, retention time.Duration) (*Logger, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("AI log directory is empty")
	}
	if retention <= 0 {
		return nil, fmt.Errorf("AI log retention must be positive")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	l := &Logger{
		dir:       dir,
		retention: retention,
		events:    make(chan Event, 2048),
		done:      make(chan struct{}),
	}
	if err := l.Cleanup(time.Now()); err != nil {
		return nil, err
	}
	go l.run()
	return l, nil
}

// Log only enqueues the event; filesystem writes happen on the logger goroutine.
func (l *Logger) Log(event Event) {
	if l == nil {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.closed {
		return
	}
	l.events <- event
}

func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.closed = true
	close(l.events)
	l.mu.Unlock()
	<-l.done
}

func (l *Logger) Cleanup(now time.Time) error {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return err
	}
	cutoff := now.Add(-l.retention)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), filePrefix) || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(l.dir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func (l *Logger) run() {
	defer close(l.done)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case event, ok := <-l.events:
			if !ok {
				return
			}
			if err := l.write(event); err != nil {
				log.Printf("AI audit log write: %v", err)
			}
		case now := <-ticker.C:
			if err := l.Cleanup(now); err != nil {
				log.Printf("AI audit log cleanup: %v", err)
			}
		}
	}
}

func (l *Logger) write(event Event) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	path := filepath.Join(l.dir, filePrefix+event.Timestamp.Format("2006-01-02")+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return f.Sync()
}
