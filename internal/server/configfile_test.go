package server

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ElodineOfficial/GobboNet/internal/auth"
	"github.com/ElodineOfficial/GobboNet/internal/config"
)

const cfSecret = "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA"

func configFileServer(t *testing.T) (*Server, string) {
	t.Helper()
	s, cfg := newTestServer(t)
	body := "# keep me\nllm_url = \"http://127.0.0.1:1\"\nllm_api_key = \"sk-secret-upstream\"\n" +
		"listen_host = \"127.0.0.1\"\nlisten_port = 8080\naccess_secret = \"" + cfSecret + "\"\nsession_ttl_hours = 12\n"
	if err := os.WriteFile(cfg.Path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return s, cfg.Path
}

type cfReq struct {
	method, path, remote, host, origin, fetchSite, body string
	headers                                             map[string]string
}

func (c cfReq) do(s *Server) *httptest.ResponseRecorder {
	if c.path == "" {
		c.path = "/config/file"
	}
	r := newReq(c.method, c.path, strings.NewReader(c.body))
	r.RemoteAddr = c.remote
	if c.host != "" {
		r.Host = c.host
	}
	if c.origin != "" {
		r.Header.Set("Origin", c.origin)
	}
	if c.fetchSite != "" {
		r.Header.Set("Sec-Fetch-Site", c.fetchSite)
	}
	for k, v := range c.headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

const local = "127.0.0.1:50123"

func openConfig(t *testing.T, s *Server) map[string]any {
	t.Helper()
	w := cfReq{method: "GET", remote: local}.do(s)
	if w.Code != 200 {
		t.Fatalf("open: %d %s", w.Code, w.Body)
	}
	return decode(t, w)
}

func saveConfig(s *Server, text, version string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(map[string]string{"text": text, "version": version})
	return cfReq{method: "POST", remote: local, origin: "http://127.0.0.1:8080", body: string(b)}.do(s)
}

func TestConfigFileOnlyFromThisPC(t *testing.T) {
	s, path := configFileServer(t)

	got := openConfig(t, s)
	text, _ := got["text"].(string)
	if got["editable"] != true || got["path"] != path || !strings.Contains(text, "# keep me") {
		t.Fatalf("this PC should get the file: %v", got)
	}
	if strings.Contains(text, cfSecret) || strings.Contains(text, "sk-secret-upstream") {
		t.Fatal("a secret was sent to the browser")
	}

	// Everyone else: told where to open it, and nothing of the file.
	for name, c := range map[string]cfReq{
		"a phone on the LAN":               {method: "GET", remote: "192.168.1.40:50000"},
		"IPv6 LAN":                         {method: "GET", remote: "[fe80::1]:50000"},
		"this PC by its LAN address":       {method: "GET", remote: "127.0.0.1:50000", host: "192.168.1.5:8080"},
		"a reverse proxy on this PC":       {method: "GET", remote: local, headers: map[string]string{"X-Forwarded-For": "100.64.0.9"}},
		"a proxy with the standard header": {method: "GET", remote: local, headers: map[string]string{"Forwarded": "for=100.64.0.9"}},
	} {
		w := c.do(s)
		body := w.Body.String()
		if w.Code != 200 || strings.Contains(body, "keep me") || strings.Contains(body, path) || !strings.Contains(body, `"editable":false`) {
			t.Errorf("%s: %d %s", name, w.Code, body)
		}
		if !strings.Contains(body, "http://127.0.0.1:") {
			t.Errorf("%s: should say where to open it: %s", name, body)
		}
	}
	// localhost and ::1 are this PC too.
	for _, c := range []cfReq{{method: "GET", remote: local, host: "localhost:8080"}, {method: "GET", remote: "[::1]:50000", host: "[::1]:8080"}} {
		if w := c.do(s); !strings.Contains(w.Body.String(), `"editable":true`) {
			t.Errorf("%s via %s: %s", c.remote, c.host, w.Body)
		}
	}

	// Another site cannot read it, even from this PC (every response is CORS *).
	for _, c := range []cfReq{
		{method: "GET", remote: local, origin: "http://attacker.test"},
		{method: "GET", remote: local, fetchSite: "cross-site"},
		{method: "GET", remote: local, fetchSite: "same-site"},
	} {
		if w := c.do(s); w.Code != 403 || strings.Contains(w.Body.String(), "keep me") {
			t.Errorf("cross-site read (%s %s): %d %s", c.origin, c.fetchSite, w.Code, w.Body)
		}
	}
}

func TestConfigFileSaveRules(t *testing.T) {
	s, path := configFileServer(t)
	before, _ := os.ReadFile(path)
	got := openConfig(t, s)
	text, version := got["text"].(string), got["version"].(string)
	edited := strings.Replace(text, "session_ttl_hours = 12", "session_ttl_hours = 4", 1)
	body, _ := json.Marshal(map[string]string{"text": edited, "version": version})

	// Refused: from elsewhere, from another site, or with no Origin at all.
	for name, c := range map[string]cfReq{
		"from the LAN":         {method: "POST", remote: "192.168.1.40:5000", origin: "http://192.168.1.5:8080", host: "192.168.1.5:8080"},
		"from another site":    {method: "POST", remote: local, origin: "http://attacker.test"},
		"another port":         {method: "POST", remote: local, origin: "http://127.0.0.1:8081"},
		"no Origin":            {method: "POST", remote: local},
		"cross-site fetch":     {method: "POST", remote: local, origin: "http://127.0.0.1:8080", fetchSite: "cross-site"},
		"through a proxy":      {method: "POST", remote: local, origin: "http://127.0.0.1:8080", headers: map[string]string{"X-Forwarded-For": "10.0.0.2"}},
		"restore from the LAN": {method: "POST", path: "/config/file/restore", remote: "192.168.1.40:5000", origin: "http://192.168.1.5:8080", host: "192.168.1.5:8080"},
	} {
		c.body = string(body)
		if w := c.do(s); w.Code != 403 {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}
	if now, _ := os.ReadFile(path); string(now) != string(before) {
		t.Fatal("a refused save changed the file")
	}

	// Saved from this PC.
	w := saveConfig(s, edited, version)
	if w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	out := decode(t, w)
	now, _ := os.ReadFile(path)
	if !strings.Contains(string(now), "session_ttl_hours = 4") || !strings.Contains(string(now), cfSecret) ||
		!strings.Contains(string(now), `llm_api_key = "sk-secret-upstream"`) || !strings.Contains(string(now), "# keep me") {
		t.Errorf("saved file:\n%s", now)
	}
	if ch, _ := json.Marshal(out["changed"]); string(ch) != `["session_ttl_hours"]` || out["has_backup"] != true {
		t.Errorf("response: %v", out)
	}
	if bak, _ := os.ReadFile(config.BackupPath(path)); string(bak) != string(before) {
		t.Error("backup is not the file before the save")
	}

	// The version it was opened at is now stale.
	if w := saveConfig(s, edited, version); w.Code != 409 {
		t.Errorf("stale save: %d %s", w.Code, w.Body)
	}
	got = openConfig(t, s)
	text, version = got["text"].(string), got["version"].(string)
	if got["has_backup"] != true {
		t.Error("has_backup")
	}

	// What would stop the next start is refused, and nothing is written.
	mid, _ := os.ReadFile(path)
	for name, change := range map[string][2]string{
		"bad TOML":            {"session_ttl_hours = 4", "session_ttl_hours = ="},
		"password change":     {`access_secret = "(hidden)"`, `access_secret = "x"  #`},
		"engine port":         {"listen_port = 8080", "listen_port = 11437"},
		"address not this PC": {`listen_host = "127.0.0.1"`, `listen_host = "203.0.113.7"`},
		"below 1024":          {"listen_port = 8080", "listen_port = 80"},
	} {
		w := saveConfig(s, strings.Replace(text, change[0], change[1], 1), version)
		if w.Code != 400 || !strings.HasPrefix(decode(t, w)["error"].(string), "Not saved: ") {
			t.Errorf("%s: %d %s", name, w.Code, w.Body)
		}
	}
	// A port something else is using.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := ln.Addr().(*net.TCPAddr).Port
	if busy >= config.MinPanelPort && busy <= config.MaxPanelPortFor("linux") {
		w := saveConfig(s, strings.Replace(text, "listen_port = 8080", "listen_port = "+strconv.Itoa(busy), 1), version)
		if w.Code != 400 {
			t.Errorf("busy port: %d %s", w.Code, w.Body)
		}
	}
	if now, _ := os.ReadFile(path); string(now) != string(mid) {
		t.Fatal("a refused save changed the file")
	}

	// Undo: back to the file before the first save, password as it is now.
	b2, _ := json.Marshal(map[string]string{"version": version})
	w = cfReq{method: "POST", path: "/config/file/restore", remote: local, origin: "http://127.0.0.1:8080", body: string(b2)}.do(s)
	if w.Code != 200 || decode(t, w)["restored"] != true {
		t.Fatalf("restore: %d %s", w.Code, w.Body)
	}
	if now, _ := os.ReadFile(path); !strings.Contains(string(now), "session_ttl_hours = 12") || !strings.Contains(string(now), cfSecret) {
		t.Errorf("restored:\n%s", now)
	}

	// Methods.
	if w := (cfReq{method: "DELETE", remote: local}).do(s); w.Code != 405 {
		t.Errorf("DELETE: %d", w.Code)
	}
	if w := (cfReq{method: "GET", path: "/config/file/restore", remote: local}).do(s); w.Code != 405 {
		t.Errorf("GET restore: %d", w.Code)
	}
}

func TestConfigFileNeedsTheSession(t *testing.T) {
	s, _ := configFileServer(t)
	secret, err := auth.NewSecret("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	s.cfg.RequireAuth = true
	s.secret = secret
	w := cfReq{method: "GET", remote: local}.do(s)
	if w.Code != 401 || strings.Contains(w.Body.String(), "keep me") {
		t.Errorf("signed out: %d %s", w.Code, w.Body)
	}
}

// reply_start_timeout_seconds reaches the jobs every reply uses and the /llm
// route lore compression uses, and nothing else.
func TestReplyStartTimeoutWiring(t *testing.T) {
	_, cfg := newTestServer(t)
	cfg.ReplyStartTimeoutSeconds = 1234
	s, err := New(cfg, config.ModeRemote, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := 1234 * time.Second
	if s.jobs.ReplyStartTimeout() != want || s.llmProxy.ResponseHeaderTimeout() != want {
		t.Errorf("jobs %s, /llm %s; want %s", s.jobs.ReplyStartTimeout(), s.llmProxy.ResponseHeaderTimeout(), want)
	}
	if s.searchProxy.ResponseHeaderTimeout() != 5*time.Minute || s.embedProxy.ResponseHeaderTimeout() != 5*time.Minute {
		t.Error("/search and /embed keep their own default")
	}
}
