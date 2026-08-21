package ailog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoggerWritesAndFlushes(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir, 48*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 21, 14, 0, 0, 0, time.Local)
	l.Log(Event{Timestamp: now, RequestID: "req-1", Provider: "Claude", Success: true, Request: `{"model":"test"}`, Response: `{"ok":true}`})
	l.Close()

	f, err := os.Open(filepath.Join(dir, "ai-2026-08-21.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var got Event
	if !bufio.NewScanner(f).Scan() {
		t.Fatal("missing log line")
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(f).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.RequestID != "req-1" || !got.Success || got.Provider != "Claude" {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestCleanupOnlyRemovesExpiredAILogs(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir, 48*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	now := time.Date(2026, 8, 21, 14, 0, 0, 0, time.Local)
	old := filepath.Join(dir, "ai-2026-08-18.jsonl")
	recent := filepath.Join(dir, "ai-2026-08-21.jsonl")
	unrelated := filepath.Join(dir, "server.log")
	for _, path := range []string{old, recent, unrelated} {
		if err := os.WriteFile(path, []byte("x"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(old, now.Add(-49*time.Hour), now.Add(-49*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(recent, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(unrelated, now.Add(-72*time.Hour), now.Add(-72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := l.Cleanup(now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("expired AI log still exists: %v", err)
	}
	for _, path := range []string{recent, unrelated} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file to remain: %s: %v", path, err)
		}
	}
}
