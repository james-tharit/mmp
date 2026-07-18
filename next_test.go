package main

import "testing"

func TestNextTrackIndex(t *testing.T) {
	// sequential
	if got := nextTrackIndex(0, 3, false); got != 1 {
		t.Fatalf("seq mid: got %d, want 1", got)
	}
	if got := nextTrackIndex(2, 3, false); got != -1 {
		t.Fatalf("seq end: got %d, want -1", got)
	}
	// single / empty playlist never advances
	if got := nextTrackIndex(0, 1, true); got != -1 {
		t.Fatalf("single: got %d, want -1", got)
	}
	// shuffle picks a valid, distinct index
	for i := 0; i < 100; i++ {
		got := nextTrackIndex(1, 4, true)
		if got < 0 || got >= 4 || got == 1 {
			t.Fatalf("shuffle: got %d, want in [0,4) and != 1", got)
		}
	}
}
