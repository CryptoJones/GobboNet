package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/auth"
)

func authedServer(t *testing.T) *Server {
	t.Helper()
	srv, _ := newTestServer(t)
	secret, err := auth.NewSecret("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	srv.cfg.RequireAuth = true
	srv.secret = secret
	return srv
}

func signIn(t *testing.T, srv *Server, remote, ua string) *http.Cookie {
	t.Helper()
	req := newReq(http.MethodPost, "/login", strings.NewReader("password=correct+horse+battery"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr, req.Header["User-Agent"] = remote, []string{ua}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("login: %d", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	t.Fatal("login set no session cookie")
	return nil
}

func probe(srv *Server, cookie *http.Cookie, remote, ua string) int {
	req := newReq(http.MethodGet, "/health-fileserver", nil)
	req.RemoteAddr, req.Header["User-Agent"] = remote, []string{ua}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec.Code
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev, flags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(prev); log.SetFlags(flags) })
	return &buf
}

// The "signed in, then asked again" report: one browser on this PC reaching
// the server as 127.0.0.1 and as ::1, or with a changed browser string, is
// still the browser that signed in.
func TestLoopbackSessionSurvivesAddressFamilyAndBrowserString(t *testing.T) {
	srv := authedServer(t)
	logs := captureLog(t)
	c := signIn(t, srv, "127.0.0.1:50000", "Chrome/141")
	for _, at := range []struct{ remote, ua string }{
		{"127.0.0.1:50001", "Chrome/141"},
		{"[::1]:50002", "Chrome/141"},
		{"127.0.0.1:50003", "Chrome/141 (changed by an extension)"},
	} {
		if code := probe(srv, c, at.remote, at.ua); code != http.StatusOK {
			t.Errorf("signed in on this PC, then %s %q: got %d, want 200", at.remote, at.ua, code)
		}
	}
	if strings.Contains(logs.String(), "[auth]") {
		t.Errorf("nothing was refused, so nothing should be logged:\n%s", logs)
	}
}

// From the network the binding is unchanged, and a refusal says why -- once.
func TestNetworkSessionStaysBoundAndRefusalIsLoggedOnce(t *testing.T) {
	srv := authedServer(t)
	logs := captureLog(t)
	c := signIn(t, srv, "192.168.1.5:50000", "Safari")
	if code := probe(srv, c, "192.168.1.5:50001", "Safari"); code != http.StatusOK {
		t.Fatalf("same phone: got %d", code)
	}
	for i := 0; i < 3; i++ {
		if code := probe(srv, c, "192.168.1.6:50001", "Safari"); code != http.StatusUnauthorized {
			t.Fatalf("another device replaying the cookie: got %d, want 401", code)
		}
	}
	if code := probe(srv, c, "192.168.1.5:50002", "Firefox"); code != http.StatusUnauthorized {
		t.Fatalf("another browser at the phone's address: got %d, want 401", code)
	}
	out := logs.String()
	if n := strings.Count(out, "different network address or browser"); n != 1 {
		t.Errorf("a refused session is reported once, not per request: %d lines\n%s", n, out)
	}
	if !strings.Contains(out, "GET /health-fileserver") {
		t.Errorf("the line names the request:\n%s", out)
	}
	if strings.Contains(out, c.Value) {
		t.Error("the log must never contain the session token")
	}

	logs.Reset()
	dead := &http.Cookie{Name: auth.CookieName, Value: "from-before-a-restart"}
	probe(srv, dead, "127.0.0.1:1", "Chrome")
	probe(srv, dead, "127.0.0.1:2", "Chrome")
	if n := strings.Count(logs.String(), "this server did not issue"); n != 1 {
		t.Errorf("a token from before a restart is reported once: %d\n%s", n, logs)
	}
	logs.Reset()
	probe(srv, nil, "127.0.0.1:3", "Chrome")
	if logs.Len() != 0 {
		t.Errorf("no cookie at all is not worth a line:\n%s", logs)
	}
}
