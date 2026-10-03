package ui

import (
	"math/rand/v2"

	tea "github.com/charmbracelet/bubbletea"

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

// nextIndex returns the track to play after cur, or -1 when there is none
// (single track, or end of a sequential playlist).
func nextIndex(cur, n int, shuffle bool) int {
	switch {
	case n <= 1:
		return -1
	case shuffle:
		for {
			if i := rand.IntN(n); i != cur {
				return i
			}
		}
	case cur+1 < n:
		return cur + 1
	}
	return -1
}
