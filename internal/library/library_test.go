package library

import (
	"testing"

	"mmp/internal/flacmeta"
)

func TestBuild(t *testing.T) {
	paths := []string{"/m/x/b.flac", "/m/x/a.flac", "/m/y/c.flac", "/m/x/z.flac"}
	meta := []*flacmeta.Metadata{
		{Album: "Blue", TrackNumber: 2},
		{Album: "Blue", TrackNumber: 1},
		{}, // untagged: falls back to directory name "y"
		{Album: "Blue", TrackNumber: 3},
	}

	albums := Build(paths, meta)
	if len(albums) != 2 || albums[0].Name != "Blue" || albums[1].Name != "y" {
		t.Fatalf("albums = %+v", albums)
	}
	var got []string
	for _, tr := range albums[0].Tracks {
		got = append(got, tr.Path)
	}
	want := []string{"/m/x/a.flac", "/m/x/b.flac", "/m/x/z.flac"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("track order = %v, want %v", got, want)
	}
}
