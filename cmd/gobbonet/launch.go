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
