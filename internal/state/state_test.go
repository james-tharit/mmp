package state

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir) // macOS falls back to $HOME/Library

	want := State{TrackPath: "/x/y.flac", Position: 5 * time.Second, Volume: -1.5, Shuffle: 2, Repeat: 1}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	if got := Load(); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	if got := Load(); got != (State{}) {
		t.Fatalf("expected zero state, got %+v", got)
	}
}
