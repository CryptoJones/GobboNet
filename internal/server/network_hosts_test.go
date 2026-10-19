package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/auth"
	"github.com/ElodineOfficial/GobboNet/internal/config"
)

func networkRequest(s *Server, method, host, origin, body string) *httptest.ResponseRecorder {
	r := newReq(method, "/network/hosts", strings.NewReader(body))
	r.Host = host
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

func TestNetworkHostsDuckDNSAndTailscale(t *testing.T) {
	s, cfg := newTestServer(t)
	const ip = "127.0.0.1:8080"
	const names = `{"allowed_hosts":["https://My-Goblin.DuckDNS.org:9066/","desktop.tail123.ts.net","desktop"]}`
	for _, h := range []string{"my-goblin.duckdns.org:9066", "desktop.tail123.ts.net:9066", "desktop:9066"} {
		w := networkRequest(s, "GET", h, "", "")
		if w.Code != 421 || !strings.Contains(decode(t, w)["detail"].(string), "CONFIG > Network access") {
			t.Fatalf("expected actionable rejection: %d %s", w.Code, w.Body)
		}
	}
	w := networkRequest(s, "POST", ip, "http://"+ip, names)
	if w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	loaded, err := config.Load(cfg.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Restart with the saved config, preserving test-only paths.
	cfg.AllowedHosts = loaded.AllowedHosts
	restarted, err := New(cfg, config.ModeRemote, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, server := range []*Server{s, restarted} {
		for _, h := range []string{"my-goblin.duckdns.org:9066", "MY-GOBLIN.DUCKDNS.ORG.:9066", "desktop.tail123.ts.net:9066", "desktop:9066", ip} {
			if w := networkRequest(server, "GET", h, "", ""); w.Code != 200 {
				t.Errorf("%s: %d %s", h, w.Code, w.Body)
			}
		}
		for _, h := range []string{"other.duckdns.org", "other.tail123.ts.net", "attacker.test"} {
			if w := networkRequest(server, "GET", h, "", ""); w.Code != 421 {
				t.Errorf("unrelated name %s accepted: %d", h, w.Code)
			}
		}
	}
	// Prevent locking the user out of the address used for this edit.
	if w := networkRequest(s, "POST", "my-goblin.duckdns.org:9066", "http://my-goblin.duckdns.org:9066", `{"allowed_hosts":[]}`); w.Code != 400 {
		t.Fatalf("self-removal: %d", w.Code)
	}
	if w := networkRequest(s, "POST", ip, "http://"+ip, `{"allowed_hosts":[]}`); w.Code != 200 {
		t.Fatalf("removal via IP: %d", w.Code)
	}
	if s.hostAllowed("my-goblin.duckdns.org") {
		t.Error("removed name still trusted")
	}
}

func TestNetworkHostsRejectUnsafeWrites(t *testing.T) {
	s, cfg := newTestServer(t)
	before, _ := os.ReadFile(cfg.Path)
	for _, origin := range []string{"", "null", "http://attacker.test", "http://127.0.0.1:8080.attacker.test", "http://127.0.0.1:8081", "http://127.0.0.1:8080/path"} {
		w := networkRequest(s, "POST", "127.0.0.1:8080", origin, `{"allowed_hosts":["attacker.test"]}`)
		if w.Code != 403 {
			t.Errorf("origin %q: %d", origin, w.Code)
		}
	}
	for _, body := range []string{`{}`, `{"allowed_hosts":null}`, `{"allowed_hosts":"*"}`, `{"allowed_hosts":["*.duckdns.org"]}`, `{"allowed_hosts":["safe.test","invalid/name"]}`, `{"allowed_hosts":[]} {}`, strings.Repeat(" ", 17<<10)} {
		if w := networkRequest(s, "POST", "127.0.0.1:8080", "http://127.0.0.1:8080", body); w.Code != 400 {
			t.Errorf("bad body accepted: %d", w.Code)
		}
	}
	after, _ := os.ReadFile(cfg.Path)
	if string(before) != string(after) {
		t.Fatal("rejected writes changed config")
	}
	if w := networkRequest(s, "DELETE", "127.0.0.1:8080", "", ""); w.Code != 405 {
		t.Errorf("method: %d", w.Code)
	}
	// Even a correctly shaped save through an unapproved Host is blocked.
	if w := networkRequest(s, "POST", "attacker.test", "http://attacker.test", `{"allowed_hosts":["attacker.test"]}`); w.Code != 421 {
		t.Errorf("self-approval: %d", w.Code)
	}
}

func TestNetworkHostsAuthAndSaveFailure(t *testing.T) {
	s, cfg := newTestServer(t)
	s.cfg.RequireAuth = true
	secret, err := auth.NewSecret("test network settings password")
	if err != nil {
		t.Fatal(err)
	}
	s.secret = secret
	for _, method := range []string{"GET", "POST"} {
		if w := networkRequest(s, method, "127.0.0.1:8080", "http://127.0.0.1:8080", `{"allowed_hosts":["my-goblin.duckdns.org"]}`); w.Code != 401 {
			t.Errorf("unauthenticated %s: %d", method, w.Code)
		}
	}
	// Existing sessions can save; custom hostnames do not bypass login.
	r := newReq("POST", "/network/hosts", strings.NewReader(`{"allowed_hosts":["my-goblin.duckdns.org"]}`))
	r.Header.Set("Origin", "http://"+r.Host)
	token, err := s.sessions.Create(auth.ClientFingerprint(r))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(auth.HeaderName, token)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("authenticated save: %d %s", w.Code, w.Body)
	}
	if w := networkRequest(s, "GET", "my-goblin.duckdns.org", "", ""); w.Code != 401 {
		t.Fatalf("custom name bypassed auth: %d", w.Code)
	}
	s.cfg.RequireAuth = false
	if err := os.Remove(cfg.Path); err != nil {
		t.Fatal(err)
	}
	if w := networkRequest(s, "POST", "127.0.0.1:8080", "http://127.0.0.1:8080", `{"allowed_hosts":["new.test"]}`); w.Code != 500 {
		t.Fatalf("save failure: %d", w.Code)
	}
	if !s.hostAllowed("my-goblin.duckdns.org") || s.hostAllowed("new.test") {
		t.Fatal("failed save changed active names")
	}
	s.cfg.Path = ""
	if w := networkRequest(s, "POST", "127.0.0.1:8080", "http://127.0.0.1:8080", `{"allowed_hosts":[]}`); w.Code != 409 {
		t.Fatalf("missing config: %d", w.Code)
	}
}

func TestNetworkHostsConcurrentRequests(t *testing.T) {
	s, _ := newTestServer(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				body, _ := json.Marshal(map[string]any{"allowed_hosts": []string{"my-goblin.duckdns.org"}})
				if w := networkRequest(s, "POST", "127.0.0.1:8080", "http://127.0.0.1:8080", string(body)); w.Code != 200 {
					t.Errorf("save: %d", w.Code)
				}
			} else {
				networkRequest(s, "GET", "my-goblin.duckdns.org", "", "")
			}
		}(i)
	}
	wg.Wait()
}
