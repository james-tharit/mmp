package main

import (
	"math/rand/v2"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

// audio controller
type audioPanel struct {
	sampleRate beep.SampleRate
	streamer   beep.StreamSeekCloser
	ctrl       *beep.Ctrl
	resampler  *beep.Resampler
	volume     *effects.Volume
}

func (ap *audioPanel) Play() {
	speaker.Play(ap.volume)
}

// SetSpeed updates the playback speed dynamically
func (ap *audioPanel) SetSpeed(ratio float64) {
	ap.resampler.SetRatio(ratio)
}

// constructor for audioPanel
func NewAudioPanel(sampleRate beep.SampleRate, streamer beep.StreamSeekCloser) (*audioPanel, error) {
	// Play once (no loop) so the track drains and the UI can auto-advance.
	ctrl := &beep.Ctrl{Streamer: streamer}

	// 1. This handles user-directed speed adjustments (the Z and X hotkeys)
	speedResampler := beep.ResampleRatio(4, 1.0, ctrl)

	// 2. This seamlessly maps the song's native sample rate to your global hardware rate
	speakerResampler := beep.Resample(4, sampleRate, speakerSR, speedResampler)

	// 3. Wrap everything up in your volume modifier
	volume := &effects.Volume{Streamer: speakerResampler, Base: 2}

	return &audioPanel{
		sampleRate: sampleRate,
		streamer:   streamer,
		ctrl:       ctrl,
		resampler:  speedResampler, // Assign speed layer here so your Z/X keybinds still map safely
		volume:     volume,
	}, nil
}

// TrackLoadedMsg is sent when a new track finishes loading
type TrackLoadedMsg struct {
	audio    *audioPanel
	metadata *FLACMetadata
	err      error
}

// loadTrackCmd asynchronously loads a new track so the UI doesn't freeze
func loadTrackCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(filePath)
		if err != nil {
			return TrackLoadedMsg{err: err}
		}
		// Note: We don't defer f.Close() here because the streamer needs it open.
		// Your audioPanel or streamer.Close() will handle closing it eventually.
		streamer, format, err := flac.Decode(f)
		if err != nil {
			return TrackLoadedMsg{err: err}
		}

		ap, err := NewAudioPanel(format.SampleRate, streamer)
		if err != nil {
			return TrackLoadedMsg{err: err}
		}

		metadata, err := getMetadata(filePath)
		if err != nil {
			return TrackLoadedMsg{err: err}
		}

		return TrackLoadedMsg{
			audio:    ap,
			metadata: metadata,
		}
	}
}

// nextTrackIndex returns the index to play next: a random distinct track when
// shuffle is on, otherwise cur+1. Returns -1 when there's nowhere to go
// (single track, or end of a sequential playlist).
func nextTrackIndex(cur, n int, shuffle bool) int {
	if n <= 1 {
		return -1
	}
	if shuffle {
		for {
			if i := rand.IntN(n); i != cur {
				return i
			}
		}
	}
	if cur+1 < n {
		return cur + 1
	}
	return -1
}
