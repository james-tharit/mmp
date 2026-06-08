package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 1. Point to your music file (M4A or FLAC)
	trackPath := "./わたがし.m4a"

	if _, err := os.Stat(trackPath); os.IsNotExist(err) {
		log.Fatalf("Error: Test file '%s' not found. Put an m4a/flac file here and name it sample.m4a", trackPath)
	}

	// 2. Initialize VLC player
	vlcPlayer, err := NewVLCPlayer()
	if err != nil {
		log.Fatal("VLC initialization failed:", err)
	}
	defer vlcPlayer.Release()

	// 3. Load track and extract metadata
	meta, err := vlcPlayer.LoadTrack(trackPath)
	if err != nil {
		log.Fatal("Failed to load track:", err)
	}

	// 4. Start playback
	if err := vlcPlayer.Play(); err != nil {
		log.Fatal("Failed to start playback:", err)
	}

	// 5. Launch BubbleTea TUI
	uiModel := NewUIModel(trackPath, vlcPlayer, meta)
	program := tea.NewProgram(uiModel)

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running TUI application: %v\n", err)
		os.Exit(1)
	}
}
