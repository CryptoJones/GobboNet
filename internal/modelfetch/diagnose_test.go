package modelfetch

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// Roadmap 1.7.7 #8b. Every failure says where it stopped and what usually
// causes it; a folder that cannot be written is found before any download;
// a file antivirus is still holding is waited for.

func asOS(t *testing.T, name string) {
	t.Helper()
	old := goos
	goos = name
	t.Cleanup(func() { goos = old })
}

func TestFileErrorsAreExplained(t *testing.T) {
	denied := &fs.PathError{Op: "open", Path: "C:\\x", Err: syscall.Errno(winAccessDenied)}
	asOS(t, "windows")
	msg := explainFileErr("write the model", `C:\Users\me\Documents\GobboNet\models`, denied)
	for _, want := range []string{"Windows would not let GobboNet write the model", "Controlled Folder Access",
		"allow", "gobbonet.exe", "Antivirus"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Windows access denied: missing %q in\n%s", want, msg)
		}
	}
	if m := explainFileErr("move the model into place", `C:\m`, syscall.Errno(winSharingViolation)); !strings.Contains(m, "usually antivirus") {
		t.Errorf("sharing violation: %s", m)
	}
	if m := explainFileErr("write the model", `C:\m`, syscall.Errno(winDiskFull)); !strings.Contains(m, "full") {
		t.Errorf("Windows disk full: %s", m)
	}

	asOS(t, "linux")
	msg = explainFileErr("create files", "/srv/models", &fs.PathError{Op: "open", Path: "/srv/models/x", Err: syscall.EACCES})
	if !strings.Contains(msg, "does not have permission") || !strings.Contains(msg, `ls -ld "/srv/models"`) || strings.Contains(msg, "Controlled Folder") {
		t.Errorf("Linux permission: %s", msg)
	}
	if m := explainFileErr("write the model", "/m", syscall.ENOSPC); !strings.Contains(m, "full") {
		t.Errorf("Linux disk full: %s", m)
	}
	if m := explainFileErr("write the model", "/m", syscall.EROFS); !strings.Contains(m, "read-only") {
		t.Errorf("read-only: %s", m)
	}
	// Windows numbers mean other things elsewhere: 32 is EPIPE on Linux.
	if m := explainFileErr("rename files", "/m", syscall.Errno(32)); strings.Contains(m, "antivirus") {
		t.Errorf("a Linux errno was read as a Windows one: %s", m)
	}
}

func TestNetworkErrorsAreExplained(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"DNS":         {&net.DNSError{Err: "no such host", Name: "huggingface.co"}, "Is this PC online"},
		"certificate": {fmt.Errorf("tls: %w", x509.UnknownAuthorityError{}), "HTTPS scanning"},
		"refused":     {&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, "firewall, proxy or antivirus"},
		"timeout":     {&net.OpError{Op: "read", Err: timeoutErr{}}, "did not answer in time"},
	}
	for name, c := range cases {
		if m := explainNetErr("huggingface.co", c.err); !strings.Contains(m, c.want) {
			t.Errorf("%s: %s", name, m)
		}
	}
	for code, want := range map[int]string{403: "accepted its licence", 401: "accepted its licence",
		404: "out of date", 429: "limiting", 503: "Try again later"} {
		if m := explainHTTPStatus(code, fmt.Sprint(code), "x.gguf"); !strings.Contains(m, want) {
			t.Errorf("%d: %s", code, m)
		}
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func failRename(t *testing.T, f func(old, new string) error) {
	t.Helper()
	old := renameFile
	renameFile = f
	t.Cleanup(func() { renameFile = old })
}

func TestCheckFolderLeavesNothingBehind(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "models")
	if err := checkFolder(dir); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("left %d file(s) behind: %v", len(entries), entries)
	}
}

func TestCheckFolderReportsEachStep(t *testing.T) {
	// A "folder" that is really a file: the folder cannot be created.
	file := filepath.Join(t.TempDir(), "not-a-folder")
	os.WriteFile(file, nil, 0o600)
	if err := checkFolder(filepath.Join(file, "models")); err == nil || !strings.Contains(err.Error(), "create the models folder") {
		t.Errorf("unmakeable folder: %v", err)
	}

	// The rename refused, as a permission problem.
	asOS(t, "linux")
	failRename(t, func(_, _ string) error { return &os.LinkError{Op: "rename", Err: syscall.EACCES} })
	dir := t.TempDir()
	err := checkFolder(dir)
	if err == nil || !strings.Contains(err.Error(), "permission to rename files") {
		t.Errorf("refused rename: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a failed check left %v behind", entries)
	}
}

