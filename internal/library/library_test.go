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

func TestFind(t *testing.T) {
	albums := []Album{
		{Tracks: []Track{{Path: "/a/1.flac"}, {Path: "/a/2.flac"}}},
		{Tracks: []Track{{Path: "/b/1.flac"}}},
	}
	if a, tr, ok := Find(albums, "/b/1.flac"); !ok || a != 1 || tr != 0 {
		t.Fatalf("Find hit = %d,%d,%v", a, tr, ok)
	}
	if a, tr, ok := Find(albums, "/missing.flac"); ok || a != 0 || tr != 0 {
		t.Fatalf("Find miss = %d,%d,%v", a, tr, ok)
	}
}

func TestSearch(t *testing.T) {
	albums := []Album{
		{Name: "Kind of Blue", Tracks: []Track{
			{Path: "/m/so-what.flac", Meta: &flacmeta.Metadata{Title: "So What", Artist: "Miles Davis"}},
			{Path: "/m/flamenco.flac", Meta: &flacmeta.Metadata{Title: "Flamenco Sketches", Artist: "Miles Davis"}},
		}},
		{Name: "OK Computer", Tracks: []Track{
			{Path: "/m/x/01.flac", Meta: &flacmeta.Metadata{Title: "Airbag", Artist: "Radiohead"}},
		}},
	}
	cases := []struct {
		query string
		want  []Ref
	}{
		{"", nil},
		{"   ", nil},
		{"MILES", []Ref{{0, 0}, {0, 1}}},
		{"miles what", []Ref{{0, 0}}},
		{"computer air", []Ref{{1, 0}}},
		{"so-what", []Ref{{0, 0}}},
		{"zzz", nil},
	}
	for _, c := range cases {
		got := Search(albums, c.query)
		if len(got) != len(c.want) {
			t.Fatalf("Search(%q) = %v, want %v", c.query, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("Search(%q) = %v, want %v", c.query, got, c.want)
			}
		}
	}
}
