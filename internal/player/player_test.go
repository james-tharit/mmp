package player

import (
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
)

type fakeStreamer struct{ pos, length int }

func (f *fakeStreamer) Stream(s [][2]float64) (int, bool) { return len(s), true }
func (f *fakeStreamer) Err() error                        { return nil }
func (f *fakeStreamer) Len() int                          { return f.length }
func (f *fakeStreamer) Position() int                     { return f.pos }
func (f *fakeStreamer) Seek(p int) error                  { f.pos = p; return nil }
func (f *fakeStreamer) Close() error                      { return nil }

func newPanel(s *fakeStreamer) *Panel {
	speakerRate = 44100
	return New(beep.SampleRate(44100), s)
}

func TestSeekClamps(t *testing.T) {
	s := &fakeStreamer{pos: 100, length: 1000}
	p := newPanel(s)

	p.SeekBy(-time.Minute)
	if s.pos != 0 {
		t.Fatalf("seek back: pos %d, want 0", s.pos)
	}
	p.SeekBy(time.Minute)
	if s.pos != 999 {
		t.Fatalf("seek forward: pos %d, want 999", s.pos)
	}
}

func TestVolumeAndSpeedClamp(t *testing.T) {
	p := newPanel(&fakeStreamer{length: 1000})

	p.AdjustVolume(1)
	if v := p.State().Volume; v != maxVolume {
		t.Fatalf("volume %v, want %v", v, maxVolume)
	}
	p.AdjustVolume(-100)
	if v := p.State().Volume; v != minVolume {
		t.Fatalf("volume %v, want %v", v, minVolume)
	}
	for i := 0; i < 100; i++ {
		p.ScaleSpeed(16.0 / 15)
	}
	if v := p.State().Speed; v != maxSpeed {
		t.Fatalf("speed %v, want %v", v, maxSpeed)
	}
	for i := 0; i < 200; i++ {
		p.ScaleSpeed(15.0 / 16)
	}
	if v := p.State().Speed; v != minSpeed {
		t.Fatalf("speed %v, want %v", v, minSpeed)
	}
}

func TestPauseAndEnded(t *testing.T) {
	s := &fakeStreamer{length: 10}
	p := newPanel(s)

	p.TogglePause()
	if !p.State().Paused {
		t.Fatal("want paused")
	}
	if p.Ended() {
		t.Fatal("pos 0 of 10 must not be ended")
	}
	s.pos = 10
	if !p.Ended() {
		t.Fatal("pos == len must be ended")
	}
}

func TestLoadErrors(t *testing.T) {
	for _, path := range []string{"", "/nonexistent/file.flac", t.TempDir()} {
		if p, err := Load(path); err == nil || p != nil {
			t.Errorf("Load(%q) = %v, %v; want nil, error", path, p, err)
		}
	}
}
