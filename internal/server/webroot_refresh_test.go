package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/version"
	"github.com/ElodineOfficial/GobboNet/internal/webui"
)

// Replacing gobbonet(.exe) must be enough to update the interface. The web
// folder beside the executable used to be stamped with the release number
// only, so a new build of the same release (every 1.7.5 build) kept serving
// the interface an earlier build had written.
func TestANewBuildOfTheSameReleaseRefreshesTheInterface(t *testing.T) {
	requireStaged(t)
	old := version.Version
	version.Version = "1.7.5-go-test-new"
	t.Cleanup(func() { version.Version = old })

	exe := t.TempDir()
	dir := filepath.Join(exe, "web")
	if err := os.MkdirAll(filepath.Join(dir, "js"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Written by an earlier build of the same release, stamped the old way.
	if err := os.WriteFile(filepath.Join(dir, ".gobbonet-ui"), []byte("version=1.7.5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chat.html"), []byte("interface from the previous build"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := builtInSource(exe)
	served, err := fs.ReadFile(src.FS, "chat.html")
	if err != nil {
		t.Fatal(err)
	}
	built, err := fs.ReadFile(webui.Built(), "chat.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(served) != string(built) {
		t.Fatal("the previous build's interface is still being served")
	}
	if got := webui.ExportedVersion(dir); got != version.Version {
		t.Errorf("folder stamped %q, want this build %q", got, version.Version)
	}
	if !strings.Contains(strings.Join(src.Notes, " "), "refreshed") {
		t.Errorf("the refresh was not reported: %q", src.Notes)
	}
}

// Restarting the same build must not touch the folder: that is where someone
// reads and edits the interface, and edits last until an update.
func TestTheSameBuildKeepsTheInterfaceItWrote(t *testing.T) {
	requireStaged(t)
	old := version.Version
	version.Version = "1.7.5-go-test-same"
	t.Cleanup(func() { version.Version = old })

	exe := t.TempDir()
	builtInSource(exe)
	page := filepath.Join(exe, "web", "chat.html")
	if err := os.WriteFile(page, []byte("edited by hand"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := builtInSource(exe)
	served, err := fs.ReadFile(src.FS, "chat.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(served) != "edited by hand" {
		t.Fatal("a restart of the same build replaced the interface")
	}
}

// A build re-cut under an UNCHANGED version must still refresh the folder when
// its interface differs (roadmap 1.7.7 #13). Before, the stamp held only the
// version, so the second build served the first build's page -- and the ABOUT
// version check could not notice, because the two version strings agreed.
func TestASameVersionBuildWithADifferentInterfaceRefreshes(t *testing.T) {
	requireStaged(t)
	old := version.Version
	version.Version = "1.7.7-go-test-recut"
	t.Cleanup(func() { version.Version = old })

	exe := t.TempDir()
	dir := filepath.Join(exe, "web")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Written by an earlier build with this very version but another page.
	stamp := "version=" + version.Version + "\ncontent=0000000000000000000000000000000000000000000000000000000000000000\n"
	if err := os.WriteFile(filepath.Join(dir, ".gobbonet-ui"), []byte(stamp), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chat.html"), []byte("interface from the earlier build"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := builtInSource(exe)
	served, err := fs.ReadFile(src.FS, "chat.html")
	if err != nil {
		t.Fatal(err)
	}
	built, err := fs.ReadFile(webui.Built(), "chat.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(served) != string(built) {
		t.Fatal("same version, different interface: the earlier build's page is still being served")
	}
	if got := webui.ExportedFingerprint(dir); got != webui.BuiltFingerprint() {
		t.Errorf("folder fingerprint %q, want this build's %q", got, webui.BuiltFingerprint())
	}
	notes := strings.Join(src.Notes, " ")
	if !strings.Contains(notes, "carries a different interface") {
		t.Errorf("the refresh was not explained as a same-version change: %q", src.Notes)
	}
}
