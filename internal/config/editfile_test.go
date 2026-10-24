package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const editSecret = "$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$aGFzaGhhc2hoYXNo"
const editKey = "sk-upstream-key-1234"

const editFile = `# A comment that has to survive.
llm_url = "http://127.0.0.1:11437"
llm_api_key = "` + editKey + `"
listen_host = "127.0.0.1"
listen_port = 9066   # the port

# --- Access control ---
access_secret = "` + editSecret + `"
session_ttl_hours = 12

[ui]
stream_replies = true
`

func writeEditFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestReadForEditHidesSecrets(t *testing.T) {
	path := writeEditFile(t, editFile)
	v, err := ReadForEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(v.Text, editSecret) || strings.Contains(v.Text, editKey) {
		t.Fatalf("a secret reached the editor:\n%s", v.Text)
	}
	for _, want := range []string{
		`access_secret = "(hidden)"  # the password hash: kept as it is. Change the password with: gobbonet set-password`,
		`llm_api_key = "(hidden)"`,
		"# A comment that has to survive.", "listen_port = 9066   # the port", "[ui]",
	} {
		if !strings.Contains(v.Text, want) {
			t.Errorf("missing %q in:\n%s", want, v.Text)
		}
	}
	if strings.Join(v.Hidden, ",") != "llm_api_key,access_secret" || v.HasBackup || v.Version != fileVersion([]byte(editFile)) {
		t.Errorf("view: %+v", v)
	}

	// An empty secret has nothing to hide, and is shown as it is.
	v2, _ := ReadForEdit(writeEditFile(t, strings.Replace(editFile, editKey, "", 1)))
	if !strings.Contains(v2.Text, `llm_api_key = ""`) || strings.Join(v2.Hidden, ",") != "access_secret" {
		t.Errorf("empty key: %v\n%s", v2.Hidden, v2.Text)
	}
	// A secret written so it cannot be hidden line by line: no editor at all.
	if _, err := ReadForEdit(writeEditFile(t, "access_secret = \"\"\"\nabc\"\"\"\nlisten_port = 9066\n")); err == nil {
		t.Error("a multi-line secret should refuse to open")
	}
}

func save(t *testing.T, path, text string) (EditResult, error) {
	t.Helper()
	v, err := ReadForEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		text = v.Text
	}
	res, err := PrepareEdit(path, text, v.Version)
	if err != nil {
		return res, err
	}
	return CommitEdit(path, v.Version, res)
}

