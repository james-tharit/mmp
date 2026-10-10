package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gopxl/beep/v2"

	"mmp/internal/flacmeta"
	"mmp/internal/library"
	"mmp/internal/player"
	"mmp/internal/state"
	"mmp/internal/ui"
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file_or_directory>\n", os.Args[0])
		os.Exit(1)
	}
	if err := run(flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	files, err := findFlacs(root)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no FLAC files found to play")
	}

	meta, err := flacmeta.ReadAll(files)
	if err != nil {
		return err
	}

	albums := library.Build(files, meta)
	saved := state.Load()
	a0, t0, resumed := library.Find(albums, saved.TrackPath)
	first := albums[a0].Tracks[t0]

	// The first track's rate becomes the hardware baseline for the whole session.
	if err := player.InitSpeaker(beep.SampleRate(first.Meta.SampleRate)); err != nil {
		return err
	}
	defer player.CloseSpeaker()

	audio, err := player.Load(first.Path)
	if err != nil {
		return err
	}
	if saved.Volume < 0 {
		audio.AdjustVolume(saved.Volume)
	}
	if resumed && saved.Position > 0 {
		audio.SeekBy(saved.Position)
	}
	audio.Play()

	final, err := ui.Run(albums, audio, saved)
	if err != nil {
		return err
	}
	_ = state.Save(final)
	return nil
}

// findFlacs returns root itself if it is a FLAC file, else every .flac beneath it.
func findFlacs(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && flacmeta.IsFlac(path) {
			files = append(files, path)
		}
		return err
	})
	return files, err
}
