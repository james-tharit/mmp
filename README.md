# Go-MMP: Cross-Platform TUI Music Player

A high-performance, lightweight Terminal User Interface (TUI) music player written in Go. This player effortlessly handles high-fidelity formats like **FLAC** and **AAC-LC (.m4a)** by leveraging native system audio engines via `libvlc` bindings, controlled completely inside a responsive terminal framework.

## Keybindings

    Space — Toggle Play / Pause state seamlessly.

    q or Ctrl+C — Stop audio tracking and safely exit the application interface.
    
---

## How It Works (The Architecture)

Instead of relying on experimental, heavy, or pure-Go decoders that struggle with complex proprietary container profiles like `.m4a`, this application separates the application layer from the hardware decoding layer:

1. **The Application Layer (Go & Bubble Tea):** Manages keybindings (`Space` to pause, `q` to quit), orchestrates application states, and extracts ID3/Vorbis comment metadata dynamically from file headers.
2. **The CGO Bridge (`libvlc-go`):** Acts as a highly optimized safe-wrapper that passes commands across the Go runtime to native C-libraries.
3. **The Playback Layer (`libvlc` Engine):** The exact engine running behind VLC Media Player handles the heavy lifting—decoding, track buffering, and direct hardware communication with system sound servers like **CoreAudio** (macOS) or **PipeWire/PulseAudio** (Linux).

---

## macOS Setup Guide

Because this project links directly to compiled native binaries via CGO, compiling on macOS requires setting up the target system headers and letting Go know where the dynamic libraries reside.

### 1. Prerequisites (Xcode Tools)
Ensure you have a C compiler installed on your Mac. Open your terminal and run:
```bash
xcode-select --install
```

### 2. Install VLC via Homebrew

Install the VLC application package. Homebrew automatically places the library binaries (libvlc.dylib) in your system environment paths.
```Bash
brew install --cask vlc
```
### 3. Configure Environment Paths

Go needs help locating the VLC headers and plugins depending on whether you are running on modern Apple Silicon (M-series) or older Intel hardware.
For Apple Silicon (M1 / M2 / M3 / M4 Macs)

Add these exports to your shell configuration profile (~/.zshrc or ~/.bash_profile):

```Bash
# Point CGO to Homebrew's ARM64 library location
export CGO_CFLAGS="-I/Applications/VLC.app/Contents/MacOS/include"
export CGO_LDFLAGS="-L/Applications/VLC.app/Contents/MacOS/lib -Wl,-rpath,/Applications/VLC.app/Contents/MacOS/lib"

# Tell libvlc where to find its runtime decoding engines
export VLC_PLUGIN_PATH="/Applications/VLC.app/Contents/MacOS/plugins"
```
For Intel Macs (x86_64)

Add these to your shell profile instead:
```Bash

export CGO_CFLAGS="-I/usr/local/include"
export CGO_LDFLAGS="-L/usr/local/lib"
export VLC_PLUGIN_PATH="/Applications/VLC.app/Contents/MacOS/plugins"
```
Don't forget to reload your shell after adding these: source ~/.zshrc
Linux Setup Guide (Debian / Kali / Ubuntu)

To run or build this application on a Debian-based workstation, you need the engine plugins and development header tracking trees:
```Bash

# Install core runtime engine, development headers, and base audio plugins
sudo apt update
sudo apt install vlc libvlc-dev vlc-plugin-base -y

# Export your system architecture's plugin directory path
export VLC_PLUGIN_PATH=/usr/lib/x86_64-linux-gnu/vlc/plugins
```
Building and Running the Application

Once your OS-specific variables are configured, clean your local Go compiler cache and run the module tidy automation to pull down the project framework stacks:
```Bash

# Download and link project framework packages
go mod tidy

# Clear old compilation flags out of cache memory
go clean -cache

# Launch the interactive terminal application dashboard
go run main.go

```
