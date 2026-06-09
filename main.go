package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

type audioPanel struct {
	sampleRate beep.SampleRate
	streamer   beep.StreamSeeker
	ctrl       *beep.Ctrl
	resampler  *beep.Resampler
	volume     *effects.Volume
}

func newAudioPanel(sampleRate beep.SampleRate, streamer beep.StreamSeeker) (*audioPanel, error) {
	loopStreamer, err := beep.Loop2(streamer)
	if err != nil {
		return nil, err
	}

	ctrl := &beep.Ctrl{Streamer: loopStreamer}
	resampler := beep.ResampleRatio(4, 1, ctrl)
	volume := &effects.Volume{Streamer: resampler, Base: 2}
	return &audioPanel{sampleRate, streamer, ctrl, resampler, volume}, nil
}

func (ap *audioPanel) play() {
	speaker.Play(ap.volume)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s song.flac\n", os.Args[0])
		os.Exit(1)
	}

	// Open and decode the audio file
	f, err := os.Open(os.Args[1])
	if err != nil {
		report(err)
	}
	defer f.Close()

	streamer, format, err := flac.Decode(f)
	if err != nil {
		report(err)
	}
	defer streamer.Close()

	// Initialize speaker
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/30))
	defer speaker.Close()

	// Create audio panel with the beep engine
	ap, err := newAudioPanel(format.SampleRate, streamer)
	if err != nil {
		report(err)
	}

	ap.play()

	// Create and run BubbleTea app
	model := NewUIModel(os.Args[1], ap)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		report(err)
	}
}

func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
