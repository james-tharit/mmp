# CLAUDE.md

`mmp` — a FLAC music player in Go with terminal UI (BubbleTea/lipgloss) over a shared audio engine.

## Commands

```bash
go build -o mmp        # build
go test ./...          # test (audio_panel_test.go, loadtrack_test.go)
./mmp <file.flac>      # run on a file
./mmp <dir>            # run on a directory (recurses for .flac)
```

Playback needs ALSA (Linux) / CoreAudio (macOS).

## Layout (single `package main`, flat)

- `main.go` — CLI parsing, path→FLAC-file resolution, speaker init, launches the TUI.
- `audio.go` — `audioPanel`: beep streamer chain (loop → speed resampler → speaker resampler → volume). Also `loadTrackCmd` for async track loading.
- `flac.go` — `FLACMetadata` + `getMetadata`: hand-rolled VorbisComment (block type 4) and PICTURE (block type 6) parsing via go-flac.
- `ui.go` — `UIModel`, the BubbleTea TUI (keybindings, ANSI album art).

## Notes

- Speaker is initialized once in `main.go` at the first track's sample rate (`speakerSR`); every track resamples to that baseline. Don't re-init the speaker per track.
- VorbisComment parsing keys are uppercased before matching (spec says field names are case-insensitive).
