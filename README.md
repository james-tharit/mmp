# mmp

![mmp](./cover.png)
A terminal FLAC music player in Go. Built with [beep](https://github.com/gopxl/beep) (audio) and [Bubble Tea](https://github.com/charmbracelet/bubbletea) (UI).

## Install

Needs Go, plus ALSA on Linux (`sudo apt install libasound2-dev`) or CoreAudio on macOS (built in).

```bash
go build -o mmp
```

## Usage

```bash
./mmp song.flac      # play a file
./mmp ~/Music        # play a directory (recurses for .flac)
```

## Keys

| Key | Action |
|-----|--------|
| `Space` | Play / pause |
| `←` / `→` | Seek -5s / +5s |
| `n` / `↓` | Next track |
| `p` / `↑` | Previous track |
| `r` | Toggle shuffle |
| `a` / `s` | Volume down / up |
| `z` / `x` | Speed down / up |
| `q` / `Esc` / `Ctrl+C` | Quit |

## Tips

Convert m4a to flac:
```bash
ffmpeg -i song.m4a -c:a flac song.flac
```

Show a file's tags:
```bash
ffprobe -v error -show_entries format_tags:stream_tags -of default=noprint_wrappers=1 song.flac
```
