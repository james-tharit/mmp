# CLAUDE.md

`mmp` — a FLAC music player in Go with two interchangeable frontends over one shared audio engine. GUI (Fyne) is the default; `-tui` selects the terminal UI (BubbleTea/lipgloss).

## Commands

```bash
go build -o mmp        # build
go test ./...          # test (audio_panel_test.go, loadtrack_test.go)
./mmp <file.flac>      # run GUI on a file
./mmp <dir>            # run on a directory (recurses for .flac)
./mmp -tui <path>      # run TUI instead
```

Linux GUI build needs OpenGL/X11/Wayland dev headers (see README "Linux Setup"). TUI builds without them. Playback needs ALSA (Linux) / CoreAudio (macOS).

## Layout (single `package main`, flat)

- `main.go` — CLI parsing, path→FLAC-file resolution, speaker init, launches GUI or TUI.
- `audio.go` — `audioPanel`: beep streamer chain (loop → speed resampler → speaker resampler → volume). Also `loadTrackCmd` for async track loading.
- `flac.go` — `FLACMetadata` + `getMetadata`: hand-rolled VorbisComment (block type 4) and PICTURE (block type 6) parsing via go-flac.
- `ui.go` — `UIModel`, the BubbleTea TUI (keybindings, ANSI album art).
- `gui.go` — `guiUI`, the Fyne GUI. Mirrors the TUI's state and controls.

## Notes

- Speaker is initialized once in `main.go` at the first track's sample rate (`speakerSR`); every track resamples to that baseline. Don't re-init the speaker per track.
- `gui.go` and `ui.go` are parallel frontends — a control or state field added to one usually needs the same in the other.
- VorbisComment parsing keys are uppercased before matching (spec says field names are case-insensitive).
