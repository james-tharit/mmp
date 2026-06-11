package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // FIX 1: Required to decode JPEG album art
	_ "image/png"  // FIX 1: Required to decode PNG album art
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep/v2/speaker"
	"golang.org/x/image/draw"
)

// UIModel represents the BubbleTea application state
type UIModel struct {
	audio         *audioPanel
	playlist      []string // path to directory of flac files
	currentIdx    int      // Keep track of current song
	metadata      *FLACMetadata
	terminalWidth int
	loading       bool   // UI flag to show loading state during track changes
	err           error  // Store playback/loading errors
	albumArtCache string // prerendered ANSI album art string to avoid re-rendering every tick
}

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

// NewUIModel creates and returns a new UI model with a playlist
func NewUIModel(playlist []string, audio *audioPanel, metadata *FLACMetadata) UIModel {
	return UIModel{
		audio:      audio,
		playlist:   playlist,
		currentIdx: 0,
		metadata:   metadata,
	}
}

// custom message for ticking the UI
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

	case TrackLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			fmt.Printf("error: %v\n", msg.err)
			return m, nil
		}

		// 1. Wipe the hardware mixer thread clean.
		speaker.Clear()

		// 2. Safely close the old track's streamer.
		// Since speaker.Clear() detached it, the background audio thread is no longer touching it.
		if m.audio != nil && m.audio.streamer != nil {
			if oldStreamer, ok := m.audio.streamer.(interface{ Close() error }); ok {
				_ = oldStreamer.Close()
			}
		}

		// 3. Swap in the new track's state data
		m.audio = msg.audio
		m.metadata = msg.metadata

		// 4. Pre-render the album art cache safely
		m.recalculateAlbumArt()

		// 5. Fire off the new track playback
		m.audio.Play()
		return m, nil

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		// 4. Only recalculate art dimensions if the window size alters
		m.recalculateAlbumArt()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit

		case "n": // NEXT TRACK
			if !m.loading && m.currentIdx < len(m.playlist)-1 {
				m.currentIdx++
				m.loading = true
				return m, loadTrackCmd(m.playlist[m.currentIdx])
			}

		case "p": // PREVIOUS TRACK
			if !m.loading && m.currentIdx > 0 {
				m.currentIdx--
				m.loading = true
				return m, loadTrackCmd(m.playlist[m.currentIdx])
			}

		case " ":
			if m.audio != nil {
				speaker.Lock()
				m.audio.ctrl.Paused = !m.audio.ctrl.Paused
				speaker.Unlock()
			}

		case "w", "right":
			if m.audio != nil {
				speaker.Lock()
				newPos := m.audio.streamer.Position() + m.audio.sampleRate.N(time.Second*5)
				newPos = min(newPos, m.audio.streamer.Len()-1)
				_ = m.audio.streamer.Seek(newPos)
				speaker.Unlock()
			}

		case "left":
			if m.audio != nil {
				speaker.Lock()
				newPos := m.audio.streamer.Position() - m.audio.sampleRate.N(time.Second*5)
				newPos = max(newPos, 0)
				_ = m.audio.streamer.Seek(newPos)
				speaker.Unlock()
			}

		case "s":
			if m.audio != nil {
				speaker.Lock()
				m.audio.volume.Volume = min(m.audio.volume.Volume+0.1, 0.0)
				speaker.Unlock()
			}

		case "a":
			if m.audio != nil {
				speaker.Lock()
				m.audio.volume.Volume = max(m.audio.volume.Volume-0.1, -5.0)
				speaker.Unlock()
			}

		case "x":
			if m.audio != nil {
				speaker.Lock()
				newRatio := m.audio.resampler.Ratio() * 16 / 15
				m.audio.resampler.SetRatio(min(newRatio, 3.0))
				speaker.Unlock()
			}

		case "z":
			if m.audio != nil {
				speaker.Lock()
				newRatio := m.audio.resampler.Ratio() * 15 / 16
				m.audio.resampler.SetRatio(max(newRatio, 0.25))
				speaker.Unlock()
			}
		}
	}
	return m, nil
}

