package jobs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// How long a reply may run is the user's call (roadmap 1.7.7 #4).
//
// There used to be a fixed thirty-minute cap on every job, and it stopped
// replies that were still streaming. Now there is no cap unless the page sends
// the user's CONFIG limit as ?max_runtime_s=, which the server enforces too, so
// it still holds with the tab closed. These pin: no cap by default, the
// limit when sent, the grace that lets the page's own stop land first, and a
// malformed limit refused without touching the generation already running.

// streamingUpstream sends a chunk every gap, for chunks chunks, and stops early
// if the job hangs up. It is alive the whole time, so the between-chunks
// watchdog never has a reason to fire: anything that stops it is a total cap.
func streamingUpstream(t *testing.T, gap time.Duration, chunks int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < chunks; i++ {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(gap):
			}
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n"))
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// startJobQuery creates a job with the given query string and returns its id
// and the HTTP status of the create.
func startJobQuery(t *testing.T, m *Manager, query string) (string, int) {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handle(rec, httptest.NewRequest(http.MethodPost, "/llm/jobs?"+query,
		strings.NewReader(`{"messages":[]}`)))
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(rec.Body.Bytes(), &created)
	return created.ID, rec.Code
}

func TestNewManagerHasNoRuntimeCap(t *testing.T) {
	m := NewManager("http://127.0.0.1:1", "", 1, 48)
	defer m.Shutdown()
	if m.maxRuntime != 0 {
		t.Fatalf("a new manager caps every job at %s; the default must be no cap", m.maxRuntime)
	}
	if m.capGrace != pageCapGrace {
		t.Errorf("capGrace = %s, want pageCapGrace (%s)", m.capGrace, pageCapGrace)
	}
}

// An explicit 0 is "no limit", and it applies even over a manager-level cap —
// which is the only way to see from a test that no deadline was set at all.
func TestZeroLimitMeansNone(t *testing.T) {
	upstream := streamingUpstream(t, 60*time.Millisecond, 15) // ~0.9 s
	m := NewManager(upstream.URL, "", 4, 24)
	defer m.Shutdown()
	m.maxRuntime = 300 * time.Millisecond // would cut the stream at a third
	m.idle = 30 * time.Second

	id, code := startJobQuery(t, m, "thread=t1&max_runtime_s=0")
	if code != http.StatusAccepted {
		t.Fatalf("create: got %d, want 202", code)
	}
	status, errMsg := awaitTerminal(t, m, id, 10*time.Second)
	if status != StatusDone {
		t.Fatalf("status = %q (err %q), want %q -- max_runtime_s=0 still capped the job",
			status, errMsg, StatusDone)
	}
}

// A job with no limit of its own runs to the end, on a manager as NewManager
// builds it.
func TestJobWithoutALimitRunsToTheEnd(t *testing.T) {
	upstream := streamingUpstream(t, 60*time.Millisecond, 15)
	m := NewManager(upstream.URL, "", 4, 24)
	defer m.Shutdown()
	m.idle = 30 * time.Second

	id, _ := startJobQuery(t, m, "thread=t1")
	if status, errMsg := awaitTerminal(t, m, id, 10*time.Second); status != StatusDone {
		t.Fatalf("status = %q (err %q), want %q", status, errMsg, StatusDone)
	}
	if got := mustGet(t, m, id).maxRuntime; got != 0 {
		t.Errorf("job.maxRuntime = %s, want 0 (no limit)", got)
	}
}

func TestPageLimitStopsAJob(t *testing.T) {
	upstream := streamingUpstream(t, 50*time.Millisecond, 200) // ~10 s, alive throughout
	m := NewManager(upstream.URL, "", 4, 24)
	defer m.Shutdown()
	m.idle = 30 * time.Second
	m.capGrace = 0

	start := time.Now()
	id, code := startJobQuery(t, m, "thread=t1&max_runtime_s=1")
	if code != http.StatusAccepted {
		t.Fatalf("create: got %d, want 202", code)
	}
	status, errMsg := awaitTerminal(t, m, id, 5*time.Second)
	if status != StatusError {
		t.Fatalf("status = %q, want %q -- the user's limit did not stop the job", status, StatusError)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("stopped after %s, before the 1s limit", elapsed)
	}
	// The message names the user's limit and where it lives -- not a server
	// constant, and not "cancelled", which would claim the user pressed stop.
	if !strings.Contains(errMsg, "1s runtime limit") || !strings.Contains(errMsg, "CONFIG") {
		t.Errorf("error message %q does not name the limit and where it is set", errMsg)
	}
}

// The page's own timer and the server's copy of the same limit would otherwise
// race. The grace makes the page's stop -- with its note -- land first while
// the page is open; the server's copy is for the closed tab.
func TestPageLimitGetsItsGrace(t *testing.T) {
	upstream := streamingUpstream(t, 50*time.Millisecond, 200)
	m := NewManager(upstream.URL, "", 4, 24)
	defer m.Shutdown()
	m.idle = 30 * time.Second
	m.capGrace = time.Second

	id, _ := startJobQuery(t, m, "thread=t1&max_runtime_s=1")
	time.Sleep(1500 * time.Millisecond) // past the limit, inside the grace
	if status, _, _, _, _ := mustGet(t, m, id).snapshot(); status != StatusRunning {
		t.Fatalf("status at 1.5s = %q, want still %q -- no grace was given", status, StatusRunning)
	}
	status, errMsg := awaitTerminal(t, m, id, 3*time.Second)
	if status != StatusError {
		t.Fatalf("status = %q, want %q after limit + grace", status, StatusError)
	}
	// The message reports the limit the user set, not limit + grace.
	if !strings.Contains(errMsg, "1s runtime limit") {
		t.Errorf("error message %q should name the user's 1s, not the grace-padded deadline", errMsg)
	}
}

func TestMalformedLimitIsRefusedBeforeSupersede(t *testing.T) {
	upstream := blockingUpstream(t)
	m := NewManager(upstream.URL, "", 1, 48)
	defer m.Shutdown()

	firstID, _ := createJob(t, m)
	waitForStatus(t, m, firstID, StatusRunning)

	for _, bad := range []string{"abc", "-5", "1.5", "99999999999999999999"} {
		_, code := startJobQuery(t, m, "thread=t1&max_runtime_s="+bad)
		if code != http.StatusBadRequest {
			t.Errorf("max_runtime_s=%s: got %d, want 400", bad, code)
		}
	}

	// Give a wrongly-placed supersede time to land before checking.
	time.Sleep(100 * time.Millisecond)
	if status, _, _, _, _ := mustGet(t, m, firstID).snapshot(); status != StatusRunning {
		t.Errorf("a rejected limit killed the live generation: status is %q", status)
	}
}
