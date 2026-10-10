package ui

import (
	"testing"

	"mmp/internal/flacmeta"
	"mmp/internal/library"
)

func albums(sizes ...int) []library.Album {
	out := make([]library.Album, len(sizes))
	for i, n := range sizes {
		out[i].Tracks = make([]library.Track, n)
		for j := range out[i].Tracks {
			out[i].Tracks[j].Meta = &flacmeta.Metadata{}
		}
	}
	return out
}

func TestNextAndPrev(t *testing.T) {
	al := albums(2, 1, 3)
	steps := []struct{ a, t, wa, wt int }{{0, 0, 0, 1}, {0, 1, 1, 0}, {1, 0, 2, 0}}
	for _, s := range steps {
		if a, tr, ok := next(al, s.a, s.t, shuffleOff, repeatOff); !ok || a != s.wa || tr != s.wt {
			t.Fatalf("next(%d,%d) = %d,%d,%v", s.a, s.t, a, tr, ok)
		}
	}
	if _, _, ok := next(al, 2, 2, shuffleOff, repeatOff); ok {
		t.Fatal("next at library end must report !ok")
	}
	if a, tr, ok := prev(al, 2, 0); !ok || a != 1 || tr != 0 {
		t.Fatalf("prev across album = %d,%d,%v", a, tr, ok)
	}
	if a, tr, ok := prev(al, 1, 0); !ok || a != 0 || tr != 1 {
		t.Fatalf("prev lands on last track = %d,%d,%v", a, tr, ok)
	}
	if _, _, ok := prev(al, 0, 0); ok {
		t.Fatal("prev at start must report !ok")
	}
}

func TestShuffleStaysInAlbum(t *testing.T) {
	al := albums(4, 4)
	for i := 0; i < 100; i++ {
		a, tr, ok := next(al, 1, 2, shuffleAlbum, repeatOff)
		if !ok || a != 1 || tr == 2 || tr < 0 || tr >= 4 {
			t.Fatalf("shuffle: %d,%d,%v", a, tr, ok)
		}
	}
	if a, tr, ok := next(albums(1, 2), 0, 0, shuffleAlbum, repeatOff); !ok || a != 1 || tr != 0 {
		t.Fatalf("single-track album must advance to next album: %d,%d,%v", a, tr, ok)
	}
}

func TestRepeat(t *testing.T) {
	al := albums(2, 3)
	// repeatOne pins the track.
	if a, tr, ok := next(al, 1, 1, shuffleOff, repeatOne); !ok || a != 1 || tr != 1 {
		t.Fatalf("repeatOne = %d,%d,%v", a, tr, ok)
	}
	// repeatOne wins over shuffle.
	if a, tr, ok := next(al, 1, 2, shuffleAll, repeatOne); !ok || a != 1 || tr != 2 {
		t.Fatalf("repeatOne over shuffle = %d,%d,%v", a, tr, ok)
	}
	// repeatAlbum wraps within the album.
	if a, tr, ok := next(al, 1, 2, shuffleOff, repeatAlbum); !ok || a != 1 || tr != 0 {
		t.Fatalf("repeatAlbum wrap = %d,%d,%v", a, tr, ok)
	}
	// repeatAlbum does not spill forward even if a next album exists.
	if a, tr, ok := next(al, 0, 1, shuffleOff, repeatAlbum); !ok || a != 0 || tr != 0 {
		t.Fatalf("repeatAlbum stays home = %d,%d,%v", a, tr, ok)
	}
	// repeatAll wraps to library start at the end.
	if a, tr, ok := next(al, 1, 2, shuffleOff, repeatAll); !ok || a != 0 || tr != 0 {
		t.Fatalf("repeatAll wrap = %d,%d,%v", a, tr, ok)
	}
}

func TestShuffleAllCrossesAlbums(t *testing.T) {
	al := albums(3, 3, 3)
	hitAlbum := map[int]bool{}
	for i := 0; i < 200; i++ {
		a, tr, ok := next(al, 1, 1, shuffleAll, repeatOff)
		if !ok || tr < 0 || tr >= len(al[a].Tracks) || (a == 1 && tr == 1) {
			t.Fatalf("shuffleAll: %d,%d,%v", a, tr, ok)
		}
		hitAlbum[a] = true
	}
	if !hitAlbum[0] || !hitAlbum[2] {
		t.Fatalf("shuffleAll never left the current album: %v", hitAlbum)
	}
}

func TestWindow(t *testing.T) {
	cases := []struct{ cursor, n, wantStart, wantEnd int }{
		{0, 3, 0, 3}, {0, 30, 0, 10}, {15, 30, 10, 20}, {29, 30, 20, 30},
	}
	for _, c := range cases {
		if s, e := window(c.cursor, c.n, 10); s != c.wantStart || e != c.wantEnd {
			t.Fatalf("window(%d,%d) = %d,%d", c.cursor, c.n, s, e)
		}
	}
}
