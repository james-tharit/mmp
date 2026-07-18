package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

// guiUI holds the Fyne application state, mirroring the TUI's UIModel.
type guiUI struct {
	win              fyne.Window
	audio            *audioPanel
	playlist         []string
	playlistMetadata []*FLACMetadata
	metadata         *FLACMetadata
	currentIdx       int
	loading          bool
	shuffle          bool

	// widgets
	shuffleBtn *widget.Button
	titleLbl  *widget.Label
	artistLbl *widget.Label
	albumLbl  *widget.Label
	sampleLbl *widget.Label
	volLbl    *widget.Label
	speedLbl  *widget.Label
	timeLbl   *widget.Label
	art       *canvas.Image
	progress  *widget.ProgressBar
	playBtn   *widget.Button
	list      *widget.List
}

// loadTrack is the synchronous sibling of loadTrackCmd; call it inside a goroutine.
func loadTrack(path string) (*audioPanel, *FLACMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	streamer, format, err := flac.Decode(f)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	ap, err := NewAudioPanel(format.SampleRate, streamer)
	if err != nil {
		streamer.Close()
		return nil, nil, err
	}
	md, err := getMetadata(path)
	if err != nil {
		streamer.Close()
		return nil, nil, err
	}
	return ap, md, nil
}

// RunGUI builds and runs the Fyne desktop UI. Same arg order as NewUIModel.
func RunGUI(playlist []string, audio *audioPanel, metadata *FLACMetadata, playlistMetadata []*FLACMetadata) {
	a := app.New()
	win := a.NewWindow("Minimal Music Player")

	m := &guiUI{
		win:              win,
		audio:            audio,
		playlist:         playlist,
		playlistMetadata: playlistMetadata,
		metadata:         metadata,
	}

	// Metadata labels
	m.titleLbl = widget.NewLabel("")
	m.artistLbl = widget.NewLabel("")
	m.albumLbl = widget.NewLabel("")
	m.sampleLbl = widget.NewLabel("")
	m.volLbl = widget.NewLabel("")
	m.speedLbl = widget.NewLabel("")
	m.timeLbl = widget.NewLabel(" 0s / 0s")

	// Album art
	m.art = &canvas.Image{FillMode: canvas.ImageFillContain}
	m.art.SetMinSize(fyne.NewSize(220, 220))

	// Progress bar (0..1)
	m.progress = widget.NewProgressBar()

	// Transport buttons
	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		m.switchTrack(m.currentIdx - 1)
	})
	seekBackBtn := widget.NewButton("-5s", func() {
		speaker.Lock()
		newPos := max(m.audio.streamer.Position()-m.audio.sampleRate.N(time.Second*5), 0)
		_ = m.audio.streamer.Seek(newPos)
		speaker.Unlock()
	})
	m.playBtn = widget.NewButtonWithIcon("Pause", theme.MediaPauseIcon(), func() {
		speaker.Lock()
		m.audio.ctrl.Paused = !m.audio.ctrl.Paused
		paused := m.audio.ctrl.Paused
		speaker.Unlock()
		if paused {
			m.playBtn.SetIcon(theme.MediaPlayIcon())
			m.playBtn.SetText("Play")
		} else {
			m.playBtn.SetIcon(theme.MediaPauseIcon())
			m.playBtn.SetText("Pause")
		}
	})
	seekFwdBtn := widget.NewButton("+5s", func() {
		speaker.Lock()
		newPos := min(m.audio.streamer.Position()+m.audio.sampleRate.N(time.Second*5), m.audio.streamer.Len()-1)
		_ = m.audio.streamer.Seek(newPos)
		speaker.Unlock()
	})
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		if idx := nextTrackIndex(m.currentIdx, len(m.playlist), m.shuffle); idx >= 0 {
			m.switchTrack(idx)
		}
	})
	m.shuffleBtn = widget.NewButton("Shuffle: OFF", func() {
		m.shuffle = !m.shuffle
		if m.shuffle {
			m.shuffleBtn.SetText("Shuffle: ON")
		} else {
			m.shuffleBtn.SetText("Shuffle: OFF")
		}
	})

	// Volume buttons
	volDownBtn := widget.NewButton("Vol-", func() {
		speaker.Lock()
		m.audio.volume.Volume = max(m.audio.volume.Volume-0.1, -5.0)
		v := m.audio.volume.Volume
		speaker.Unlock()
		m.setVol(v)
	})
	volUpBtn := widget.NewButton("Vol+", func() {
		speaker.Lock()
		m.audio.volume.Volume = min(m.audio.volume.Volume+0.1, 0.0)
		v := m.audio.volume.Volume
		speaker.Unlock()
		m.setVol(v)
	})

	// Speed buttons
	speedDownBtn := widget.NewButton("Speed-", func() {
		speaker.Lock()
		m.audio.resampler.SetRatio(max(m.audio.resampler.Ratio()*15/16, 0.25))
		r := m.audio.resampler.Ratio()
		speaker.Unlock()
		m.setSpeed(r)
	})
	speedUpBtn := widget.NewButton("Speed+", func() {
		speaker.Lock()
		m.audio.resampler.SetRatio(min(m.audio.resampler.Ratio()*16/15, 3.0))
		r := m.audio.resampler.Ratio()
		speaker.Unlock()
		m.setSpeed(r)
	})

	// Song list
	m.list = widget.NewList(
		func() int { return len(m.playlistMetadata) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			title, artist := "Unknown", "Unknown"
			if md := m.playlistMetadata[id]; md != nil {
				if md.Title != "" {
					title = md.Title
				}
				if md.Artist != "" {
					artist = md.Artist
				}
			}
			o.(*widget.Label).SetText(fmt.Sprintf("%s - %s", title, artist))
		},
	)
	m.list.OnSelected = func(id widget.ListItemID) {
		if id != m.currentIdx {
			m.switchTrack(id)
		}
	}

	// Layout
	metaBox := container.NewVBox(
		m.titleLbl,
		m.artistLbl,
		m.albumLbl,
		m.sampleLbl,
		container.NewHBox(widget.NewLabel("Volume:"), m.volLbl, volDownBtn, volUpBtn),
		container.NewHBox(widget.NewLabel("Speed:"), m.speedLbl, speedDownBtn, speedUpBtn),
	)
	top := container.NewBorder(nil, nil, m.art, nil, metaBox)
	transport := container.NewHBox(prevBtn, seekBackBtn, m.playBtn, seekFwdBtn, nextBtn, m.shuffleBtn)
	bottom := container.NewVBox(m.progress, m.timeLbl, transport)
	leftPane := container.NewBorder(top, bottom, nil, nil)
	content := container.NewHSplit(leftPane, m.list)
	content.Offset = 0.65
	win.SetContent(content)
	win.Resize(fyne.NewSize(760, 420))

	// Initial state (audio is already playing from main).
	m.refreshMeta()
	m.refreshArt()
	m.setVol(m.audio.volume.Volume)
	m.setSpeed(m.audio.resampler.Ratio())
	m.list.Select(m.currentIdx)

	// Ticker: update progress + time every 100ms.
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			ap := m.audio
			if ap == nil {
				continue
			}
			speaker.Lock()
			posIdx := ap.streamer.Position()
			lenIdx := ap.streamer.Len()
			sr := ap.sampleRate
			speaker.Unlock()

			// Auto-advance when the current track has drained.
			if lenIdx > 0 && posIdx >= lenIdx && !m.loading {
				if idx := nextTrackIndex(m.currentIdx, len(m.playlist), m.shuffle); idx >= 0 {
					m.switchTrack(idx)
				}
				continue
			}

			frac := 0.0
			if lenIdx > 0 {
				frac = float64(posIdx) / float64(lenIdx)
			}
			frac = min(max(frac, 0), 1)
			txt := fmt.Sprintf(" %s / %s", sr.D(posIdx).Round(time.Second), sr.D(lenIdx).Round(time.Second))

			fyne.Do(func() {
				m.progress.SetValue(frac)
				m.timeLbl.SetText(txt)
			})
		}
	}()
	win.SetOnClosed(func() { ticker.Stop() })

	win.ShowAndRun()
}

