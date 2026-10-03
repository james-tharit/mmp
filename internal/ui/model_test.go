package ui

import "testing"

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		next, _ := m.handleKey(k)
		m = next.(Model)
	}
	return m
}

func TestBrowseDoesNotMovePlayback(t *testing.T) {
	m := Model{albums: albums(3, 2, 4)}

	m = press(m, "tab", "down", "down") // albums level, cursor on album 2
	if m.level != levelAlbums || m.selAlbum != 2 || m.album != 0 || m.track != 0 {
		t.Fatalf("browse: %+v", m)
	}
	m = press(m, "down") // clamped at last album
	if m.selAlbum != 2 {
		t.Fatalf("cursor not clamped: %d", m.selAlbum)
	}

	m = press(m, "enter", "down", "down") // open album 2, cursor on track 2
	if m.level != levelTracks || m.selTrack != 2 || m.album != 0 {
		t.Fatalf("open album: %+v", m)
	}

	m = press(m, "enter") // play selected track
	if m.album != 2 || m.track != 2 || !m.loading {
		t.Fatalf("choose: album=%d track=%d loading=%v", m.album, m.track, m.loading)
	}
}

func TestCursorFollowsPlaybackOnlyWhenOnPlayingItem(t *testing.T) {
	m := Model{albums: albums(3, 3)}
	m = m.advanceForTest()
	if m.selAlbum != 0 || m.selTrack != 1 {
		t.Fatalf("cursor should follow: %d,%d", m.selAlbum, m.selTrack)
	}

	m = press(m, "tab", "down", "enter") // wander to album 1
	m.loading = false
	m = m.advanceForTest()
	if m.selAlbum != 1 || m.selTrack != 0 {
		t.Fatalf("wandered cursor must stay put: %d,%d", m.selAlbum, m.selTrack)
	}
}

func (m Model) advanceForTest() Model {
	m.advance()
	return m
}
