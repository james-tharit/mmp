package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

// hardware sample rate
var speakerSR beep.SampleRate

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path_to_file_or_directory>\n", os.Args[0])
		os.Exit(1)
	}

	targetPath := os.Args[1]

	// 1. Check if the path is a file or a directory
	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		report(fmt.Errorf("error accessing path: %w", err))
	}

	var flacFiles []string

	if fileInfo.IsDir() {
		// Scenario A: It's a directory, scan for FLAC files
		files, err := os.ReadDir(targetPath)
		if err != nil {
			report(fmt.Errorf("failed to read directory: %w", err))
		}

		for _, file := range files {
			if !file.IsDir() && isFlac(file.Name()) {
				flacFiles = append(flacFiles, filepath.Join(targetPath, file.Name()))
			}
		}
	} else {
		// Scenario B: It's a single file, verify it's a FLAC
		if isFlac(targetPath) {
			flacFiles = append(flacFiles, targetPath)
		} else {
			fmt.Fprintln(os.Stderr, "Error: The provided file is not a FLAC file.")
			os.Exit(1)
		}
	}

	// 2. Ensure we have at least one file to play
	if len(flacFiles) == 0 {
		fmt.Fprintln(os.Stderr, "No FLAC files found to play.")
		os.Exit(1)
	}

	// 3. Setup and play the first track initially
	firstTrack := flacFiles[0]

	f, err := os.Open(firstTrack)
	if err != nil {
		report(err)
	}
	// Note: these stay open for the first track until main exits,
	// which is perfectly fine since the UI handles closing later tracks.
	defer f.Close()

	streamer, format, err := flac.Decode(f)
	if err != nil {
		report(err)
	}
	defer streamer.Close()

	// Save the baseline hardware sample rate globally
	speakerSR = format.SampleRate
	// Initialize speaker using the first track's sample rate as the hardware baseline
	speaker.Init(speakerSR, speakerSR.N(time.Millisecond*100))
	defer speaker.Close()

	ap, err := NewAudioPanel(format.SampleRate, streamer)
	if err != nil {
		report(err)
	}

	ap.Play()

	metadata, err := getMetadata(firstTrack)
	if err != nil {
		report(err)
	}

	// 4. Pass the whole PLAYLIST slice to BubbleTea instead of just the first track string
	model := NewUIModel(flacFiles, ap, metadata)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		report(err)
	}
}

func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
