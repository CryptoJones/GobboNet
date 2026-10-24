package server

// Issue #63: the home-screen manifest and icons are served before sign-in, and
// nothing else is.

import (
	"bytes"
	"encoding/json"
	"image/png"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/auth"
	"github.com/ElodineOfficial/GobboNet/internal/webui"
)

func signedOutServer(t *testing.T) (*Server, string) {
	t.Helper()
	srv, cfg := newTestServer(t)
	secret, err := auth.NewSecret("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	srv.cfg.RequireAuth = true
	srv.secret = secret
	for name, body := range map[string]string{
		"manifest.webmanifest":   `{"name":"GobboNet"}`,
		"icons/gobbonet-180.png": "png",
		"icons/gobbonet-192.png": "png",
		"icons/gobbonet-256.png": "png",
		"icons/someone-else.png": "not on the list",
		"manifest.json":          `{"not":"this one"}`,
	} {
		p := filepath.Join(cfg.WebRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return srv, cfg.WebRoot
}

func TestHomeScreenFilesAreServedSignedOut(t *testing.T) {
	srv, _ := signedOutServer(t)
	for path := range publicAssets {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			rec := do(t, srv, method, path, nil)
			if rec.Code != http.StatusOK {
				t.Errorf("%s %s signed out: got %d, want 200", method, path, rec.Code)
			}
		}
	}
	rec := do(t, srv, http.MethodGet, "/manifest.webmanifest", nil)
	if ct := rec.Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Errorf("manifest Content-Type = %q", ct)
	}
	if ct := do(t, srv, http.MethodGet, "/icons/gobbonet-192.png", nil).Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("icon Content-Type = %q", ct)
	}
}

// Named one by one: a file that merely sits beside them is still behind the
// password, as is anything that is not a plain read.
func TestNothingElseIsPublic(t *testing.T) {
	srv, _ := signedOutServer(t)
	for _, path := range []string{
		"/icons/someone-else.png", "/manifest.json", "/chat.html", "/", "/icons/",
		"/icons/../chat.html", "/icons/gobbonet-180.png/", "/ICONS/gobbonet-180.png",
	} {
		if rec := do(t, srv, http.MethodGet, path, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s signed out: got %d, want 401", path, rec.Code)
		}
	}
	if rec := do(t, srv, http.MethodPost, "/manifest.webmanifest", strings.NewReader("{}")); rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /manifest.webmanifest signed out: got %d, want 401", rec.Code)
	}
}

// The list, the page and the shipped files agree. Skipped in a checkout that
// has not run stage-web.sh, where there is no built-in page to check.
func TestTheBuiltInPageNamesOnlyPublicFiles(t *testing.T) {
	if !webui.Staged() {
		t.Skip("stage-web.sh has not been run")
	}
	built := webui.Built()
	raw, err := fs.ReadFile(built, "manifest.webmanifest")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("http")) {
		t.Error("the manifest names a URL; everything in it must be this machine's")
	}
	var m struct {
		Display  string `json:"display"`
		StartURL string `json:"start_url"`
		Scope    string `json:"scope"`
		Icons    []struct {
			Src, Sizes, Type string
		} `json:"icons"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifest is not JSON: %v", err)
	}
	if m.Display != "standalone" || m.StartURL != "./" || m.Scope != "./" {
		t.Errorf("manifest: %+v", m)
	}
	// Every file the page and the manifest name, with the size it claims.
	want := map[string]string{"/manifest.webmanifest": ""}
	for _, ic := range m.Icons {
		want["/"+ic.Src] = ic.Sizes
		if ic.Type != "image/png" {
			t.Errorf("%s: type %q", ic.Src, ic.Type)
		}
	}
	chat, err := fs.ReadFile(built, "chat.html")
	if err != nil {
		t.Fatal(err)
	}
	head := string(chat[:bytes.Index(chat, []byte("</head>"))])
	link := func(rel string) string {
		match := regexp.MustCompile(`rel="` + rel + `" href="([^"]+)"`).FindStringSubmatch(head)
		if match == nil {
			t.Fatalf("chat.html has no %s link", rel)
		}
		return "/" + match[1]
	}
	if got := link("manifest"); got != "/manifest.webmanifest" {
		t.Errorf("chat.html's manifest is %s", got)
	}
	want[link("apple-touch-icon")] = "180x180"

	for _, href := range regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(auth.HomeScreenTags, -1) {
		if _, ok := want[href[1]]; !ok {
			t.Errorf("the login page names %s, which chat.html and the manifest do not", href[1])
		}
	}
	for path := range publicAssets {
		if _, ok := want[path]; !ok {
			t.Errorf("%s is public but nothing names it", path)
		}
	}
	for path, sizes := range want {
		if !publicAssets[path] {
			t.Errorf("%s is named by the page but is not public", path)
			continue
		}
		b, err := fs.ReadFile(built, strings.TrimPrefix(path, "/"))
		if err != nil {
			t.Errorf("%s is not in the built-in page: %v", path, err)
			continue
		}
		if sizes == "" {
			continue
		}
		img, err := png.DecodeConfig(bytes.NewReader(b))
		if err != nil {
			t.Errorf("%s is not a PNG: %v", path, err)
			continue
		}
		if got := strconv.Itoa(img.Width) + "x" + strconv.Itoa(img.Height); got != sizes {
			t.Errorf("%s is %s, named as %s", path, got, sizes)
		}
	}
}
