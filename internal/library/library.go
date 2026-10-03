package library

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"mmp/internal/flacmeta"
)

type Track struct {
	Path string
	Meta *flacmeta.Metadata
}

type Album struct {
	Name   string
	Tracks []Track
}

// Build groups tracks by Album tag, falling back to the parent directory name
// when untagged. Albums keep first-seen order; tracks sort by track number, then path.
func Build(paths []string, meta []*flacmeta.Metadata) []Album {
	var albums []Album
	index := map[string]int{}
	for i, p := range paths {
		name := meta[i].Album
		if name == "" {
			name = filepath.Base(filepath.Dir(p))
		}
		a, ok := index[name]
		if !ok {
			a = len(albums)
			index[name] = a
			albums = append(albums, Album{Name: name})
		}
		albums[a].Tracks = append(albums[a].Tracks, Track{Path: p, Meta: meta[i]})
	}
	for _, a := range albums {
		slices.SortStableFunc(a.Tracks, func(x, y Track) int {
			return cmp.Or(cmp.Compare(x.Meta.TrackNumber, y.Meta.TrackNumber), cmp.Compare(x.Path, y.Path))
		})
	}
	return albums
}

// Ref locates a track in an []Album.
type Ref struct{ Album, Track int }

// Search returns every track whose title, artist, album or filename contains
// all whitespace-separated terms of query (case-insensitive). An empty query matches nothing.
func Search(albums []Album, query string) []Ref {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}
	var hits []Ref
	for ai, a := range albums {
		for ti, tr := range a.Tracks {
			hay := strings.ToLower(strings.Join([]string{tr.Meta.Title, tr.Meta.Artist, a.Name, filepath.Base(tr.Path)}, "\n"))
			if !slices.ContainsFunc(terms, func(t string) bool { return !strings.Contains(hay, t) }) {
				hits = append(hits, Ref{ai, ti})
			}
		}
	}
	return hits
}
