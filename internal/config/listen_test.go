package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadListenReadsTheFileOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	write := func(body string) {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write("llm_url = \"http://x:1\"\n[ui]\nauto_scroll = \"off\"\n")
	got, err := ReadListen(path)
	if err != nil || got != (SavedListen{DefaultListenHost, DefaultListenPort}) {
		t.Errorf("absent keys: %+v %v, want the defaults", got, err)
	}

	write("listen_host = \"192.168.1.20\"\nlisten_port = 9070\n")
	t.Setenv("GOBBONET_LISTEN_PORT", "9999") // ignored: the file is what is saved
	got, err = ReadListen(path)
	if err != nil || got != (SavedListen{"192.168.1.20", 9070}) {
		t.Errorf("got %+v %v", got, err)
	}
}

func TestListenEnvOverrides(t *testing.T) {
	t.Setenv("GOBBONET_LISTEN_HOST", "")
	t.Setenv("GOBBONET_LISTEN_PORT", "")
	t.Setenv("GEMMA_LISTEN_HOST", "")
	t.Setenv("GEMMA_LISTEN_PORT", "")
	if got := ListenEnvOverrides(); len(got) != 0 {
		t.Errorf("nothing set: %v", got)
	}
	t.Setenv("GEMMA_LISTEN_PORT", "9070")
	t.Setenv("GOBBONET_LISTEN_HOST", "127.0.0.1")
	got := ListenEnvOverrides()
	if len(got) != 2 || got[0] != "GOBBONET_LISTEN_HOST" || got[1] != "GEMMA_LISTEN_PORT" {
		t.Errorf("got %v", got)
	}
}

// The panel's range is the Windows launchers' range; a port outside it would
// be silently replaced with 9066 by launch.bat on the next start.
func TestPanelPortRangeMatchesTheLaunchers(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, f := range []string{"launch.bat", "setup-lan.bat"} {
		raw, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		s := string(raw)
		if !strings.Contains(s, "lss 1024") || !strings.Contains(s, "gtr 32767") {
			t.Errorf("%s no longer bounds the port at 1024-32767; update MinPanelPort/MaxPanelPort with it", f)
		}
	}
	if MinPanelPort != 1024 || MaxPanelPortFor("windows") != 32767 {
		t.Errorf("Windows panel range %d-%d", MinPanelPort, MaxPanelPortFor("windows"))
	}
}

// Elsewhere the Linux setup wizard's range applies, which reaches 65535.
func TestPanelPortRangeMatchesTheLinuxWizard(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "installer-linux", "gobbonet-setup.py"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "1024 <= port <= 65535") {
		t.Error("the Linux wizard no longer allows 1024-65535; update MaxPanelPortElsewhere with it")
	}
	if MaxPanelPortFor("linux") != 65535 || MaxPanelPortFor("darwin") != 65535 {
		t.Errorf("elsewhere: %d", MaxPanelPortFor("linux"))
	}
}

func TestEnginePorts(t *testing.T) {
	c := Config{LLMURL: "http://127.0.0.1:8081", EmbedURL: "http://192.168.1.9:9999"}
	got := c.EnginePorts()
	for _, p := range []int{11436, 11437, 8081} {
		if !got[p] {
			t.Errorf("%d missing from %v", p, got)
		}
	}
	if got[9999] {
		t.Error("a port on another machine is not this machine's engine port")
	}
}
