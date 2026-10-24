//go:build !windows

package supervisor

import (
	"strings"
	"testing"
	"time"
)

// The load timeout is the configured one (model_load_timeout_seconds), is
// reported in /swap-status for the page's poll, and names itself when it runs
// out -- it used to say "within 3m0s" whatever the deadline was.
func TestLoadTimeoutIsConfiguredReportedAndNamed(t *testing.T) {
	s := lifecycleSupervisor(t)
	if got := s.Status().LoadTimeoutSeconds; got != 3 {
		t.Errorf("/swap-status load_timeout_seconds = %d, want 3 (Options.LoadTimeout)", got)
	}
	s.opts.LoadTimeout = 0
	if got := s.Status().LoadTimeoutSeconds; got != 180 {
		t.Errorf("unset: %d, want the 180 s default", got)
	}

	s.opts.LoadTimeout = 300 * time.Millisecond
	t.Setenv("GOBBONET_LIFECYCLE_STALL", "1")
	err := s.Boot("model.gguf")
	if err == nil {
		t.Fatal("expected a load timeout")
	}
	msg := err.Error() + " | " + s.Status().Message
	for _, want := range []string{"within 300ms", "model_load_timeout_seconds", "CONFIG > Config File"} {
		if !strings.Contains(msg, want) {
			t.Errorf("timeout message lacks %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "3m0s") {
		t.Errorf("message names the old constant: %s", msg)
	}
}
