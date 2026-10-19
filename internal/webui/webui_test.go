package webui

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// The unpacked web/ folder is refreshed when, and only when, it no longer
// matches this build -- by version OR by the page built into it (roadmap 1.7.7
// #13). These drive the core with two different pages under one version, which
// the real embed cannot do inside a single test binary.

func page(chat string) fstest.MapFS {
	return fstest.MapFS{
		"chat.html":       {Data: []byte(chat)},
		"js/01-config.js": {Data: []byte("const GOBBONET_UI_VERSION = '1.7.7';\n")},
		".gitkeep":        {Data: []byte{}},
	}
}

func fp(t *testing.T, fsys fstest.MapFS) string {
	t.Helper()
	f, err := Fingerprint(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFingerprintFollowsTheContent(t *testing.T) {
	a, a2, b := fp(t, page("A")), fp(t, page("A")), fp(t, page("B"))
	if a != a2 {
		t.Error("the same page gave two fingerprints")
	}
	if a == b {
		t.Error("two different pages gave one fingerprint")
	}
	if len(a) != 64 {
		t.Errorf("fingerprint %q is not a SHA-256 hex digest", a)
	}

	// .gitkeep is never exported, so it must not count.
	noKeep := page("A")
	delete(noKeep, ".gitkeep")
	if fp(t, noKeep) != a {
		t.Error(".gitkeep changed the fingerprint")
	}

	// A file moving is a different page even if no bytes changed.
	moved := page("A")
	moved["js/02-config.js"] = moved["js/01-config.js"]
	delete(moved, "js/01-config.js")
	if fp(t, moved) == a {
		t.Error("renaming a file did not change the fingerprint")
	}
}

func TestSameVersionDifferentPageIsRewritten(t *testing.T) {
	dir := t.TempDir()
	const v = "1.7.7-go-test"
	if _, err := exportFrom(dir, v, page("first build"), fp(t, page("first build"))); err != nil {
		t.Fatal(err)
	}
	second := page("second build")
	if !needsExport(dir, v, fp(t, second)) {
		t.Fatal("same version, different page: not rewritten -- the 1.7.7 #13 bug")
	}
	if _, err := exportFrom(dir, v, second, fp(t, second)); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "chat.html"))
	if string(got) != "second build" {
		t.Fatalf("chat.html = %q after the rewrite, want the second build's page", got)
	}
	if needsExport(dir, v, fp(t, second)) {
		t.Error("still asks for a rewrite after writing the second build's page")
	}
}

func TestHandEditsSurviveRestartsOfTheSameBuild(t *testing.T) {
	dir := t.TempDir()
	const v = "1.7.7-go-test"
	p := page("as built")
	if _, err := exportFrom(dir, v, p, fp(t, p)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chat.html"), []byte("edited by hand"), 0o644); err != nil {
		t.Fatal(err)
	}
	if needsExport(dir, v, fp(t, p)) {
		t.Fatal("a restart of the same build would replace hand edits -- the check must " +
			"compare the binary's page, never the files on disk")
	}
}

func TestAStampFromBefore177IsRewrittenOnce(t *testing.T) {
	dir := t.TempDir()
	const v = "1.7.7-go-test"
	if err := os.WriteFile(filepath.Join(dir, "chat.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The old format: a version line and no content line.
	if err := os.WriteFile(filepath.Join(dir, stampName), []byte("version="+v+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := page("new")
	if !needsExport(dir, v, fp(t, p)) {
		t.Fatal("a stamp with no content line was trusted")
	}
	if _, err := exportFrom(dir, v, p, fp(t, p)); err != nil {
		t.Fatal(err)
	}
	if needsExport(dir, v, fp(t, p)) {
		t.Error("rewritten once, but still asks again -- it would rewrite on every start")
	}
}

func TestVersionChangeAndDamageStillRewrite(t *testing.T) {
	dir := t.TempDir()
	p := page("x")
	if _, err := exportFrom(dir, "1.7.7-go-a", p, fp(t, p)); err != nil {
		t.Fatal(err)
	}
	if !needsExport(dir, "1.7.7-go-b", fp(t, p)) {
		t.Error("a different version did not ask for a rewrite")
	}
	os.Remove(filepath.Join(dir, "chat.html"))
	if !needsExport(dir, "1.7.7-go-a", fp(t, p)) {
		t.Error("a folder that lost chat.html was trusted")
	}
	os.Remove(filepath.Join(dir, stampName)) // an export interrupted before its last step
	if !needsExport(dir, "1.7.7-go-a", fp(t, p)) {
		t.Error("a folder with no stamp was trusted")
	}
	if !needsExport(dir, "1.7.7-go-a", "") {
		t.Error("an unreadable built-in page (empty fingerprint) must never match")
	}
}

func TestTheStampRecordsBoth(t *testing.T) {
	dir := t.TempDir()
	p := page("x")
	want := fp(t, p)
	if _, err := exportFrom(dir, "1.7.7-go-a", p, want); err != nil {
		t.Fatal(err)
	}
	if got := ExportedVersion(dir); got != "1.7.7-go-a" {
		t.Errorf("ExportedVersion = %q", got)
	}
	if got := ExportedFingerprint(dir); got != want {
		t.Errorf("ExportedFingerprint = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitkeep")); err == nil {
		t.Error(".gitkeep was written into the user's folder")
	}
}
