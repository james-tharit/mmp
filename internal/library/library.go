package library

import (
	"cmp"
	"path/filepath"
	"slices"

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
