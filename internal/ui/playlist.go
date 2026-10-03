package ui

import (
	"math/rand/v2"

	tea "github.com/charmbracelet/bubbletea"

	"mmp/internal/library"
	"mmp/internal/player"
)

type trackLoadedMsg struct {
	audio *player.Panel
	err   error
}

func loadTrack(path string) tea.Cmd {
	return func() tea.Msg {
		p, err := player.Load(path)
		return trackLoadedMsg{audio: p, err: err}
	}
}

// next returns the position after (a, t). Shuffle stays inside the album and
// picks a different random track; otherwise it walks on, spilling into the
// next album. ok is false at the end of the library.
func next(albums []library.Album, a, t int, shuffle bool) (int, int, bool) {
	n := len(albums[a].Tracks)
	switch {
	case shuffle && n > 1:
		for {
			if i := rand.IntN(n); i != t {
				return a, i, true
			}
		}
	case t+1 < n:
		return a, t + 1, true
	case a+1 < len(albums):
		return a + 1, 0, true
	}
	return 0, 0, false
}

func prev(albums []library.Album, a, t int) (int, int, bool) {
	switch {
	case t > 0:
		return a, t - 1, true
	case a > 0:
		return a - 1, len(albums[a-1].Tracks) - 1, true
	}
	return 0, 0, false
}

// window returns the [start, end) slice of n rows to show, keeping cursor centered.
func window(cursor, n, rows int) (int, int) {
	start := max(0, min(cursor-rows/2, n-rows))
	return start, min(start+rows, n)
}
