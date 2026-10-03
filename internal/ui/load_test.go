package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTrackReportsErrors(t *testing.T) {
	notFlac := filepath.Join(t.TempDir(), "x.flac")
	if err := os.WriteFile(notFlac, []byte("not a flac file"), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{
		"missing":   "/nonexistent/file.flac",
		"empty":     "",
		"directory": t.TempDir(),
		"not flac":  notFlac,
	} {
		t.Run(name, func(t *testing.T) {
			msg, ok := loadTrack(path)().(trackLoadedMsg)
			if !ok {
				t.Fatal("want trackLoadedMsg")
			}
			if msg.err == nil || msg.audio != nil {
				t.Fatalf("got audio=%v err=%v, want nil audio and an error", msg.audio, msg.err)
			}
		})
	}
}
