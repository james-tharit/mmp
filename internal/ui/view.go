package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const footerIcon = "🛰️"

var footerKeys = []string{"[Space] Pause", "[←/→] Seek", "[A/S] Vol", "[Z/X] Speed", "[N/P] Prev/Next"}

func (m Model) View() string {
	if m.err != nil {
		return appStyle.Render(fmt.Sprintf("Error: %v\nPress 'q' to quit.", m.err))
	}

	st := m.audio.State()

	badge := playingStyle.Render("▶ PLAYING")
	switch {
	case m.loading:
		badge = pausedStyle.Render("⏳ LOADING")
	case st.Paused:
		badge = pausedStyle.Render("⏸ PAUSED")
	}
	title := fmt.Sprintf(" Minimal Music Player [%d/%d] ", m.idx+1, len(m.playlist))
	header := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render(title), "   ", badge)

	art := m.art
	if art == "" {
		art = lipgloss.Place(artCols, artRows, lipgloss.Center, lipgloss.Center, helpStyle.Render("♪ no cover"))
	}
	artColumn := lipgloss.NewStyle().MarginLeft(2).Width(artCols).Height(artRows).MaxHeight(artRows).Render(art)
	infoColumn := m.renderInfo(st.Position, st.Length, st.Volume, st.Speed)
	listColumn := m.renderList()

	main := lipgloss.JoinHorizontal(lipgloss.Top, artColumn, infoColumn, listColumn)
	if m.narrow() {
		main = lipgloss.JoinVertical(lipgloss.Left, artColumn, "", infoColumn, "", listColumn)
	}

	shuffle := "OFF"
	if m.shuffle {
		shuffle = "ON"
	}
	keys := append(footerKeys[:len(footerKeys):len(footerKeys)], "[R] Shuffle:"+shuffle, "[Esc/Q] Quit")

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", main, "", m.renderFooter(keys))
	return appStyle.Render(body) + "\n"
}

func (m Model) narrow() bool { return m.width > 0 && m.width < wideLayoutMin }

// panel styles a column: left-bordered when wide, plain and clamped to the terminal when stacked.
func (m Model) panel(width int) lipgloss.Style {
	if !m.narrow() {
		return lipgloss.NewStyle().Width(width).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(gray).PaddingLeft(2).PaddingRight(1)
	}
	if w := m.width - frameInset; w < width {
		width = max(w, 20)
	}
	return lipgloss.NewStyle().Width(width).PaddingLeft(2)
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func row(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render(label), valueStyle.Render(value))
}

func (m Model) renderInfo(position, length time.Duration, volume, speed float64) string {
	md := m.current()
	sampleRate := "NaN Hz"
	if md.SampleRate > 0 {
		sampleRate = fmt.Sprintf("%d Hz", md.SampleRate)
	}
	volumePercent := max(0, int((volume+5.0)/5.0*100))

	return m.panel(infoWidth).Render(lipgloss.JoinVertical(lipgloss.Left,
		row("Title:", orDefault(md.Title, "Unknown Title")),
		row("Artist:", orDefault(md.Artist, "Unknown Artist")),
		row("Album:", orDefault(md.Album, "Unknown Album")),
		row("Samples:", sampleRate),
		row("Volume (A/S):", fmt.Sprintf("%d%%", volumePercent)),
		row("Speed  (Z/X):", fmt.Sprintf("%.2fx", speed)),
		"",
		progressBar(position, length),
		helpStyle.Render(fmt.Sprintf(" %s / %s", position.Round(time.Second), length.Round(time.Second))),
	))
}

func (m Model) renderList() string {
	start := max(0, min(m.idx-listRows/2, len(m.meta)-listRows))
	end := min(start+listRows, len(m.meta))

	lines := make([]string, listRows)
	for i := start; i < end; i++ {
		md := m.meta[i]
		line := lipgloss.NewStyle().Inline(true).MaxWidth(listWidth - 5).
			Render(orDefault(md.Title, "Unknown") + " - " + orDefault(md.Artist, "Unknown"))
		if i == m.idx {
			line = playingStyle.Render("▶ " + line)
		} else {
			line = "  " + line
		}
		lines[i-start] = line
	}
	return m.panel(listWidth).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func progressBar(position, length time.Duration) string {
	filled := 0
	if length > 0 {
		filled = int(math.Round(float64(position) / float64(length) * barWidth))
		filled = max(0, min(filled, barWidth))
	}
	return barFullStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", barWidth-filled))
}

// renderFooter joins help items with " • ", wrapping between items to fit the terminal.
func (m Model) renderFooter(items []string) string {
	limit := m.width - frameInset
	var lines []string
	line := footerIcon
	for _, it := range items {
		switch {
		case line == footerIcon:
			line += " " + it
		case limit <= 0 || lipgloss.Width(line+" • "+it) <= limit:
			line += " • " + it
		default:
			lines = append(lines, line)
			line = it
		}
	}
	return helpStyle.Render(strings.Join(append(lines, line), "\n"))
}
