package modelfetch

// Roadmap 1.7.7 #8b: why in-app model downloads failed.
//
// The maintainer's account is that the failures were "mostly a permissions
// issue", and that after a lot of attempts the feature was held off. No error
// text from those attempts survives. So this file does two jobs:
//
//   1. Every failure names the step it happened at and says, in plain words,
//      what usually causes that error on that platform -- so the next report
//      arrives with its cause attached instead of "it didn't work".
//   2. The two permission-shaped failures that can be headed off are: a folder
//      that cannot be written is found in a second, before gigabytes are
//      fetched (checkFolder), and a just-written file that antivirus is still
//      holding is waited for rather than reported as a failure (moveIntoPlace).
//
// Each step is also logged with a "[download]" prefix, which is what to copy
// out of GobboNet's window when reporting a failure.

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// Stages, in order. Status.Stage carries the one running or the one that
// failed, so a report says where it stopped without reading any code.
const (
	StageFolder   = "folder"   // can the models folder be written at all
	StageChecksum = "checksum" // reading the published hash
	StageConnect  = "connect"  // reaching Hugging Face and getting an answer
	StageDownload = "download" // the bytes arriving and being written
	StageVerify   = "verify"   // checksum and size checks
	StageMove     = "move"     // renaming the .part into place
	StageDone     = "done"
)

func logf(format string, args ...any) { log.Printf("[download] "+format, args...) }

// Windows error numbers that syscall does not name portably.
const (
	winAccessDenied     = 5
	winWriteProtect     = 19
	winSharingViolation = 32
	winLockViolation    = 33
	winHandleDiskFull   = 39
	winDiskFull         = 112
)

func errnoOf(err error) (uintptr, bool) {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return uintptr(errno), true
	}
	return 0, false
}

func isWin(err error, codes ...uintptr) bool {
	n, ok := errnoOf(err)
	if !ok || goos != "windows" {
		return false
	}
	for _, c := range codes {
		if n == c {
			return true
		}
	}
	return false
}

// goos is runtime.GOOS, a variable so the wording for each platform can be
// tested from any platform.
var goos = runtime.GOOS

// explainFileErr turns a failed file operation in dir into a sentence a user
// can act on. op is what was being attempted, as a verb phrase: "create files",
// "write the model", "move the model into place".
func explainFileErr(op, dir string, err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission) || isWin(err, winAccessDenied):
		if goos == "windows" {
			return fmt.Sprintf("Windows would not let GobboNet %s in %s (%v).\n\n"+
				"If this folder is inside Documents, Desktop, Pictures or a OneDrive folder, "+
				"Windows Security's Controlled Folder Access may be blocking it: open Windows "+
				"Security > Virus & threat protection > Ransomware protection, and allow "+
				"gobbonet.exe — or move GobboNet to a folder outside those. Antivirus software "+
				"can block it the same way.", op, dir, err)
		}
		return fmt.Sprintf("GobboNet does not have permission to %s in %s (%v).\n\n"+
			"Check who owns that folder (ls -ld \"%s\"). It must be writable by the user "+
			"GobboNet runs as.", op, dir, err, dir)
	case errors.Is(err, syscall.ENOSPC) || isWin(err, winDiskFull, winHandleDiskFull):
		return fmt.Sprintf("The disk holding %s is full (%v). Free some space and try again.", dir, err)
	case errors.Is(err, syscall.EROFS) || isWin(err, winWriteProtect):
		return fmt.Sprintf("%s is on a read-only drive (%v), so GobboNet cannot %s there.", dir, err, op)
	case isWin(err, winSharingViolation, winLockViolation):
		return fmt.Sprintf("Another program is holding the file in %s, so GobboNet could not %s (%v).\n\n"+
			"This is usually antivirus scanning the new file. Wait a minute and try again; "+
			"if it keeps happening, allow gobbonet.exe or the models folder in the antivirus.", dir, op, err)
	}
	return fmt.Sprintf("Could not %s in %s: %v", op, dir, err)
}

