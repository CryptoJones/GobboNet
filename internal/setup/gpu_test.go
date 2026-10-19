package setup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Roadmap 1.7.7 #8a: /api/gpu, which the wizard compares min_vram against.

func gpuEngine(t *testing.T, listing string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script stand-in for the engine")
	}
	p := filepath.Join(t.TempDir(), "llama-server")
	script := "#!/bin/sh\n[ \"$1\" = \"--list-devices\" ] || exit 9\nprintf '%s' '" + listing + "'\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func getGPU(t *testing.T, s *server) gpuResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/gpu", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/gpu: %d %s", rec.Code, rec.Body.String())
	}
	var g gpuResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGPUReportsTheDedicatedCard(t *testing.T) {
	s := newTestServer(t)
	s.opts.ServerExe = gpuEngine(t, "Available devices:\n  CUDA0: NVIDIA GeForce RTX 4060 (8188 MiB, 7900 MiB free)\n")
	g := getGPU(t, s)
	if !g.Known || g.VRAMGB != 8 || g.Device != "NVIDIA GeForce RTX 4060" || len(g.Devices) != 1 {
		t.Errorf("got %+v", g)
	}
}

// Every case where there is no honest figure answers known=false, which the
// wizard reads as "ask nothing". A failed probe never blocks a download.
func TestGPUUnknownCasesAskNothing(t *testing.T) {
	cases := map[string]string{
		"no GPU":          "Available devices:\n  (none)\n",
		"integrated only": "Available devices:\n  Vulkan0: AMD Radeon 780M Graphics (16384 MiB, 15000 MiB free)\n",
		"not an engine":   "usage: something else\n",
	}
	for name, listing := range cases {
		s := newTestServer(t)
		s.opts.ServerExe = gpuEngine(t, listing)
		g := getGPU(t, s)
		if g.Known || g.VRAMGB != 0 || g.Note == "" {
			t.Errorf("%s: got %+v; want unknown, with a note saying why", name, g)
		}
	}
	s := newTestServer(t)
	s.opts.ServerExe = "" // a remote-only install has no engine to ask
	if g := getGPU(t, s); g.Known {
		t.Errorf("no engine: %+v", g)
	}
}

// The Python shell in front of the Linux wizard proxies only the paths it
// names, so the new one has to be among them -- in both copies.
func TestTheLinuxShellProxiesAPIGPU(t *testing.T) {
	for _, p := range []string{"installer-linux/gobbonet-setup.py", "linux-amd64/gobbonet-setup.py"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", p))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "elif self.path in ('/api/download', '/api/gpu'):") {
			t.Errorf("%s does not proxy GET /api/gpu", p)
		}
	}
}
