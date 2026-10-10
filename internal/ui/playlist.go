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

// repeat controls what next() does at a boundary.
type repeat int

const (
	repeatOff   repeat = iota // stop at library end
	repeatOne                 // same track forever
	repeatAlbum               // wrap within the current album
	repeatAll                 // wrap to library start
)

func (r repeat) label() string {
	switch r {
	case repeatOne:
		return "🔁 REPEAT ONE"
	case repeatAlbum:
		return "🔁 REPEAT ALBUM"
	case repeatAll:
		return "🔁 REPEAT ALL"
	}
	return "🔁 REPEAT OFF"
}

// shuffleMode picks the scope random picks draw from.
type shuffleMode int

const (
	shuffleOff   shuffleMode = iota
	shuffleAlbum             // random within the current album
	shuffleAll               // random anywhere in the library
)

func (s shuffleMode) label() string {
	switch s {
	case shuffleAlbum:
		return "🔀 SHUFFLE ALBUM"
	case shuffleAll:
		return "🔀 SHUFFLE ALL"
	}
	return "🔀 SHUFFLE OFF"
}

// randomTrack picks a (album, track) uniformly across every track in the
// library, avoiding the current one unless it is the only track.
func randomTrack(albums []library.Album, curA, curT int) (int, int) {
	total := 0
	for _, a := range albums {
		total += len(a.Tracks)
	}
	if total <= 1 {
		return curA, curT
	}
	for {
		i := rand.IntN(total)
		for ai, a := range albums {
			if i < len(a.Tracks) {
				if ai != curA || i != curT {
					return ai, i
				}
				break
			}
			i -= len(a.Tracks)
		}
	}
}

// next returns the position after (a, t). repeatOne overrides shuffle.
// shuffleAll picks any track in the library; shuffleAlbum stays in the
// current album; otherwise walks on, spilling into the next album.
// ok is false at the end of the library when repeat is off.
func next(albums []library.Album, a, t int, sh shuffleMode, rep repeat) (int, int, bool) {
	n := len(albums[a].Tracks)
	if rep == repeatOne {
		return a, t, true
	}
	switch {
	case sh == shuffleAll:
		ra, rt := randomTrack(albums, a, t)
		return ra, rt, true
	case sh == shuffleAlbum && n > 1:
		for {
			if i := rand.IntN(n); i != t {
				return a, i, true
			}
		}
	case t+1 < n:
		return a, t + 1, true
	case rep == repeatAlbum:
		return a, 0, true
	case a+1 < len(albums):
		return a + 1, 0, true
	case rep == repeatAll:
		return 0, 0, true
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