// View renders the terminal UI
func (m UIModel) View() string {
	if m.err != nil {
		return appStyle.Render(fmt.Sprintf("Error: %v\nPress 'q' to quit.", m.err))
	}

	var positionIdx, lengthIdx int
	var position, length time.Duration
	var volume, speed float64
	var isPaused bool

	if m.audio != nil {
		speaker.Lock()
		positionIdx = m.audio.streamer.Position()
		lengthIdx = m.audio.streamer.Len()
		position = m.audio.sampleRate.D(positionIdx)
		length = m.audio.sampleRate.D(lengthIdx)
		volume = m.audio.volume.Volume
		speed = m.audio.resampler.Ratio()
		isPaused = m.audio.ctrl.Paused
		speaker.Unlock()
	}

	// 1. Header Row
	var statusBadge string
	if m.loading {
		statusBadge = pausedStatusStyle.Render("⏳ LOADING")
	} else if isPaused {
		statusBadge = pausedStatusStyle.Render("⏸ PAUSED")
	} else {
		statusBadge = playingStatusStyle.Render("▶ PLAYING")
	}

	// Show Track X of Y
	titleText := fmt.Sprintf(" Minimal Music Player [%d/%d] ", m.currentIdx+1, len(m.playlist))
	headerRow := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render(titleText), "   ", statusBadge)

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
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Volume (A/S):"), valueStyle.Render(fmt.Sprintf("%d%%", volumePercent))),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Speed  (Z/X):"), valueStyle.Render(fmt.Sprintf("%.2fx", speed))),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, progressBar, helpStyle.Render(timeFormat)),
	)

	leftColumn := lipgloss.NewStyle().Width(48).Render(rawLeftColumn)

	// 3. Build Right Panel (Album Art) using the PRE-RENDERED Cache
	var rightColumn string
	if m.albumArtCache != "" {
		rightColumn = lipgloss.NewStyle().
			MarginLeft(4).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(gray).
			PaddingLeft(2).
			Render(m.albumArtCache)
	}

	// 4. Combine Left Column and Right Column
	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	// 5. Footer & Assembly
	// ADDED: [N/P] Next/Prev to the footer instructions
	footer := helpStyle.Render("🛰️ [Space] Pause • [←/→] Seek • [A/S] Vol • [Z/X] Speed • [N/P] Prev/Next • [Esc/Q] Quit")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		headerRow,
		"",
		mainLayout,
		"",
		footer,
	)

	return appStyle.Render(body) + "\n"
}

func (m *UIModel) recalculateAlbumArt() {
	if m.metadata == nil || len(m.metadata.AlbumArt) == 0 {
		m.albumArtCache = ""
		return
	}

	targetWidth := 24
	if m.terminalWidth < 24 {
		targetWidth = m.terminalWidth - 12
		if targetWidth < 12 {
			targetWidth = 6
		}
	}

	// 2. Render the high-quality ANSI block
	m.albumArtCache = renderAlbumArt(m.metadata.AlbumArt, targetWidth)
}

// renderAlbumArt converts raw image bytes into a high-quality 24-bit ANSI string
func renderAlbumArt(artBytes []byte, width int) string {
	if len(artBytes) == 0 || width <= 0 {
		return ""
	}

	img, _, err := image.Decode(bytes.NewReader(artBytes))
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	if imgWidth == 0 || imgHeight == 0 {
		return ""
	}

	// Calculate target pixel dimensions.
	// width = target columns. pixelHeight = target rows.
	pixelWidth := width
	pixelHeight := (width * imgHeight) / imgWidth

	if pixelHeight == 0 {
		pixelHeight = 1
	}

	// 1. High-quality scaling using x/image/draw (Catmull-Rom interpolation)
	dst := image.NewRGBA(image.Rect(0, 0, pixelWidth, pixelHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// 2. Build the ANSI string using True Color (24-bit) and Half-Blocks
	var sb strings.Builder

	// Step by 2 vertically because 1 terminal character will hold 2 vertical pixels
	for y := 0; y < pixelHeight; y += 2 {
		for x := 0; x < pixelWidth; x++ {
			// Get the top pixel (Foreground)
			top := dst.RGBAAt(x, y)

			// Get the bottom pixel (Background) - defaulting to transparent/black if out of bounds
			bottom := color.RGBA{}
			if y+1 < pixelHeight {
				bottom = dst.RGBAAt(x, y+1)
			}

			// Format: \x1b[38;2;R;G;B;48;2;R;G;Bm▀
			// 38;2 sets foreground True Color, 48;2 sets background True Color
			sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀",
				top.R, top.G, top.B,
				bottom.R, bottom.G, bottom.B))
		}
		// Reset ANSI formatting at the end of the row and add a newline
		sb.WriteString("\x1b[0m\n")
	}

	return sb.String()
}
