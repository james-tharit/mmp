# Go-MMP: Cross-Platform TUI Music Player

A high-performance, lightweight Terminal User Interface (TUI) music player written in Go. This player effortlessly handles high-fidelity formats like **FLAC** with real-time playback control, speed adjustment, and volume control, all driven by the pure-Go `beep` audio engine with a responsive BubbleTea terminal interface.

## Keybindings

    Space        — Toggle Play / Pause state seamlessly.
    
    W / Right    — Seek forward 1 second.
    
    Left Arrow   — Seek backward 1 second.
    
    A / S        — Volume down / up.
    
    Z / X        — Playback speed down / up.
    
    q / Ctrl+C   — Stop playback and safely exit the application interface.
    
---

## How It Works (The Architecture)

This application uses a clean, pure-Go architecture with real-time audio control:

1. **The Application Layer (Go & Bubble Tea):** Manages keybindings (Space to pause, arrow keys to seek, A/S for volume, Z/X for speed), orchestrates application state, and renders a responsive terminal interface with real-time playback information.

2. **The Audio Engine (`beep`):** A pure-Go audio streaming library that handles:
   - FLAC file decoding and streaming
   - Sample rate resampling for speed control
   - Volume mixing and effects processing
   - Direct speaker output with minimal latency

3. **The Speaker Module (`speaker`):** Initializes and manages the audio output pipeline, coordinating between the decoder, effects chain, and system audio hardware.

---

## System Setup

### Prerequisites
- Go 1.21 or later
- ALSA (Linux) or CoreAudio (macOS) already installed on your system

### Linux Setup (Debian / Ubuntu / Kali)

Install the necessary audio development libraries:
```bash
sudo apt update
sudo apt install libasound2-dev -y
```

### macOS Setup

No additional setup required—CoreAudio is built-in. Just ensure you have Xcode Command Line Tools:
```bash
xcode-select --install
```
## Building and Running the Application

Clone the repository and build:
```bash
# Download and link project dependencies
go mod tidy

# Build the application
go build -o mmp

# Run with a FLAC file
./mmp song.flac
```

Or run directly:
```bash
go run main.go song.flac
```


Change .m4a to .flac
```sh
ffmpeg -i <song>.m4a -c:a flac <output>.flac
```

See Metadata of file
```sh
ffprobe -v error -show_entries format_tags:stream_tags -of default=noprint_wrappers=1 <target>.flac
```