package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TrackInfo holds metadata about an audio track
type TrackInfo struct {
	Title    string
	Artist   string
	Album    string
	Format   string
	Duration time.Duration
}

// UIModel represents the BubbleTea application state
type UIModel struct {
	player    *VLCPlayer
	meta      TrackInfo
	isPlaying bool
	filePath  string
}

// Define UI styles using Lipgloss
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF66")).MarginLeft(2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).MarginTop(1).MarginLeft(2)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).BorderForeground(lipgloss.Color("#00AAFF")).Width(50)
)

// NewUIModel creates and returns a new UI model
func NewUIModel(filePath string, player *VLCPlayer, meta TrackInfo) UIModel {
	return UIModel{
		player:    player,
		meta:      meta,
		isPlaying: true,
		filePath:  filePath,
	}
}

// Init implements the tea.Model interface (initialization)
func (m UIModel) Init() tea.Cmd {
	return nil
}

// Update handles user input and actions
func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.player.Stop()
			return m, tea.Quit

		case " ": // Spacebar toggles pause/play
			if m.isPlaying {
				m.player.Pause()
				m.isPlaying = false
			} else {
				m.player.Resume()
				m.isPlaying = true
			}
		}
	}
	return m, nil
}

// View renders the terminal UI
func (m UIModel) View() string {
	status := "⏸ PAUSED"
	if m.isPlaying {
		status = "▶ PLAYING"
	}

	// Build the visual interface
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