// explainNetErr says what a failed request to host most likely means.
func explainNetErr(host string, err error) string {
	var dnsErr *net.DNSError
	var certErr *tls.CertificateVerificationError
	var unknownCA x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	var netErr net.Error
	switch {
	case errors.As(err, &dnsErr):
		return fmt.Sprintf("Could not look up %s (%v). Is this PC online?", host, err)
	case errors.As(err, &certErr) || errors.As(err, &unknownCA) || errors.As(err, &hostnameErr):
		return fmt.Sprintf("The connection to %s failed its security check (%v).\n\n"+
			"Antivirus \"HTTPS scanning\" or a company proxy that inspects traffic usually "+
			"causes this. Turning that off for GobboNet, or trying another network, will tell.", host, err)
	case errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET):
		return fmt.Sprintf("The connection to %s was refused or cut off (%v). A firewall, proxy or "+
			"antivirus may be blocking GobboNet from the internet.", host, err)
	case errors.As(err, &netErr) && netErr.Timeout():
		return fmt.Sprintf("%s did not answer in time (%v). The connection may be slow, or something "+
			"between this PC and the internet is holding the request.", host, err)
	}
	return fmt.Sprintf("Could not reach %s: %v", host, err)
}

// explainHTTPStatus says what a refusal from the model host means.
func explainHTTPStatus(code int, status, file string) string {
	switch {
	case code == 401 || code == 403:
		return fmt.Sprintf("Hugging Face refused %s (%s). The model may need a signed-in account "+
			"that has accepted its licence, which GobboNet does not have — this catalogue "+
			"entry needs to point at an open copy.", file, status)
	case code == 404:
		return fmt.Sprintf("Hugging Face has no file called %s (%s): the catalogue entry is out of "+
			"date.", file, status)
	case code == 429:
		return fmt.Sprintf("Hugging Face is limiting downloads from this connection (%s). Wait a "+
			"while and try again.", status)
	case code >= 500:
		return fmt.Sprintf("Hugging Face had a problem (%s). Try again later.", status)
	}
	return fmt.Sprintf("Download failed: the server answered %s.", status)
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	return "the model host"
}

// checkFolder does, with a few bytes, every file operation the download will
// do with gigabytes: create the folder, create a file, write, close, rename,
// delete. A folder that refuses any of them is reported now, in the second it
// takes, rather than after a 20 GB download has nowhere to go.
func checkFolder(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return errors.New(explainFileErr("create the models folder", dir, err))
	}
	var b [6]byte
	_, _ = rand.Read(b[:])
	probe := filepath.Join(dir, ".gobbonet-write-test-"+hex.EncodeToString(b[:]))
	f, err := os.OpenFile(probe+".part", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.New(explainFileErr("create files", dir, err))
	}
	_, werr := f.Write([]byte("GobboNet checks it can write here before downloading.\n"))
	cerr := f.Close()
	if werr != nil || cerr != nil {
		os.Remove(probe + ".part")
		return errors.New(explainFileErr("write files", dir, errors.Join(werr, cerr)))
	}
	if err := moveIntoPlace(probe+".part", probe); err != nil {
		os.Remove(probe + ".part")
		return errors.New(explainFileErr("rename files", dir, err))
	}
	if err := os.Remove(probe); err != nil {
		// Written and renamed fine; a leftover test file is not worth failing
		// over, but it is worth knowing about.
		logf("could not remove the write test %s: %v", probe, err)
	}
	return nil
}

// renameFile is os.Rename, a variable so tests can make it fail on cue.
var renameFile = os.Rename

// moveRetry is how long moveIntoPlace keeps trying.
var moveRetry = 20 * time.Second

// moveIntoPlace renames part to final, waiting out the transient locks Windows
// reports when another program has the file open.
//
// The usual one is antivirus. It opens a newly written file to scan it the
// moment the writer closes it, and while it does, renaming the file fails with
// a sharing violation -- or, for some products, "Access is denied". That is the
// last step of a download that has otherwise succeeded, and failing it throws
// away gigabytes over a lock that clears in seconds. So those errors are
// retried for a while. Elsewhere a rename failure is not transient, and is
// returned at once.
func moveIntoPlace(part, final string) error {
	deadline := time.Now().Add(moveRetry)
	wait := 100 * time.Millisecond
	for attempt := 1; ; attempt++ {
		err := renameFile(part, final)
		if err == nil {
			if attempt > 1 {
				logf("moved %s into place on attempt %d", filepath.Base(final), attempt)
			}
			return nil
		}
		transient := isWin(err, winSharingViolation, winLockViolation, winAccessDenied)
		if !transient || time.Now().After(deadline) {
			return err
		}
		if attempt == 1 {
			logf("%s is held by another program (%v); retrying for up to %s", filepath.Base(part), err, moveRetry)
		}
		time.Sleep(wait)
		if wait < 2*time.Second {
			wait *= 2
		}
	}
}