func TestEditRoundTripKeepsSecretsCommentsAndABackup(t *testing.T) {
	path := writeEditFile(t, editFile)
	v, _ := ReadForEdit(path)
	edited := strings.Replace(v.Text, "session_ttl_hours = 12", "session_ttl_hours = 6", 1)
	edited = strings.Replace(edited, "stream_replies = true", "stream_replies = false", 1)
	res, err := PrepareEdit(path, edited, v.Version)
	if err != nil {
		t.Fatal(err)
	}
	if mustRead(t, path) != editFile {
		t.Fatal("PrepareEdit wrote to the file")
	}
	res, err = CommitEdit(path, v.Version, res)
	if err != nil {
		t.Fatal(err)
	}
	got := mustRead(t, path)
	for _, want := range []string{`access_secret = "` + editSecret + `"`, `llm_api_key = "` + editKey + `"`,
		"session_ttl_hours = 6", "stream_replies = false", "# A comment that has to survive.", "listen_port = 9066   # the port"} {
		if !strings.Contains(got, want) {
			t.Errorf("saved file lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, HiddenValue) {
		t.Error("the placeholder was written to disk")
	}
	if strings.Join(res.Changed, ",") != "session_ttl_hours,ui.stream_replies" {
		t.Errorf("changed: %v", res.Changed)
	}
	if mustRead(t, BackupPath(path)) != editFile {
		t.Error("the backup is not the file as it was")
	}
	if runtime.GOOS != "windows" {
		for _, p := range []string{path, BackupPath(path)} {
			if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
				t.Errorf("%s is %v, want 0600: it holds the password hash", p, st.Mode().Perm())
			}
		}
	}
	if cfg, err := Load(path); err != nil || cfg.SessionTTLHours != 6 || cfg.AccessSecret != editSecret || cfg.LLMAPIKey != editKey {
		t.Errorf("reloaded: %v %+v", err, cfg)
	}
	if res.Version != fileVersion([]byte(got)) {
		t.Error("the new version is not the file's")
	}
}

func TestEditSecretsRules(t *testing.T) {
	path := writeEditFile(t, editFile)
	v, _ := ReadForEdit(path)
	cases := []struct {
		name, from, to, wantErr string
	}{
		{"changing the password hash", `access_secret = "(hidden)"`, `access_secret = "$argon2id$other"`, "cannot be changed here"},
		{"removing the password line", `access_secret = "(hidden)"`, `# gone`, "keep the access_secret line"},
		{"moving it into [ui]", `access_secret = "(hidden)"`, `# moved`, "keep the access_secret line"},
	}
	for _, c := range cases {
		text := strings.Replace(v.Text, v.Text[strings.Index(v.Text, c.from):strings.Index(v.Text, c.from)+len(c.from)+len(hiddenNote["access_secret"])], c.to, 1)
		if c.name == "moving it into [ui]" {
			text += "access_secret = \"(hidden)\"\n"
		}
		if _, err := PrepareEdit(path, text, v.Version); err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if mustRead(t, path) != editFile {
		t.Fatal("a refused save changed the file")
	}

	// A new upstream key may be written in place of (hidden); it is never shown.
	text := strings.Replace(v.Text, `llm_api_key = "(hidden)"`, `llm_api_key = "sk-new"`, 1)
	res, err := save(t, path, text)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mustRead(t, path), `llm_api_key = "sk-new"`) || strings.Join(res.Changed, ",") != "llm_api_key" {
		t.Errorf("new key: %v\n%s", res.Changed, mustRead(t, path))
	}
	if v2, _ := ReadForEdit(path); strings.Contains(v2.Text, "sk-new") {
		t.Error("the new key is shown back")
	}

	// (hidden) where the file has nothing to keep.
	empty := writeEditFile(t, "listen_port = 9066\n")
	ve, _ := ReadForEdit(empty)
	if _, err := PrepareEdit(empty, ve.Text+"llm_api_key = \"(hidden)\"\n", ve.Version); err == nil {
		t.Error("(hidden) with nothing behind it should be refused")
	}
}

func TestEditRefusesWhatWouldStopTheServer(t *testing.T) {
	path := writeEditFile(t, editFile)
	v, _ := ReadForEdit(path)
	for name, c := range map[string][2]string{
		"bad TOML":               {"session_ttl_hours = 12", "session_ttl_hours = = 12"},
		"wrong type":             {"listen_port = 9066", `listen_port = "nine"`},
		"misspelt key":           {"session_ttl_hours = 12", "sesion_ttl_hours = 12"},
		"port out of range":      {"listen_port = 9066", "listen_port = 70000"},
		"missing server_exe":     {"session_ttl_hours = 12", `server_exe = "/no/such/llama-server"`},
		"missing key file":       {"session_ttl_hours = 12", `llm_api_key_file = "/no/such/key"`},
		"gpu_reserve_mib":        {"session_ttl_hours = 12", "gpu_reserve_mib = -5"},
		"duplicate key":          {"session_ttl_hours = 12", "session_ttl_hours = 12\nsession_ttl_hours = 3"},
		"load timeout too short": {"session_ttl_hours = 12", "model_load_timeout_seconds = 3"},
	} {
		if _, err := PrepareEdit(path, strings.Replace(v.Text, c[0], c[1], 1), v.Version); err == nil {
			t.Errorf("%s: accepted", name)
		} else if name == "misspelt key" && !strings.Contains(err.Error(), "did you mean session_ttl_hours") {
			t.Errorf("misspelt key: %v", err)
		} else if name == "bad TOML" && !strings.Contains(err.Error(), "line") {
			t.Errorf("bad TOML should say which line: %v", err)
		}
	}
	if _, err := PrepareEdit(path, strings.Repeat("#", MaxEditBytes+1), v.Version); err == nil {
		t.Error("oversized text accepted")
	}
	if mustRead(t, path) != editFile {
		t.Fatal("a refused save changed the file")
	}

	// A key from a newer version is let through, and reported.
	res, err := save(t, path, v.Text+"\n")
	if err != nil || len(res.Changed) != 0 {
		t.Fatalf("unchanged save: %v %v", err, res.Changed)
	}
	v, _ = ReadForEdit(path)
	res, err = save(t, path, strings.Replace(v.Text, "session_ttl_hours = 12", "session_ttl_hours = 12\nsome_future_setting = 3", 1))
	if err != nil || strings.Join(res.Warnings, ",") != "some_future_setting" {
		t.Errorf("future key: %v %v", err, res.Warnings)
	}
}

func TestEditConflicts(t *testing.T) {
	path := writeEditFile(t, editFile)
	v, _ := ReadForEdit(path)
	// Another panel saves in between (or a sign-in upgrades the hash).
	if err := Set(path, "session_ttl_hours", "8"); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareEdit(path, v.Text, v.Version); !errors.Is(err, ErrEditConflict) {
		t.Errorf("stale version: %v", err)
	}
	// Changed between prepare and commit.
	v, _ = ReadForEdit(path)
	res, err := PrepareEdit(path, v.Text, v.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := Set(path, "session_ttl_hours", "9"); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitEdit(path, v.Version, res); !errors.Is(err, ErrEditConflict) {
		t.Errorf("commit over a newer file: %v", err)
	}
	if !strings.Contains(mustRead(t, path), "session_ttl_hours = 9") {
		t.Error("the newer file was overwritten")
	}
}

func TestRestoreBackupKeepsTodaysPassword(t *testing.T) {
	path := writeEditFile(t, editFile)
	v, _ := ReadForEdit(path)
	if _, err := save(t, path, strings.Replace(v.Text, "session_ttl_hours = 12", "session_ttl_hours = 2", 1)); err != nil {
		t.Fatal(err)
	}
	// The password changes after the save; going back must not undo that.
	if err := Set(path, "access_secret", "$argon2id$v=19$new"); err != nil {
		t.Fatal(err)
	}
	v, _ = ReadForEdit(path)
	res, err := PrepareRestore(path, v.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CommitEdit(path, v.Version, res); err != nil {
		t.Fatal(err)
	}
	got := mustRead(t, path)
	if !strings.Contains(got, "session_ttl_hours = 12") || !strings.Contains(got, `access_secret = "$argon2id$v=19$new"`) || strings.Contains(got, editSecret) {
		t.Errorf("restored:\n%s", got)
	}
	// And the version it replaced is now the one to go back to.
	if !strings.Contains(mustRead(t, BackupPath(path)), "session_ttl_hours = 2") {
		t.Error("restore should keep what it replaced as the backup")
	}
	if _, err := PrepareRestore(writeEditFile(t, editFile), fileVersion([]byte(editFile))); err == nil {
		t.Error("restore with no backup should say so")
	}
}

func TestDefaultConfigOpensAndSavesUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := WriteDefault(path); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, "access_secret", editSecret); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, path)
	res, err := save(t, path, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changed) != 0 || mustRead(t, path) != before {
		t.Errorf("an untouched save changed the shipped config: %v", res.Changed)
	}
}
