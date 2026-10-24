package proxy

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// /llm takes reply_start_timeout_seconds; the default stays for the rest.
func TestResponseHeaderTimeoutIsSettable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() { // accept, then say nothing
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()

	p, err := New("/llm", "http://"+ln.Addr().String(), "")
	if err != nil {
		t.Fatal(err)
	}
	if p.ResponseHeaderTimeout() != 5*time.Minute {
		t.Fatalf("default = %s", p.ResponseHeaderTimeout())
	}
	p.SetResponseHeaderTimeout(300 * time.Millisecond)
	p.fails.logf = func(string, ...any) {}

	w := httptest.NewRecorder()
	start := time.Now()
	p.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/llm/v1/chat/completions", strings.NewReader("{}")))
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("the 300ms bound did not apply: %s", took)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusBadGateway || !strings.Contains(body["detail"], "no reply started within 300ms") ||
		!strings.Contains(body["detail"], "reply_start_timeout_seconds") {
		t.Errorf("%d %v", w.Code, body)
	}
	p.SetResponseHeaderTimeout(0)
	if p.ResponseHeaderTimeout() != 300*time.Millisecond {
		t.Error("0 must leave the bound alone")
	}
}
