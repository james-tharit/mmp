package main

import (
	"testing"

	"github.com/gopxl/beep/v2"
)

// MockStreamSeekCloser correctly implements github.com/gopxl/beep/v2's StreamSeekCloser interface
type MockStreamSeekCloser struct {
	ErrToReturn error
	LenToReturn int
}

func (m *MockStreamSeekCloser) Stream(samples [][2]float64) (n int, ok bool) {
	// Return a sample to signal that data exists
	return len(samples), true
}

func (m *MockStreamSeekCloser) Err() error {
	return m.ErrToReturn
}

func (m *MockStreamSeekCloser) Len() int {
	return m.LenToReturn
}

func (m *MockStreamSeekCloser) Position() int {
	return 0
}

func (m *MockStreamSeekCloser) Seek(p int) error {
	return nil
}

func (m *MockStreamSeekCloser) Close() error {
	return nil
}

func TestNewAudioPanel(t *testing.T) {
	speakerSR = beep.SampleRate(44100)

	t.Run("Success Path - Pipeline components chain correctly", func(t *testing.T) {
		// Arrange
		sampleRate := beep.SampleRate(44100)
		mockStreamer := &MockStreamSeekCloser{LenToReturn: 1000}

		// Act
		panel, err := NewAudioPanel(sampleRate, mockStreamer)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if panel == nil {
			t.Fatal("expected panel to not be nil")
		}

		// Verify internal assignments
		if panel.sampleRate != sampleRate {
			t.Errorf("expected sampleRate %d, got %d", sampleRate, panel.sampleRate)
		}

		if panel.streamer != mockStreamer {
			t.Error("expected streamer to match the passed mockStreamer")
		}

		if panel.ctrl == nil {
			t.Error("expected ctrl to be initialized, got nil")
		}

		if panel.resampler == nil {
			t.Error("expected speed resampler to be initialized, got nil")
		}

		if panel.volume == nil {
			t.Error("expected volume to be initialized, got nil")
		}
	})
}
