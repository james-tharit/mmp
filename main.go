package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	vlc "github.com/adrg/libvlc-go/v3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhowden/tag"
)

// Define UI styles using Lipgloss
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF66")).MarginLeft(2)
	metaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginLeft(2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginTop(1).MarginLeft(2)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).BorderForeground(lipgloss.Color("#00AAFF")).Width(50)
)

// Track metadata structure
type TrackInfo struct {
	Title    string
	Artist   string
	Album    string
	Format   string
	Duration time.Duration
}

// Bubble Tea Model representing application state
type model struct {
	player    *vlc.Player
	meta      TrackInfo
	isPlaying bool
	filePath  string
}

// Extract tags from M4A or FLAC files safely
func getTrackMetadata(path string) TrackInfo {
	fallback := TrackInfo{Title: "Unknown Title", Artist: "Unknown Artist", Album: "Unknown Album", Format: "Unknown"}

	f, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		// If no tags exist, use the filename as a fallback title
		parts := strings.Split(path, "/")
		fallback.Title = parts[len(parts)-1]
		return fallback
	}

	extParts := strings.Split(path, ".")
	format := strings.ToUpper(extParts[len(extParts)-1])

	return TrackInfo{
		Title:  m.Title(),
		Artist: m.Artist(),
		Album:  m.Album(),
		Format: format,
	}
}

// Initialize the TUI model
func initialModel(path string, p *vlc.Player) model {
	return model{
		player:    p,
		meta:      getTrackMetadata(path),
		isPlaying: true,
		filePath:  path,
	}
}

// Init command for Bubble Tea
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles user input and actions (Model-View-Update architecture)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.player.Stop()
			return m, tea.Quit

		case " ": // Spacebar toggles pause/play
			if m.isPlaying {
				m.player.SetPause(true)
				m.isPlaying = false
			} else {
				m.player.Play()
				m.isPlaying = true
			}
		}
	}
	return m, nil
}

// View renders the terminal screen layout
func (m model) View() string {
	status := "⏸ PAUSED"
	if m.isPlaying {
		status = "▶ PLAYING"
	}

	// Build the visual interface text blocks
	uiText := fmt.Sprintf(
		"%s\n\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n\n"+
			"%s",
		titleStyle.Render("GO-MMP: TERMINAL AUDIO PLAYER"),
		lipgloss.NewStyle().Bold(true).Render("  Track: "), m.meta.Title,
		lipgloss.NewStyle().Bold(true).Render("  Artist:"), m.meta.Artist,
		lipgloss.NewStyle().Bold(true).Render("  Album: "), m.meta.Album,
		lipgloss.NewStyle().Bold(true).Render("  Codec: "), fmt.Sprintf("%s (libvlc backend)", m.meta.Format),
		lipgloss.NewStyle().Background(lipgloss.Color("#333333")).PaddingLeft(1).PaddingRight(1).Render(status),
	)

	help := helpStyle.Render("Controls: [Space] Play/Pause  |  [q] Quit Player")

	return boxStyle.Render(uiText + help)
}

func main() {
	// 1. Point to your music file (M4A or FLAC)
	trackPath := "./0000620687.320.m4a"

	if _, err := os.Stat(trackPath); os.IsNotExist(err) {
		log.Fatalf("Error: Test file '%s' not found. Put an m4a/flac file here and name it sample.m4a", trackPath)
	}

	// 2. Initialize libvlc quietly without standard library terminal noise
	if err := vlc.Init("--no-video", "--quiet"); err != nil {
		log.Fatal("VLC Init Error:", err)
	}
	defer vlc.Release()

	player, err := vlc.NewPlayer()
	if err != nil {
		log.Fatal("Player Error:", err)
	}
	defer player.Release()

	// 3. Prepare the audio track
	_, err = player.LoadMediaFromPath(trackPath)
	if err != nil {
		log.Fatal("Load Error:", err)
	}

	// Start playback immediately before launching the TUI view loop
	player.Play()

	// 4. Fire up the Bubble Tea TUI Engine
	p := tea.NewProgram(initialModel(trackPath, player))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI application: %v", err)
		os.Exit(1)
	}
}
