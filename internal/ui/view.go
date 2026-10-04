package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"mmp/internal/player"
)

const footerIcon = "🛰️"

var footerKeys = []string{"[Space] Pause", "[←/→] Seek", "[A/S] Vol", "[Z/X] Speed", "[N/P] Prev/Next", "[↑/↓] Browse", "[Tab] Albums", "[Enter] Select", "[/] Search", "[R] Shuffle", "[Esc/Q] Quit"}

var searchKeys = []string{"[↑/↓] Select", "[Enter] Play", "[Esc] Cancel"}

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
	shuffle := pausedStyle.Render("🔀 SHUFFLE OFF")
	if m.shuffle {
		shuffle = playingStyle.Render("🔀 SHUFFLE ON")
	}
	status := lipgloss.JoinHorizontal(lipgloss.Center, badge, "   ", shuffle)

	title := titleStyle.Render(fmt.Sprintf(" Minimal Music Player [%d/%d] ", m.track+1, len(m.tracks())))
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, "   ", status)

	art := m.art
	if art == "" {
		art = lipgloss.Place(artCols, artRows, lipgloss.Center, lipgloss.Center, helpStyle.Render("♪ no cover"))
	}
	artColumn := lipgloss.NewStyle().MarginLeft(2).Width(artCols).Height(artRows).MaxHeight(artRows).Render(art)
	infoColumn := m.renderInfo(st)
	listColumn := m.renderList()

	main := lipgloss.JoinHorizontal(lipgloss.Top, artColumn, infoColumn, listColumn)
	if m.narrow() {
		header = title
		main = lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().MarginLeft(2).Render(status), artColumn, "", infoColumn, "", listColumn)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, header, main, "", m.renderFooter(m.footerKeys(), lipgloss.Width(main)))
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

func (m Model) renderInfo(st player.State) string {
	md := m.current()
	sampleRate := "NaN Hz"
	if md.SampleRate > 0 {
		sampleRate = fmt.Sprintf("%d Hz", md.SampleRate)
	}
	volumePercent := max(0, int((st.Volume+5.0)/5.0*100))

	return m.panel(infoWidth).Render(lipgloss.JoinVertical(lipgloss.Left,
		row("Title:", orDefault(md.Title, "Unknown Title")),
		row("Artist:", orDefault(md.Artist, "Unknown Artist")),
		row("Album:", orDefault(md.Album, "Unknown Album")),
		row("Samples:", sampleRate),
		row("Volume (A/S):", fmt.Sprintf("%d%%", volumePercent)),
		row("Speed  (Z/X):", fmt.Sprintf("%.2fx", st.Speed)),
		"",
		progressBar(st.Position, st.Length, st.Paused),
		helpStyle.Render(fmt.Sprintf(" %s / %s", st.Position.Round(time.Second), st.Length.Round(time.Second))),
	))
}

func (m Model) footerKeys() []string {
	if m.searching {
		return searchKeys
	}
	return footerKeys
}

func (m Model) listContents() (title string, items []listItem, cursor int) {
	switch {
	case m.searching:
		for _, h := range m.hits {
			a, tr := m.albums[h.Album], m.albums[h.Album].Tracks[h.Track]
			items = append(items, listItem{
				fmt.Sprintf("%s - %s · %s", orDefault(tr.Meta.Title, "Unknown"), orDefault(tr.Meta.Artist, "Unknown"), a.Name),
				h.Album == m.album && h.Track == m.track,
			})
		}
		return "/ " + m.query + "▌", items, m.selHit
	case m.level == levelTracks:
		for i, tr := range m.browsedTracks() {
			items = append(items, listItem{
				orDefault(tr.Meta.Title, "Unknown") + " - " + orDefault(tr.Meta.Artist, "Unknown"),
				m.selAlbum == m.album && i == m.track,
			})
		}
		return m.albums[m.selAlbum].Name, items, m.selTrack
	}
	for i, a := range m.albums {
		items = append(items, listItem{fmt.Sprintf("%s (%d)", a.Name, len(a.Tracks)), i == m.album})
	}
	return "Albums", items, m.selAlbum
}

func (m Model) renderList() string {
	title, items, cursor := m.listContents()

	start, end := window(cursor, len(items), listRows)
	clip := lipgloss.NewStyle().Inline(true).MaxWidth(listWidth - 5)
	lines := make([]string, listRows+1)
	lines[0] = listTitleStyle.Render(clip.Render(title))
	if m.searching && len(items) == 0 {
		hint := "no matches"
		if m.query == "" {
			hint = "type to search"
		}
		lines[1] = helpStyle.UnsetMarginTop().Render("  " + hint)
	}
	for i := start; i < end; i++ {
		marker := "  "
		if items[i].playing {
			marker = "▶ "
		}
		line := marker + clip.Render(items[i].text)
		switch {
		case i == cursor:
			line = valueStyle.Render(line)
		case items[i].playing:
			line = playingStyle.Render(line)
		}
		lines[i-start+1] = line
	}
	return m.panel(listWidth).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

type listItem struct {
	text    string
	playing bool
}

func progressBar(position, length time.Duration, paused bool) string {
	filled := 0
	if length > 0 {
		filled = int(math.Round(float64(position) / float64(length) * barWidth))
		filled = max(0, min(filled, barWidth))
	}
	full := barFullStyle
	if paused {
		full = barPausedStyle
	}
	return full.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", barWidth-filled))
}

func (m Model) renderFooter(items []string, contentWidth int) string {
	limit := contentWidth
	if m.width > 0 {
		limit = min(limit, m.width-frameInset)
	}
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
