package main

import (
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
	loopStreamer, err := beep.Loop2(streamer)
	if err != nil {
		return nil, err
	}

	ctrl := &beep.Ctrl{Streamer: loopStreamer}

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