func TestMoveWaitsOutAntivirusOnWindows(t *testing.T) {
	asOS(t, "windows")
	old := moveRetry
	moveRetry = 5 * time.Second
	defer func() { moveRetry = old }()

	var calls atomic.Int32
	failRename(t, func(a, b string) error {
		if calls.Add(1) <= 2 {
			return &os.LinkError{Op: "rename", Old: a, New: b, Err: syscall.Errno(winSharingViolation)}
		}
		return nil
	})
	if err := moveIntoPlace("a.part", "a"); err != nil || calls.Load() != 3 {
		t.Errorf("err=%v after %d tries; want success on the third", err, calls.Load())
	}

	// A real failure is not retried.
	calls.Store(0)
	failRename(t, func(_, _ string) error { calls.Add(1); return syscall.Errno(winDiskFull) })
	if err := moveIntoPlace("a.part", "a"); err == nil || calls.Load() != 1 {
		t.Errorf("disk full retried %d times", calls.Load())
	}

	// And it gives up eventually.
	moveRetry = 300 * time.Millisecond
	failRename(t, func(_, _ string) error { return syscall.Errno(winSharingViolation) })
	start := time.Now()
	if err := moveIntoPlace("a.part", "a"); err == nil || time.Since(start) > 3*time.Second {
		t.Errorf("err=%v after %s", err, time.Since(start))
	}
}

func TestMoveDoesNotRetryElsewhere(t *testing.T) {
	asOS(t, "linux")
	var calls atomic.Int32
	failRename(t, func(_, _ string) error { calls.Add(1); return syscall.EACCES })
	if err := moveIntoPlace("a.part", "a"); err == nil || calls.Load() != 1 {
		t.Errorf("retried %d times on Linux", calls.Load())
	}
}

// A folder that cannot be written is found before a single request is made.
func TestRunStopsAtTheFolderBeforeDownloading(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()
	file := filepath.Join(t.TempDir(), "not-a-folder")
	os.WriteFile(file, nil, 0o600)

	d := New(testEntry(""), filepath.Join(file, "models"))
	d.hostBase = srv.URL
	d.Run()
	st := d.Status()
	if st.State != "error" || st.Stage != StageFolder || hits.Load() != 0 {
		t.Errorf("state=%s stage=%s requests=%d: %s", st.State, st.Stage, hits.Load(), st.Message)
	}
}

func TestRunExplainsARefusal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gated", http.StatusForbidden)
	}))
	defer srv.Close()
	d := New(testEntry(""), t.TempDir())
	d.hostBase = srv.URL
	d.Run()
	st := d.Status()
	if st.State != "error" || st.Stage != StageConnect || !strings.Contains(st.Message, "licence") {
		t.Errorf("stage=%s: %s", st.Stage, st.Message)
	}
}

// A connection that goes quiet used to leave the download "running" forever.
func TestRunGivesUpOnAStall(t *testing.T) {
	old := stallTimeout
	stallTimeout = 300 * time.Millisecond
	defer func() { stallTimeout = old }()
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/raw/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", "1000000")
		w.Write([]byte("the first few bytes"))
		w.(http.Flusher).Flush()
		select { // then nothing
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	dir := t.TempDir()
	d := New(testEntry(""), dir)
	d.hostBase = srv.URL
	done := make(chan struct{})
	go func() { d.Run(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("still running ten seconds after the connection went quiet")
	}
	st := d.Status()
	if st.State != "error" || st.Stage != StageDownload || !strings.Contains(st.Message, "nothing arrived") {
		t.Errorf("stage=%s: %s", st.Stage, st.Message)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.gguf.part")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the partial file was left behind")
	}
}

func TestASuccessfulRunEndsDone(t *testing.T) {
	body := []byte("weights")
	srv := fakeHF(t, body, sum(body))
	dir := t.TempDir()
	d := New(testEntry(""), dir)
	d.hostBase = srv.URL
	d.sizeFloor = 1
	d.Run()
	if st := d.Status(); st.State != "done" || st.Stage != StageDone {
		t.Fatalf("%+v", st)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "model.gguf" {
		t.Errorf("the models folder holds %v; want only the model", entries)
	}
}
