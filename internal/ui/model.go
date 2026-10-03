package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mmp/internal/flacmeta"
	"mmp/internal/player"
)

const (
	seekStep    = 5 * time.Second
	volumeStep  = 0.1
	speedFactor = 16.0 / 15
)

type Model struct {
	audio    *player.Panel
	playlist []string
	meta     []*flacmeta.Metadata
	idx      int
	width    int
	loading  bool
	shuffle  bool
	err      error
	art      string
}

type tickMsg time.Time

// Run blocks until the user quits. meta[i] describes playlist[i]; audio is the already-loaded first track.
func Run(playlist []string, audio *player.Panel, meta []*flacmeta.Metadata) error {
	m := Model{audio: audio, playlist: playlist, meta: meta}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) current() *flacmeta.Metadata { return m.meta[m.idx] }

func (m Model) Init() tea.Cmd { return tick() }

func (m *Model) load(idx int) tea.Cmd {
	m.idx, m.loading = idx, true
	return loadTrack(m.playlist[idx])
}

func (m *Model) advance() tea.Cmd {
	if m.loading {
		return nil
	}
	if i := nextIndex(m.idx, len(m.playlist), m.shuffle); i >= 0 {
		return m.load(i)
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		var next tea.Cmd
		if !m.loading && m.audio.Ended() {
			next = m.advance()
		}
		return m, tea.Batch(tick(), next)

	case trackLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Clear detaches the old streamer from the audio thread, making Close safe.
		player.ClearSpeaker()
		_ = m.audio.Close()
		m.audio = msg.audio
		m.audio.Play()
		return m, m.refreshArt()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, m.refreshArt()

	case tea.KeyMsg:
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "n", "down":
		return m, m.advance()
	case "p", "up":
		if !m.loading && m.idx > 0 {
			return m, m.load(m.idx - 1)
		}
	case "r":
		m.shuffle = !m.shuffle
	case " ":
		m.audio.TogglePause()
	case "right":
		m.audio.SeekBy(seekStep)
	case "left":
		m.audio.SeekBy(-seekStep)
	case "s":
		m.audio.AdjustVolume(volumeStep)
	case "a":
		m.audio.AdjustVolume(-volumeStep)
	case "x":
		m.audio.ScaleSpeed(speedFactor)
	case "z":
		m.audio.ScaleSpeed(1 / speedFactor)
	}
	return m, nil
}
