package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ElodineOfficial/GobboNet/internal/config"
)

// Roadmap 1.7.7 #7 -- choosing the address phones connect to.

// aLANAddress is a real, non-loopback address on this machine, or skips.
func aLANAddress(t *testing.T) string {
	t.Helper()
	for _, a := range lanChoices() {
		if ip := net.ParseIP(a.IP); ip != nil && ip.To4() != nil {
			return a.IP
		}
	}
	t.Skip("this machine has no non-loopback IPv4 address to pin")
	return ""
}

// panelPort is a free port inside the range the panel accepts. freePort's
// come from the kernel's ephemeral range, which on Linux sits above it.
func panelPort(t *testing.T) int {
	t.Helper()
	for p := 21000 + os.Getpid()%5000; p < 32767; p++ {
		if ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", p)); err == nil {
			ln.Close()
			return p
		}
	}
	t.Skip("no free port in 1024-32767")
	return 0
}

func get(t *testing.T, url string) int {
	t.Helper()
	c := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode
}

// Pinning an address narrows what the NETWORK can reach, never what this PC
// can: 127.0.0.1 is what the launchers open and probe, and a single listener
// on the pinned address would have stopped it answering.
func TestPinnedAddressAlsoServesLoopback(t *testing.T) {
	ip := aLANAddress(t)
	port := freePort(t)
	s := bindServer(t, ip, port)
	b, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	s.SetBind(b)
	done := make(chan error, 1)
	go func() { done <- s.Serve(b.Listener) }()

	if b.Loopback != "127.0.0.1" {
		t.Errorf("Loopback = %q, want 127.0.0.1 beside the pinned %s", b.Loopback, ip)
	}
	if !b.LANReachable() {
		t.Error("a pinned LAN address must still report LAN access")
	}
	if got, want := b.LocalURL(), fmt.Sprintf("http://127.0.0.1:%d/", port); got != want {
		t.Errorf("LocalURL = %q, want %q", got, want)
	}
	for _, h := range []string{ip, "127.0.0.1"} {
		if code := get(t, fmt.Sprintf("http://%s:%d/health-fileserver", h, port)); code != http.StatusOK {
			t.Errorf("%s answered %d, want 200", h, code)
		}
	}

	// Close stops both sockets, and Serve returns as it does for one.
	b.Listener.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return after the joined listener was closed")
	}
	for _, h := range []string{ip, "127.0.0.1"} {
		if c, err := net.DialTimeout("tcp", net.JoinHostPort(h, fmt.Sprint(port)), time.Second); err == nil {
			c.Close()
			t.Errorf("%s:%d still accepts after Close", h, port)
		}
	}
}

// Wildcards already include loopback, and loopback is loopback: neither gets a
// companion, and neither changes shape.
func TestOnlyAPinnedAddressGetsACompanion(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "127.0.0.1"} {
		s := bindServer(t, host, freePort(t))
		b, err := s.Listen()
		if err != nil {
			t.Fatalf("%s: Listen: %v", host, err)
		}
		if b.Loopback != "" {
			t.Errorf("%s got a loopback companion %q", host, b.Loopback)
		}
		if _, joined := b.Listener.(*joinedListener); joined {
			t.Errorf("%s: listener was wrapped", host)
		}
		b.Listener.Close()
	}
}

// Something else on 127.0.0.1:port must not stop the pinned address the user
// asked for; it is reported, and the PC is told to use the pinned address.
func TestCompanionFailureIsNotFatal(t *testing.T) {
	ip := aLANAddress(t)
	port := freePort(t)
	squat, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)))
	if err != nil {
		t.Fatal(err)
	}
	defer squat.Close()

	b, err := bindServer(t, ip, port).Listen()
	if err != nil {
		t.Fatalf("Listen failed outright: %v", err)
	}
	defer b.Listener.Close()
	if b.Loopback != "" || b.LoopbackErr == nil {
		t.Errorf("Loopback=%q LoopbackErr=%v; want none and an error", b.Loopback, b.LoopbackErr)
	}
	if got, want := b.LocalURL(), fmt.Sprintf("http://%s:%d/", ip, port); got != want {
		t.Errorf("LocalURL = %q, want the pinned %q", got, want)
	}
}

func TestPinnedHost(t *testing.T) {
	for host, want := range map[string]bool{
		"192.168.1.20": true, "10.0.0.4": true, "fd00::5": true,
		"0.0.0.0": false, "::": false, "": false, "127.0.0.1": false, "::1": false, "localhost": false,
	} {
		if got := pinnedHost(host); got != want {
			t.Errorf("pinnedHost(%q) = %v, want %v", host, got, want)
		}
	}
}

// --- /lan --------------------------------------------------------------------

func lanServer(t *testing.T) (*Server, config.Config) {
	t.Helper()
	srv, cfg := newTestServer(t)
	srv.SetBind(&Bind{Host: "0.0.0.0", Port: 9066})
	return srv, cfg
}

