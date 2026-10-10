// Package state persists playback state to ~/.config/mmp/state.json so the
// next launch resumes where the user left off. Missing or corrupt files
// return a zero State; writes fail silently (resume is a nicety, not critical).
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	TrackPath string        `json:"track_path"`
	Position  time.Duration `json:"position"`
	Volume    float64       `json:"volume"`
	Shuffle   int           `json:"shuffle"`
	Repeat    int           `json:"repeat"`
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mmp", "state.json"), nil
}

// Load returns the saved state, or a zero State on first launch / any error.
func Load() State {
	var s State
	p, err := path()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// Save writes state to the config file; the caller may ignore errors.
func Save(s State) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