// switchTrack loads playlist[idx] off the UI thread, then swaps it in.
func (m *guiUI) switchTrack(idx int) {
	if m.loading {
		return
	}
	if idx < 0 || idx >= len(m.playlist) {
		return
	}
	m.loading = true
	go func() {
		ap, md, err := loadTrack(m.playlist[idx])
		fyne.DoAndWait(func() {
			m.loading = false
			if err != nil {
				m.timeLbl.SetText(fmt.Sprintf(" load error: %v", err))
				return
			}
			speaker.Clear()
			if s := m.audio.streamer; s != nil {
				s.Close()
			}
			m.audio, m.metadata, m.currentIdx = ap, md, idx
			m.refreshMeta()
			m.refreshArt()
			m.setVol(m.audio.volume.Volume)
			m.setSpeed(m.audio.resampler.Ratio())
			m.list.Select(idx)
			m.audio.Play()
		})
	}()
}

func (m *guiUI) refreshMeta() {
	title, artist, album := "Unknown Title", "Unknown Artist", "Unknown Album"
	sampleRateStr := "NaN Hz"
	if m.metadata != nil {
		if m.metadata.Title != "" {
			title = m.metadata.Title
		}
		if m.metadata.Artist != "" {
			artist = m.metadata.Artist
		}
		if m.metadata.Album != "" {
			album = m.metadata.Album
		}
		if m.metadata.SampleRate > 0 {
			sampleRateStr = fmt.Sprintf("%d Hz", m.metadata.SampleRate)
		}
	}
	m.titleLbl.SetText(title)
	m.artistLbl.SetText(artist)
	m.albumLbl.SetText(album)
	m.sampleLbl.SetText(sampleRateStr)
}

func (m *guiUI) refreshArt() {
	if m.metadata == nil || len(m.metadata.AlbumArt) == 0 {
		m.art.Image = nil
		m.art.Refresh()
		return
	}
	img, _, err := image.Decode(bytes.NewReader(m.metadata.AlbumArt))
	if err != nil {
		m.art.Image = nil
		m.art.Refresh()
		return
	}
	m.art.Image = img
	m.art.Refresh()
}

func (m *guiUI) setVol(v float64) {
	pct := int((v + 5.0) / 5.0 * 100)
	if pct < 0 {
		pct = 0
	}
	m.volLbl.SetText(fmt.Sprintf("%d%%", pct))
}

func (m *guiUI) setSpeed(r float64) {
	m.speedLbl.SetText(fmt.Sprintf("%.2fx", r))
}
