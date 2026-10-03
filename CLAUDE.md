# CLAUDE.md

`mmp` — a FLAC music player in Go with terminal UI (BubbleTea/lipgloss) over a shared audio engine.

## Commands

```bash
go build -o mmp        # build
go test ./...          # test
./mmp <file.flac>      # run on a file
./mmp <dir>            # run on a directory (recurses for .flac)
```

Playback needs ALSA (Linux) / CoreAudio (macOS).

## Layout

`main.go` (CLI, path→FLAC resolution, speaker init) over `internal/` packages; tests sit beside the code they cover.

- `internal/player` — `Panel`: beep chain (stream → speed resampler → speaker resampler → volume). All speaker locking lives here; `Load` opens a track.
- `internal/flacmeta` — `Metadata` + `Read`: hand-rolled VorbisComment (block 4) and PICTURE (block 6) parsing via go-flac.
- `internal/library` — `Build` groups tracks into `Album`s (Album tag, else parent dir; sorted by track number) and `Search` (all terms, case-insensitive).
- `internal/ui` — BubbleTea TUI: `model.go` (state, playing vs browse cursor, keys), `view.go` (layout), `art.go` (cover), `playlist.go` (async load, next-track), `styles.go`.
- `internal/kitty` — kitty graphics protocol cover art.
- `internal/spectrum` — FFT spectrogram + lossy/lossless guess. Not wired into the UI yet.

## Notes

- Speaker is initialized once (`player.InitSpeaker`) at the first track's sample rate; every track resamples to that baseline. Don't re-init the speaker per track.
- VorbisComment parsing keys are uppercased before matching (spec says field names are case-insensitive).
