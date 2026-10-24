package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Two waits that were constants until 1.7.8: how long a model may take to
// load (3 minutes) and how long a reply may take to start (5 minutes).
func TestTimeoutSettings(t *testing.T) {
	load := func(t *testing.T, body string) (Config, error) {
		t.Helper()
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return Load(path)
	}

	cfg, err := load(t, "listen_port = 9066\n")
	if err != nil || cfg.ModelLoadTimeout() != 3*time.Minute || cfg.ReplyStartTimeout() != 5*time.Minute {
		t.Fatalf("absent keys must keep the old constants: %v %s %s", err, cfg.ModelLoadTimeout(), cfg.ReplyStartTimeout())
	}
	cfg, err = load(t, "model_load_timeout_seconds = 900\nreply_start_timeout_seconds = 1800\n")
	if err != nil || cfg.ModelLoadTimeout() != 15*time.Minute || cfg.ReplyStartTimeout() != 30*time.Minute {
		t.Fatalf("set values: %v %s %s", err, cfg.ModelLoadTimeout(), cfg.ReplyStartTimeout())
	}
	cfg, err = load(t, "model_load_timeout_seconds = 0\nreply_start_timeout_seconds = -1\n")
	if err != nil || cfg.ModelLoadTimeoutSeconds != 180 || cfg.ReplyStartTimeoutSeconds != 300 {
		t.Fatalf("0 or less is the default, as for the job settings: %v %+v", err, cfg)
	}
	for _, c := range []struct{ body, key string }{
		{"model_load_timeout_seconds = 5\n", "model_load_timeout_seconds must be between 10 and 7200"},
		{"model_load_timeout_seconds = 7201\n", "model_load_timeout_seconds must be between 10 and 7200"},
		{"reply_start_timeout_seconds = 9\n", "reply_start_timeout_seconds must be between 10 and 86400"},
		{"reply_start_timeout_seconds = 86401\n", "reply_start_timeout_seconds must be between 10 and 86400"},
		{"reply_start_timeout_seconds = \"long\"\n", "reply_start_timeout_seconds"},
	} {
		if _, err := load(t, c.body); err == nil || !strings.Contains(err.Error(), c.key) {
			t.Errorf("%q: %v", c.body, err)
		}
	}

	// Settable from the launchers and listed by `gobbonet config keys`.
	keys := strings.Join(Keys(), " ")
	for _, k := range []string{"model_load_timeout_seconds", "reply_start_timeout_seconds"} {
		if !strings.Contains(keys, k) {
			t.Errorf("%s missing from Keys()", k)
		}
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := WriteDefault(path); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, "model_load_timeout_seconds", "600"); err != nil {
		t.Fatal(err)
	}
	if cfg, err := Load(path); err != nil || cfg.ModelLoadTimeoutSeconds != 600 || cfg.ReplyStartTimeoutSeconds != 300 {
		t.Errorf("config set: %v %d %d", err, cfg.ModelLoadTimeoutSeconds, cfg.ReplyStartTimeoutSeconds)
	}
	raw, _ := os.ReadFile(path)
	if strings.Count(string(raw), "model_load_timeout_seconds = ") != 1 || !strings.Contains(string(raw), "model_load_timeout_seconds = 600") {
		t.Error("config set must replace the documented line, not add another")
	}

	// The shipped file documents both, at their defaults.
	for _, want := range []string{"model_load_timeout_seconds = 180", "reply_start_timeout_seconds = 300"} {
		if !strings.Contains(DefaultTOML, want) {
			t.Errorf("DefaultTOML lacks %q", want)
		}
	}
}
