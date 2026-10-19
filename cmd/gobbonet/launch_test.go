package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/config"
)

func TestLaunchSetupDecision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if first, err := launchNeedsSetup(path, false); !first || err != nil {
		t.Fatalf("fresh: %v, %v", first, err)
	}
	if first, err := launchNeedsSetup(path, true); first || err == nil {
		t.Fatalf("explicit missing config must fail: %v, %v", first, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("launch decision must not create config")
	}
	if err := config.WriteDefault(path); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if first, err := launchNeedsSetup(path, false); first || err != nil {
		t.Fatalf("existing remote config must not trigger setup: %v, %v", first, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("existing config was changed")
	}
	if err := os.WriteFile(path, []byte("invalid = ["), 0600); err != nil {
		t.Fatal(err)
	}
	if first, err := launchNeedsSetup(path, false); first || err == nil {
		t.Fatalf("corrupt config must fail without setup: %v, %v", first, err)
	}
}

func TestLaunchHelp(t *testing.T) {
	if err := cmdLaunch([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
}
