package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ElodineOfficial/GobboNet/internal/keyring"
)

// LegacyMigration records completion without copying conversation contents into
// the marker. Keep the marker after a purge so an old backup cannot resurrect.
type LegacyMigration struct {
	Imported         bool     `json:"imported"`
	RecoveryProfiles []string `json:"recovery_profiles"`
}

// ImportLegacy discovers only the old server's documented filename in the
// supplied installation directories. Existing current histories are never
// overwritten; distinct old copies become selectable recovery profiles.
// The original files remain untouched. All new history files use the current
// encryption key, if any. Call only after the normal authentication/unlock gate.
func ImportLegacy(statePath string, dirs []string, key *keyring.Keyring) (LegacyMigration, error) {
	mu := lockFor(statePath)
	mu.Lock()
	defer mu.Unlock()
	marker := filepath.Join(filepath.Dir(statePath), "legacy-import.json")
	result := LegacyMigration{RecoveryProfiles: []string{}}
	if raw, err := os.ReadFile(marker); err == nil {
		if err := json.Unmarshal(raw, &result); err != nil {
			return result, fmt.Errorf("legacy history migration record is unreadable: %w", err)
		}
		return result, nil
	} else if !os.IsNotExist(err) {
		return result, err
	}
	type source struct {
		raw  []byte
		hash string
	}
	sources := []source{}
	seen := map[string]bool{}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, ".gobbonet-state.json")
		file, err := os.Open(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return result, fmt.Errorf("could not read older history: %w", err)
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() {
			file.Close()
			return result, fmt.Errorf("older history is not a readable regular file: %s", path)
		}
		raw, err := io.ReadAll(io.LimitReader(file, MaxBodyBytes+1))
		file.Close()
		if err != nil || len(raw) > MaxBodyBytes {
			return result, fmt.Errorf("older history is unreadable or exceeds the history size limit: %s", path)
		}
		raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
		d, err := parseDocument(raw)
		if err != nil || d == nil || d.keys == nil || (!d.hadThreads && d.keys["characterCards"] == nil && d.keys["personaCards"] == nil && d.keys["settings"] == nil) {
			return result, fmt.Errorf("older history has an unsupported format; original preserved at %s", path)
		}
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		if !seen[hash] {
			sources = append(sources, source{raw, hash})
			seen[hash] = true
		}
	}
	if len(sources) == 0 {
		return result, nil
	}
	_, err := os.Stat(statePath)
	absent := os.IsNotExist(err)
	if err != nil && !absent {
		return result, err
	}
	// A previous attempt may have committed the shared copy before the marker
	// write failed. Recognize that exact copy instead of creating a duplicate.
	if !absent && len(sources) == 1 {
		if current, err := (target{path: statePath, key: key}).read(); err == nil && bytes.Equal(current, sources[0].raw) {
			absent = true
		}
	}
	// A single unambiguous old shared file can be adopted automatically. With
	// an existing history (even empty) or multiple copies, retain both explicitly.
	for _, src := range sources {
		dest := statePath
		profile := ""
		if !absent || len(sources) > 1 {
			profile = "legacy-" + src.hash[:16]
			dest, _ = profilePath(statePath, profile)
		}
		// Idempotent retry after a crash between the history write and marker.
		if existing, err := (target{path: dest, key: key}).read(); err == nil {
			if !bytes.Equal(existing, src.raw) {
				return result, fmt.Errorf("legacy recovery destination already contains different history; no files overwritten")
			}
		} else if !os.IsNotExist(err) {
			return result, err
		} else if err := (target{path: dest, key: key}).write(src.raw); err != nil {
			return result, err
		}
		if profile == "" {
			result.Imported = true
		} else {
			result.RecoveryProfiles = append(result.RecoveryProfiles, profile)
		}
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if err := writeAtomic(marker, raw); err != nil {
		return result, fmt.Errorf("older history copied but migration record could not be saved: %w", err)
	}
	return result, nil
}
