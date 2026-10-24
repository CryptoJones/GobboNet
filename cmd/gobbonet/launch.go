package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ElodineOfficial/GobboNet/internal/config"
)

// launch is the desktop entry point. Serving an existing configuration must
// never re-run the batch questionnaire or rewrite the user's settings.
func cmdLaunch(argv []string) error {
	fs := flag.NewFlagSet("gobbonet launch", flag.ContinueOnError)
	configPath := stringFlag(fs, "config", "path to config.toml")
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("launch accepts only --config")
	}
	path, explicit := config.Discover(*configPath)
	first, err := launchNeedsSetup(path, explicit)
	if err != nil {
		return err
	}
	if !first && !explicit && runtime.GOOS == "windows" {
		if exe, err := os.Executable(); err == nil {
			first = launchSetupUnfinished(path, filepath.Dir(exe))
		}
	}
	if first {
		if runtime.GOOS != "windows" {
			return errors.New("run gobbonet setup before launching")
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "launch.bat")); err != nil {
			return fmt.Errorf("first-run setup is missing: reinstall the full GobboNet package: %w", err)
		}
		// A fixed relative command avoids cmd.exe quoting an installation path.
		cmd := exec.Command("cmd.exe", "/d", "/c", "launch.bat")
		cmd.Dir = dir
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		// Normal launch always uses Go. The explicit batch entry remains available
		// to users who deliberately choose the legacy server.
		for _, env := range os.Environ() {
			key, _, _ := strings.Cut(env, "=")
			if !strings.EqualFold(key, "GOBBONET_LEGACY_SERVER") {
				cmd.Env = append(cmd.Env, env)
			}
		}
		return cmd.Run()
	}
	return cmdServe([]string{"--config", path, "--open"})
}

// setupPendingName is the mark launch.bat leaves beside itself from the moment
// it hands the password to gobbonet.exe -- early, so that a first run which
// stops does not ask for a second password (#63) -- until it reaches :launch.
// That early write creates config.toml, and launchNeedsSetup serves any
// config.toml it finds.
const setupPendingName = ".gobbonet-setup-pending"

// launchSetupUnfinished reports whether launch.bat started setup in dir and
// stopped before the end, leaving a config that names no engine: what a first
// run closed during its download or model menu leaves behind. Serving that
// would open a chat with nothing behind it and never offer setup again, so
// launch runs setup instead.
//
// Both conditions, not the mark alone: "Setup or add a model" runs launch.bat
// on a working install too, and stopping that leaves the mark beside a config
// whose engine is set. That install carries on serving.
func launchSetupUnfinished(path, dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, setupPendingName)); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "launch.bat")); err != nil {
		return false
	}
	cfg, err := config.Load(path)
	if err != nil {
		return false
	}
	return cfg.ServerExe == ""
}

func launchNeedsSetup(path string, explicit bool) (bool, error) {
	_, err := config.Load(path)
	if errors.Is(err, config.ErrNotFound) && !explicit {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("cannot launch with configuration %s: %w", path, err)
	}
	return false, nil
}
