package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mmp/internal/flacmeta"
	"mmp/internal/library"
	"mmp/internal/player"
)

const (
	seekStep    = 5 * time.Second
	volumeStep  = 0.1
	speedFactor = 16.0 / 15
)

type level int

const (
	levelTracks level = iota
	levelAlbums
)

type Model struct {
	audio  *player.Panel
	albums []library.Album

	album, track       int // playing
	selAlbum, selTrack int // browse cursor
	level              level

	width   int
	loading bool
	shuffle bool
	err     error
	art     string
}

type tickMsg time.Time

// Run blocks until the user quits. audio is the already-loaded albums[0].Tracks[0].
func Run(albums []library.Album, audio *player.Panel) error {
	_, err := tea.NewProgram(Model{audio: audio, albums: albums}, tea.WithAltScreen()).Run()
	return err
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) tracks() []library.Track        { return m.albums[m.album].Tracks }
func (m Model) current() *flacmeta.Metadata    { return m.tracks()[m.track].Meta }
func (m Model) Init() tea.Cmd                  { return tick() }
func (m Model) browsedTracks() []library.Track { return m.albums[m.selAlbum].Tracks }

// following reports whether the cursor sits on the playing item and should move with playback.
func (m Model) following() bool {
	return m.selAlbum == m.album && (m.level == levelAlbums || m.selTrack == m.track)
}

func (m *Model) load(a, t int) tea.Cmd {
	if m.following() {
		m.selAlbum, m.selTrack = a, t
	}
	m.album, m.track, m.loading = a, t, true
	return loadTrack(m.albums[a].Tracks[t].Path)
}

func (m *Model) advance() tea.Cmd {
	if m.loading {
		return nil
	}
	if a, t, ok := next(m.albums, m.album, m.track, m.shuffle); ok {
		return m.load(a, t)
	}
	return nil
}

func (m *Model) back() tea.Cmd {
	if m.loading {
		return nil
	}
	if a, t, ok := prev(m.albums, m.album, m.track); ok {
		return m.load(a, t)
	}
	return nil
}

func (m *Model) openAlbum() {
	m.level, m.selTrack = levelTracks, 0
	if m.selAlbum == m.album {
		m.selTrack = m.track
	}
}

func (m *Model) moveCursor(delta int) {
	if m.level == levelAlbums {
		m.selAlbum = max(0, min(m.selAlbum+delta, len(m.albums)-1))
		return
	}
	m.selTrack = max(0, min(m.selTrack+delta, len(m.browsedTracks())-1))
}

func (m *Model) choose() tea.Cmd {
	if m.level == levelAlbums {
		m.openAlbum()
		return nil
	}
	if m.loading {
		return nil
	}
	return m.load(m.selAlbum, m.selTrack)
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
	case "n":
		return m, m.advance()
	case "p":
		return m, m.back()
	case "up":
		m.moveCursor(-1)
	case "down":
		m.moveCursor(1)
	case "tab":
		if m.level == levelAlbums {
			m.openAlbum()
		} else {
			m.level = levelAlbums
		}
	case "enter":
		return m, m.choose()
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
