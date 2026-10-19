package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyHistoryAvailableThroughSplitRoutes(t *testing.T) {
	s, cfg := newTestServer(t)
	old := []byte(`{"threads":[{"id":"old","messages":[{"role":"user","content":"preserved"}]}],"characterCards":[{"id":"goblin"}]}`)
	if err := os.WriteFile(filepath.Join(filepath.Dir(cfg.Path), ".gobbonet-state.json"), old, 0600); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/state/info", "/state/index", "/state/threads/old", "/state"} {
		w := do(t, s, http.MethodGet, route, nil)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", route, w.Code, w.Body)
		}
	}
	w := do(t, s, http.MethodGet, "/state/migration", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"imported":true`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if got, _ := os.ReadFile(cfg.StatePath()); string(got) != string(old) {
		t.Fatal("legacy snapshot was not preserved")
	}
}
func TestInvalidLegacyPausesWrites(t *testing.T) {
	s, cfg := newTestServer(t)
	os.WriteFile(filepath.Join(cfg.DataDir, ".gobbonet-state.json"), []byte("broken"), 0600)
	w := do(t, s, http.MethodPost, "/state", strings.NewReader(`{"threads":[]}`))
	if w.Code != http.StatusConflict {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(cfg.StatePath()); !os.IsNotExist(err) {
		t.Fatal("empty client overwrote failed migration")
	}
}
