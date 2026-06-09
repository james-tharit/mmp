package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep/v2/speaker"
)

// UIModel represents the BubbleTea application state
type UIModel struct {
	audio    *audioPanel
	filePath string
}

// Define Modern UI Theme Styles
var (
	purple = lipgloss.Color("#9867C5")
	yellow = lipgloss.Color("#F1C40F")
	green  = lipgloss.Color("#2ECC71")
	gray   = lipgloss.Color("#444444")
	white  = lipgloss.Color("#EEEEEE")

	appStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 1).
			MarginBottom(1)

	playingStatusStyle = lipgloss.NewStyle().Bold(true).Foreground(green)
	pausedStatusStyle  = lipgloss.NewStyle().Bold(true).Foreground(gray)

	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Width(12)
	valueStyle = lipgloss.NewStyle().Foreground(yellow).Bold(true)

	barEmptyStyle = lipgloss.NewStyle().Foreground(gray)
	barFullStyle  = lipgloss.NewStyle().Foreground(purple)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			MarginTop(1)
)

// NewUIModel creates and returns a new UI model
func NewUIModel(filePath string, audio *audioPanel) UIModel {
	return UIModel{
		audio:    audio,
		filePath: filePath,
	}
}

type tickMsg time.Time

func (m UIModel) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m UIModel) Init() tea.Cmd {
	return m.tick()
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, m.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit

		case " ":
			speaker.Lock()
			m.audio.ctrl.Paused = !m.audio.ctrl.Paused
			speaker.Unlock()

		case "w", "right":
			speaker.Lock()
			newPos := m.audio.streamer.Position() + m.audio.sampleRate.N(time.Second*5) // 5s skip feels smoother
			newPos = min(newPos, m.audio.streamer.Len()-1)
			_ = m.audio.streamer.Seek(newPos)
			speaker.Unlock()

		case "left":
			speaker.Lock()
			newPos := m.audio.streamer.Position() - m.audio.sampleRate.N(time.Second*5)
			newPos = max(newPos, 0)
			_ = m.audio.streamer.Seek(newPos)
			speaker.Unlock()

		case "s":
			speaker.Lock()
			// Cap the volume at 0.0 (original maximum recorded volume)
			// to completely prevent digital clipping.
			m.audio.volume.Volume = min(m.audio.volume.Volume+0.1, 0.0)
			speaker.Unlock()

		case "a":
			speaker.Lock()
			// -5.0 in beep is incredibly quiet (almost silent).
			m.audio.volume.Volume = max(m.audio.volume.Volume-0.1, -5.0)
			speaker.Unlock()

		case "x":
			speaker.Lock()
			newRatio := m.audio.resampler.Ratio() * 16 / 15
			m.audio.resampler.SetRatio(min(newRatio, 3.0))
			speaker.Unlock()

		case "z":
			speaker.Lock()
			newRatio := m.audio.resampler.Ratio() * 15 / 16
			m.audio.resampler.SetRatio(max(newRatio, 0.25))
			speaker.Unlock()
		}
	}
	return m, nil
}

// View renders the terminal UI
func (m UIModel) View() string {
	speaker.Lock()
	positionIdx := m.audio.streamer.Position()
	lengthIdx := m.audio.streamer.Len()
	position := m.audio.sampleRate.D(positionIdx)
	length := m.audio.sampleRate.D(lengthIdx)
	volume := m.audio.volume.Volume
	speed := m.audio.resampler.Ratio()
	isPaused := m.audio.ctrl.Paused
	speaker.Unlock()

	// 1. Header and Status Badge
	var statusBadge string
	if isPaused {
		statusBadge = pausedStatusStyle.Render("⏸ PAUSED")
	} else {
		statusBadge = playingStatusStyle.Render("▶ PLAYING")
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render(" SPEEDY PLAYER "), "   ", statusBadge)

	// 2. Visual Progress Bar calculation (Width: 30 blocks)
	barWidth := 30
	var percent float64
	if lengthIdx > 0 {
		percent = float64(positionIdx) / float64(lengthIdx)
	}
	filledWidth := int(math.Round(percent * float64(barWidth)))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	progressBar := barFullStyle.Render(strings.Repeat("█", filledWidth)) +
		barEmptyStyle.Render(strings.Repeat("░", barWidth-filledWidth))

	timeFormat := fmt.Sprintf(" %s / %s", position.Round(time.Second), length.Round(time.Second))
	progressRow := fmt.Sprintf("%s%s\n", progressBar, helpStyle.Render(timeFormat))

	// 3. Audio Info Metrics Panel
	volumePercent := int((volume + 5.0) / 7.0 * 100) // Rough map to an easily read 0-100% metric
	if volumePercent < 0 {
		volumePercent = 0
	}

	metricsPanel := fmt.Sprintf(
		"%s%s\n%s%s",
		labelStyle.Render("Volume (A/S):"), valueStyle.Render(fmt.Sprintf("%d%%", volumePercent)),
		labelStyle.Render("Speed  (Z/X):"), valueStyle.Render(fmt.Sprintf("%.2fx", speed)),
	)

	// 4. Compact Footer Controls Guide
	footer := helpStyle.Render("⚡ [Space] Pause • [←/→] Seek • [A/S] Vol • [Z/X] Speed • [Esc/Q] Quit")

	// Assembly
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		headerRow,
		"",
		progressRow,
		metricsPanel,
		"",
		footer,
	)

	return appStyle.Render(body) + "\n"
}
