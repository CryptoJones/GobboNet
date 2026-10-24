package jobs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// reply_start_timeout_seconds: how long a reply may take to start.
func TestReplyStartTimeoutIsConfigurable(t *testing.T) {
	if m := NewManager("http://127.0.0.1:1", "", 1, 1); m.ReplyStartTimeout() != 5*time.Minute {
		t.Fatalf("default = %s, want the old 5 minutes", m.ReplyStartTimeout())
	}

	// A silent upstream: the configured bound ends it, and the message says
	// how long it waited and which setting moves it.
	m := NewManager(silentListener(t), "", 4, 24)
	m.SetReplyStartTimeout(400 * time.Millisecond)
	if m.ReplyStartTimeout() != 400*time.Millisecond {
		t.Fatal(m.ReplyStartTimeout())
	}
	status, msg := awaitTerminal(t, m, startJob(t, m), 5*time.Second)
	if status != StatusError {
		t.Fatalf("status %q", status)
	}
	for _, want := range []string{"did not start replying within 400ms", "reply_start_timeout_seconds", "CONFIG > Config File"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q: %s", want, msg)
		}
	}

	// A slow start inside the bound is a reply, not a failure.
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer slow.Close()
	m2 := NewManager(slow.URL, "", 4, 24)
	m2.SetReplyStartTimeout(2 * time.Second)
	if status, msg := awaitTerminal(t, m2, startJob(t, m2), 5*time.Second); status != StatusDone {
		t.Errorf("slow start within the bound: %q %s", status, msg)
	}

	m2.SetReplyStartTimeout(0)
	if m2.ReplyStartTimeout() != 2*time.Second {
		t.Error("0 must leave the bound alone")
	}
}
