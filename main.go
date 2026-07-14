package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

// hardware sample rate
var speakerSR beep.SampleRate

func main() {
	tui := flag.Bool("tui", false, "run the terminal UI")
	gui := flag.Bool("gui", false, "run the graphical UI (default)")
	flag.Parse()

	targetPath := flag.Arg(0)
	if targetPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [-tui|-gui] <path_to_file_or_directory>\n", os.Args[0])
		os.Exit(1)
	}

	// 1. Check if the path is a file or a directory
	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		report(fmt.Errorf("error accessing path: %w", err))
	}

	var flacFiles []string

	if fileInfo.IsDir() {
		// Scenario A: It's a directory, scan it and all sub-directories for FLAC files
		err := filepath.WalkDir(targetPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && isFlac(d.Name()) {
				flacFiles = append(flacFiles, path)
			}
			return nil
		})
		if err != nil {
			report(fmt.Errorf("failed to read directory: %w", err))
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

	// Get metadata for all songs in the playlist
	playlistMetadata, err := getAllMetadata(flacFiles)
	if err != nil {
		report(err)
	}

	// 4. Launch the chosen UI. GUI is the default; -tui opts into the terminal UI.
	if *tui && !*gui {
		RunTUI(flacFiles, ap, metadata, playlistMetadata)
	} else {
		RunGUI(flacFiles, ap, metadata, playlistMetadata)
	}
}

func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
