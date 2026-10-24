package proxy

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// logCapture collects what the proxy logs.
type logCapture struct {
	mu    sync.Mutex
	lines []string
}

func (c *logCapture) logf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, fmt.Sprintf(format, args...))
}

func (c *logCapture) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.lines...)
}

// freePort returns a loopback port with nothing listening on it.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func get(p *Proxy, path string) int {
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

// The chat page's five-second /llm/health poll against an engine that is not
// running: one log line for the run, not one per poll (issue #63). Then the
// engine comes up and the log says so once, with the count.
func TestRefusedPollsLogOnceThenRecovery(t *testing.T) {
	addr := freePort(t)
	p, err := New("/llm", "http://"+addr, "")
	if err != nil {
		t.Fatal(err)
	}
	var logs logCapture
	p.fails.logf = logs.logf

	for i := 0; i < 10; i++ {
		if code := get(p, "/llm/health"); code != http.StatusBadGateway {
			t.Fatalf("poll %d: got %d, want 502 -- the client must still be told every time", i, code)
		}
	}
	// A different path failing against the same dead upstream is the same run.
	if code := get(p, "/llm/v1/models"); code != http.StatusBadGateway {
		t.Fatalf("models: got %d, want 502", code)
	}
	lines := logs.snapshot()
	if len(lines) != 1 || !strings.Contains(lines[0], "[proxy] GET /health ->") {
		t.Fatalf("11 refused requests logged %d line(s), want 1:\n%s", len(lines), strings.Join(lines, "\n"))
	}

	// The engine starts.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("could not re-bind %s for the recovery half: %v", addr, err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})}
	go srv.Serve(l)
	defer srv.Close()

	if code := get(p, "/llm/health"); code != http.StatusOK {
		t.Fatalf("after the engine started: got %d, want 200", code)
	}
	get(p, "/llm/health")
	lines = logs.snapshot()
	want := []string{"repeated 10 more time(s)", "is answering again"}
	if len(lines) != 3 || !strings.Contains(lines[1], want[0]) || !strings.Contains(lines[2], want[1]) {
		t.Fatalf("after recovery got:\n%s\nwant the first line, then %q, then %q, and nothing for later successes",
			strings.Join(lines, "\n"), want[0], want[1])
	}
	if strings.Contains(lines[2], "@") {
		t.Errorf("the recovery line must not print URL credentials: %s", lines[2])
	}

	// It goes away again: a new run starts with a fresh first line.
	srv.Close()
	get(p, "/llm/health")
	if lines = logs.snapshot(); len(lines) != 4 || !strings.Contains(lines[3], "[proxy] GET /health ->") {
		t.Fatalf("a second outage should log its own first line:\n%s", strings.Join(lines, "\n"))
	}
}

// A healthy upstream logs nothing at all.
func TestHealthyUpstreamLogsNothing(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer up.Close()
	p, err := New("/llm", up.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	var logs logCapture
	p.fails.logf = logs.logf
	for i := 0; i < 5; i++ {
		get(p, "/llm/health")
	}
	if lines := logs.snapshot(); len(lines) != 0 {
		t.Fatalf("got %v", lines)
	}
}

// A different failure is its own run, and the old run's count is closed first,
// so nothing is lost when the reason changes.
func TestChangingFailureClosesTheRun(t *testing.T) {
	var logs logCapture
	f := failLog{logf: logs.logf}
	refused := errors.New("dial tcp 127.0.0.1:8080: connect: connection refused")
	timeout := errors.New("net/http: timeout awaiting response headers")
	f.failed("GET", "/health", refused)
	f.failed("GET", "/health", refused)
	f.failed("GET", "/health", refused)
	f.failed("POST", "/v1/chat/completions", timeout)
	f.failed("POST", "/v1/chat/completions", timeout)
	f.answered("http://127.0.0.1:8080")
	f.answered("http://127.0.0.1:8080")
	got := strings.Join(logs.snapshot(), "\n")
	want := strings.Join([]string{
		"[proxy] GET /health -> dial tcp 127.0.0.1:8080: connect: connection refused",
		"[proxy] (that failure repeated 2 more time(s))",
		"[proxy] POST /v1/chat/completions -> net/http: timeout awaiting response headers",
		"[proxy] (that failure repeated 1 more time(s))",
		"[proxy] http://127.0.0.1:8080 is answering again",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
