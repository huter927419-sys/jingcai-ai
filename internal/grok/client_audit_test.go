package grok

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jingcai-ai/internal/ailog"
)

func TestAnalyzeWritesAuditLogWithoutAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"lambda_home\":1.2,\"lambda_away\":0.8,\"headline\":\"test\"}"}}]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	audit, err := ailog.New(dir, 48*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	client := NewNamed("Claude", "secret-key", server.URL, "claude-test")
	client.Audit = audit
	if _, err := client.Analyze("match prompt"); err != nil {
		t.Fatal(err)
	}
	audit.Close()

	event := readAuditEvent(t, dir)
	if !event.Success || event.Stage != "complete" || event.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected audit result: %+v", event)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "ai-"+event.Timestamp.Format("2006-01-02")+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-key") || strings.Contains(strings.ToLower(string(raw)), "authorization") {
		t.Fatal("audit log contains credentials or authorization header")
	}
	if !strings.Contains(event.Request, "match prompt") || !strings.Contains(event.Response, "lambda_home") {
		t.Fatalf("request or response missing: %+v", event)
	}
}

func TestAnalyzeLogsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	audit, err := ailog.New(dir, 48*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	client := NewNamed("Claude", "secret-key", server.URL, "claude-test")
	client.Audit = audit
	if _, err := client.Analyze("match prompt"); err == nil {
		t.Fatal("expected HTTP error")
	}
	audit.Close()

	event := readAuditEvent(t, dir)
	if event.Success || event.Stage != "http_status" || event.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("unexpected audit failure: %+v", event)
	}
	if !strings.Contains(event.Error, "HTTP 429") || !strings.Contains(event.Response, "rate limited") {
		t.Fatalf("failure details missing: %+v", event)
	}
}

func readAuditEvent(t *testing.T, dir string) ailog.Event {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "ai-*.jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("unexpected audit files: %v %v", files, err)
	}
	f, err := os.Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("missing audit line")
	}
	var event ailog.Event
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	return event
}
