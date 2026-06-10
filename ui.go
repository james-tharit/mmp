package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/nfnt/resize"
)

// UIModel represents the BubbleTea application state
type UIModel struct {
	audio         *audioPanel
	filePath      string
	metadata      *FLACMetadata
	terminalWidth int
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
func NewUIModel(filePath string, audio *audioPanel, metadata *FLACMetadata) UIModel {
	return UIModel{
		audio:    audio,
		filePath: filePath,
		metadata: metadata,
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

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		return m, nil
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

// View renders the terminal UI with side-by-side metadata and album art layout
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

	// 1. Header Row
	var statusBadge string
	if isPaused {
		statusBadge = pausedStatusStyle.Render("⏸ PAUSED")
	} else {
		statusBadge = playingStatusStyle.Render("▶ PLAYING")
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render(" Minimal Music Player "), "   ", statusBadge)

	// 2. Build the Left Panel
	title := "Unknown Title"
	artist := "Unknown Artist"
	album := "Unknown Album"
	sampleRateStr := "NaN Hz"

	if m.metadata != nil {
		if m.metadata.Title != "" {
			title = m.metadata.Title
		}
		if m.metadata.Artist != "" {
			artist = m.metadata.Artist
		}
		if m.metadata.Album != "" {
			album = m.metadata.Album
		}
		if m.metadata.SampleRate > 0 {
			sampleRateStr = fmt.Sprintf("%d Hz", m.metadata.SampleRate)
		}
	}

	// Progress Bar Calculations
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

	// Metrics
	volumePercent := int((volume + 5.0) / 5.0 * 100)
	if volumePercent < 0 {
		volumePercent = 0
	}

	rawLeftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Title:"), valueStyle.Render(title)),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Artist:"), valueStyle.Render(artist)),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Album:"), valueStyle.Render(album)),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Samples:"), valueStyle.Render(sampleRateStr)),
		lipgloss.JoinHorizontal(lipgloss.Left, progressBar, helpStyle.Render(timeFormat)),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Volume (A/S):"), valueStyle.Render(fmt.Sprintf("%d%%", volumePercent))),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Speed  (Z/X):"), valueStyle.Render(fmt.Sprintf("%.2fx", speed))),
	)

	// FIX: Give the left column a strictly defined horizontal canvas (48 characters wide)
	// This ensures it never wraps unexpectedly and makes layout math predictable.
	leftColumn := lipgloss.NewStyle().Width(48).Render(rawLeftColumn)

	// 3. Build Right Panel (Album Art)
	var rightColumn string
	if m.metadata != nil && len(m.metadata.AlbumArt) > 0 {

		targetWidth := 50 // High default fallback
		if m.terminalWidth > 0 {
			// FIX: Allocate 100% of the leftover terminal width directly to the album art.
			// Math: Total Width minus Left Column (48), Main Outer Padding (4),
			// and Right Column Margins/Borders (8) = 60 character buffer.
			targetWidth = m.terminalWidth - 60

			// Enforce structural limits so it renders properly even on extreme window sizes
			if targetWidth < 30 {
				targetWidth = 30
			}
			if targetWidth > 150 {
				targetWidth = 150
			}
		}

		// Pass the dynamic available width into the corrected renderer
		artString := renderAlbumArt(m.metadata.AlbumArt, targetWidth)

		if artString != "" {
			rightColumn = lipgloss.NewStyle().
				MarginLeft(4).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(gray).
				PaddingLeft(2).
				Render(artString)
		}
	}

	// 4. Combine Left Column and Right Column Horizontally Safely
	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	// 5. Footer & Assembly
	footer := helpStyle.Render("⚡ [Space] Pause • [←/→] Seek • [A/S] Vol • [Z/X] Speed • [Esc/Q] Quit")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		headerRow,
		"",
		mainLayout,
		"",
		footer,
	)

	// FIX: Explicitly stretch the outermost application box style to span 100%
	// of the available terminal window width.
	activeAppStyle := appStyle.Copy()
	if m.terminalWidth > 0 {
		activeAppStyle = activeAppStyle.Width(m.terminalWidth - 4) // Subtract padding allowance
	}

	return activeAppStyle.Render(body) + "\n"
}

// renderAlbumArt converts the raw image bytes into an ANSI color string block
func renderAlbumArt(artBytes []byte, width int) string {
	if len(artBytes) == 0 {
		return ""
	}

	img, _, err := image.Decode(bytes.NewReader(artBytes))
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	if imgWidth == 0 {
		return ""
	}

	// Calculate target pixel dimensions directly to preserve original aspect ratio.
	pixelWidth := width
	pixelHeight := (width * imgHeight) / imgWidth

	if pixelHeight == 0 {
		pixelHeight = 1
	}

	// Resize the image to fit your target pixel dimensions precisely
	scaledImg := resize.Resize(uint(pixelWidth), uint(pixelHeight), img, resize.Lanczos3)

	ansiImg, err := ansimage.NewFromImage(scaledImg, color.Transparent, ansimage.DitheringWithBlocks)
	if err != nil {
		return ""
	}

	return ansiImg.Render()
}
