package server

import (
	"os"
	"path/filepath"

	"github.com/ElodineOfficial/GobboNet/internal/keyring"
	"github.com/ElodineOfficial/GobboNet/internal/state"
)

func (s *Server) importLegacyHistory(key *keyring.Keyring) (state.LegacyMigration, error) {
	s.legacyMu.Lock()
	defer s.legacyMu.Unlock()
	if s.legacyDone {
		return s.legacyResult, nil
	}
	dirs := []string{s.cfg.DataDir}
	if s.cfg.Path != "" {
		dirs = append(dirs, filepath.Dir(s.cfg.Path))
	}
	if s.cfg.WebRoot != "" {
		dirs = append(dirs, s.cfg.WebRoot)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	result, err := state.ImportLegacy(s.cfg.StatePath(), dirs, key)
	if err == nil {
		s.legacyDone = true
		s.legacyResult = result
	}
	return result, err
}