func TestLANGetContract(t *testing.T) {
	srv, _ := lanServer(t)
	rec := do(t, srv, http.MethodGet, "/lan", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /lan: %d %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	for _, k := range []string{"os", "port_min", "port_max", "running", "addresses", "choices", "saved"} {
		if _, ok := body[k]; !ok {
			t.Errorf("GET /lan is missing %q: %s", k, rec.Body.String())
		}
	}
	// Windows: launch.bat's 1024-32767. Elsewhere: the Linux wizard's 1024-65535.
	wantMax := 65535
	if runtime.GOOS == "windows" {
		wantMax = 32767
	}
	if int(body["port_min"].(float64)) != 1024 || int(body["port_max"].(float64)) != wantMax {
		t.Errorf("port range %v-%v, want 1024-%d", body["port_min"], body["port_max"], wantMax)
	}
	saved := body["saved"].(map[string]any)
	// The test config file names neither key, so the defaults are what is saved.
	if saved["host"] != "0.0.0.0" || int(saved["port"].(float64)) != 9066 {
		t.Errorf("saved = %v, want the defaults", saved)
	}
	running := body["running"].(map[string]any)
	if running["local_url"] != "http://127.0.0.1:9066/" || running["lan_access"] != true {
		t.Errorf("running = %v", running)
	}
}

func postLAN(t *testing.T, srv *Server, body, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := newReq(http.MethodPost, "/lan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

// A save lands in the config file, for the next start -- and the running
// server reports the file's values as saved while still running the old ones.
func TestLANSaveWritesTheConfigFile(t *testing.T) {
	srv, cfg := lanServer(t)
	port := panelPort(t)
	rec := postLAN(t, srv, fmt.Sprintf(`{"host":"127.0.0.1","port":%d}`, port), "http://127.0.0.1:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /lan: %d %s", rec.Code, rec.Body.String())
	}
	saved, err := config.ReadListen(cfg.Path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Host != "127.0.0.1" || saved.Port != port {
		t.Errorf("config file now says %+v, want 127.0.0.1:%d", saved, port)
	}
	// And Load agrees, so the next start really does use it.
	loaded, err := config.Load(cfg.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ListenHost != "127.0.0.1" || loaded.ListenPort != port {
		t.Errorf("Load reads %s:%d", loaded.ListenHost, loaded.ListenPort)
	}

	body := decode(t, do(t, srv, http.MethodGet, "/lan", nil))
	if s := body["saved"].(map[string]any); s["host"] != "127.0.0.1" {
		t.Errorf("GET after save: saved = %v", s)
	}
	if r := body["running"].(map[string]any); r["host"] != "0.0.0.0" {
		t.Errorf("GET after save: running changed to %v; it applies on the next start", r)
	}
}

func TestLANSaveRefusesWhatWouldNotStart(t *testing.T) {
	srv, cfg := lanServer(t)
	before, _ := os.ReadFile(cfg.Path)

	busyPort := panelPort(t)
	busy, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", busyPort))
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	cases := []struct {
		name, body string
		code       int
		says       string
	}{
		{"an address this PC does not have", `{"host":"192.0.2.99","port":9070}`, 400, "not an address this PC has"},
		{"a name, not an address", `{"host":"example.com","port":9070}`, 400, "not an address this PC has"},
		{"a reserved port", `{"host":"0.0.0.0","port":80}`, 400, "The port must be between 1024 and"},
		{"a port past the top", `{"host":"0.0.0.0","port":70000}`, 400, "The port must be between 1024 and"},
		{"the engine's port", `{"host":"0.0.0.0","port":11437}`, 400, "belongs to the AI engine"},
		{"the embedding server's port", `{"host":"0.0.0.0","port":11436}`, 400, "belongs to the AI engine"},
		{"a port in use", fmt.Sprintf(`{"host":"0.0.0.0","port":%d}`, busyPort), 409, "being used by another program"},
		{"half a request", `{"host":"0.0.0.0"}`, 400, "both host and port"},
	}
	for _, c := range cases {
		rec := postLAN(t, srv, c.body, "")
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.says) {
			t.Errorf("%s: %d %s; want %d mentioning %q", c.name, rec.Code, rec.Body.String(), c.code, c.says)
		}
	}
	if after, _ := os.ReadFile(cfg.Path); string(after) != string(before) {
		t.Error("a refused save changed the config file")
	}
}

// The port the server is on now is ours; checking it would always fail.
func TestLANSaveKeepsTheRunningPortWithoutProbing(t *testing.T) {
	srv, _ := lanServer(t)
	old := probePort
	probePort = func(string, int) error { return errors.New("probe must not run for the running port") }
	defer func() { probePort = old }()
	if rec := postLAN(t, srv, `{"host":"127.0.0.1","port":9066}`, ""); rec.Code != http.StatusOK {
		t.Errorf("changing only the address: %d %s", rec.Code, rec.Body.String())
	}
}

// The route that can widen who reaches the server cannot be driven by another
// site's page, password or not.
func TestLANSaveRefusesOtherOrigins(t *testing.T) {
	srv, cfg := lanServer(t)
	before, _ := os.ReadFile(cfg.Path)
	for _, origin := range []string{"http://evil.example", "null", "http://127.0.0.1:9999"} {
		if rec := postLAN(t, srv, `{"host":"0.0.0.0","port":9066}`, origin); rec.Code != http.StatusForbidden {
			t.Errorf("Origin %s: %d, want 403", origin, rec.Code)
		}
	}
	if after, _ := os.ReadFile(cfg.Path); string(after) != string(before) {
		t.Error("a cross-origin request changed the config file")
	}
}

func TestLANReportsEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOBBONET_LISTEN_PORT", "9999")
	srv, _ := lanServer(t)
	body := decode(t, do(t, srv, http.MethodGet, "/lan", nil))
	env, _ := body["env_overrides"].([]any)
	if len(env) != 1 || env[0] != "GOBBONET_LISTEN_PORT" {
		t.Errorf("env_overrides = %v", body["env_overrides"])
	}
}

func TestLANChoicesAreAcceptedAddresses(t *testing.T) {
	for _, a := range lanChoices() {
		if !validLANHost(a.IP) {
			t.Errorf("offered %s but would refuse it", a.IP)
		}
	}
	for _, h := range []string{"0.0.0.0", "::", "127.0.0.1", "::1"} {
		if !validLANHost(h) {
			t.Errorf("refuses %s", h)
		}
	}
}
