package state

import (
	"bytes"
	"github.com/ElodineOfficial/GobboNet/internal/keyring"
	"os"
	"path/filepath"
	"testing"
)

const oldHistory = `{"threads":[{"id":"old","messages":[{"role":"user","content":"keep me"}]}],"characterCards":[{"id":"c","name":"Goblin"}],"settings":{"theme":"green"},"unknown":{"precise":9007199254740993}}`

func legacyFile(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, ".gobbonet-state.json")
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func readLegacyTest(t *testing.T, p string) []byte {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestLegacyImportAndRestart(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "encrypted"}[encrypted], func(t *testing.T) {
			dir := t.TempDir()
			sp := filepath.Join(dir, "state.json")
			old := legacyFile(t, dir, "\xef\xbb\xbf"+oldHistory)
			var key *keyring.Keyring
			if encrypted {
				key = unlockedKeyring(t)
			}
			result, err := ImportLegacy(sp, []string{dir, dir}, key)
			if err != nil || !result.Imported || len(result.RecoveryProfiles) != 0 {
				t.Fatalf("%+v %v", result, err)
			}
			raw, err := (target{path: sp, key: key}).read()
			if err != nil || string(raw) != oldHistory {
				t.Fatalf("history altered: %s %v", raw, err)
			}
			if string(readLegacyTest(t, old)) != "\xef\xbb\xbf"+oldHistory {
				t.Fatal("original altered")
			}
			if encrypted && !keyring.IsSealed(readLegacyTest(t, sp)) {
				t.Fatal("encrypted install wrote plaintext")
			}
			if _, err := ImportLegacy(sp, []string{dir}, key); err != nil {
				t.Fatal(err)
			}
			// A deliberate deletion after migration must stay deleted, even at restart.
			if err := os.Remove(sp); err != nil {
				t.Fatal(err)
			}
			if _, err := ImportLegacy(sp, []string{dir}, key); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(sp); !os.IsNotExist(err) {
				t.Fatal("deleted history was resurrected")
			}
		})
	}
}
func TestLegacyPreservesCurrentAndProfiles(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "state.json")
	legacyFile(t, dir, oldHistory)
	current := []byte(`{"threads":[],"characterCards":[{"id":"new"}]}`)
	os.WriteFile(sp, current, 0600)
	phone := filepath.Join(dir, "state-phone.json")
	os.WriteFile(phone, []byte(`{"threads":[{"id":"phone"}]}`), 0600)
	result, err := ImportLegacy(sp, []string{dir}, nil)
	if err != nil || result.Imported || len(result.RecoveryProfiles) != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if !bytes.Equal(readLegacyTest(t, sp), current) {
		t.Fatal("current history overwritten")
	}
	recovery, _ := profilePath(sp, result.RecoveryProfiles[0])
	if string(readLegacyTest(t, recovery)) != oldHistory {
		t.Fatal("recovery copy differs")
	}
	if string(readLegacyTest(t, phone)) != `{"threads":[{"id":"phone"}]}` {
		t.Fatal("other profile changed")
	}
}
func TestLegacyMultipleCopiesAndInvalidInput(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	sp := filepath.Join(dir, "state.json")
	legacyFile(t, dir, oldHistory)
	legacyFile(t, other, `{"threads":[],"settings":{"theme":"other"}}`)
	result, err := ImportLegacy(sp, []string{dir, other}, nil)
	if err != nil || result.Imported || len(result.RecoveryProfiles) != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err := os.Stat(sp); !os.IsNotExist(err) {
		t.Fatal("guessed a shared history from conflicting sources")
	}
	for _, body := range []string{"null", "[]", "broken", "{}", `{"threads":"bad"}`} {
		dir := t.TempDir()
		sp := filepath.Join(dir, "state.json")
		p := legacyFile(t, dir, body)
		if _, err := ImportLegacy(sp, []string{dir}, nil); err == nil {
			t.Errorf("accepted %s", body)
		}
		if string(readLegacyTest(t, p)) != body {
			t.Fatal("invalid original changed")
		}
		if _, err := os.Stat(sp); !os.IsNotExist(err) {
			t.Fatal("failed migration published history")
		}
	}
}
func TestLegacyRetriesAfterMarkerFailure(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "state.json")
	legacyFile(t, dir, oldHistory)
	// Simulate a crash after the atomic data commit, before the marker commit.
	os.WriteFile(sp, []byte(oldHistory), 0600)
	result, err := ImportLegacy(sp, []string{dir}, nil)
	if err != nil || !result.Imported || len(result.RecoveryProfiles) != 0 {
		t.Fatalf("retry duplicated history: %+v %v", result, err)
	}
}
func TestLegacyMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "state.json")
	legacyFile(t, dir, `{"characterCards":[{"id":"only-card"}]}`)
	r, err := ImportLegacy(sp, []string{dir}, nil)
	if err != nil || !r.Imported {
		t.Fatalf("%+v %v", r, err)
	}
}
