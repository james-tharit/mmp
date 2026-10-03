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
		if a, tr, ok := next(al, s.a, s.t, false); !ok || a != s.wa || tr != s.wt {
			t.Fatalf("next(%d,%d) = %d,%d,%v", s.a, s.t, a, tr, ok)
		}
	}
	if _, _, ok := next(al, 2, 2, false); ok {
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
		a, tr, ok := next(al, 1, 2, true)
		if !ok || a != 1 || tr == 2 || tr < 0 || tr >= 4 {
			t.Fatalf("shuffle: %d,%d,%v", a, tr, ok)
		}
	}
	if a, tr, ok := next(albums(1, 2), 0, 0, true); !ok || a != 1 || tr != 0 {
		t.Fatalf("single-track album must advance to next album: %d,%d,%v", a, tr, ok)
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
