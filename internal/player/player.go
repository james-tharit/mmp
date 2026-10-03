package player

import (
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

const (
	minVolume, maxVolume = -5.0, 0.0
	minSpeed, maxSpeed   = 0.25, 3.0
)

// speakerRate is the hardware rate; every track resamples to it.
var speakerRate beep.SampleRate

// InitSpeaker must be called once, before the first Load.
func InitSpeaker(rate beep.SampleRate) error {
	speakerRate = rate
	return speaker.Init(rate, rate.N(100*time.Millisecond))
}

func CloseSpeaker() { speaker.Close() }

// Panel is one playing track: loop-free stream → speed → speaker rate → volume.
type Panel struct {
	rate     beep.SampleRate
	streamer beep.StreamSeekCloser
	ctrl     *beep.Ctrl
	speed    *beep.Resampler
	volume   *effects.Volume
}

type State struct {
	Position, Length time.Duration
	Volume, Speed    float64
	Paused           bool
}

func New(rate beep.SampleRate, streamer beep.StreamSeekCloser) *Panel {
	ctrl := &beep.Ctrl{Streamer: streamer}
	speed := beep.ResampleRatio(4, 1.0, ctrl)
	toSpeaker := beep.Resample(4, rate, speakerRate, speed)
	return &Panel{
		rate:     rate,
		streamer: streamer,
		ctrl:     ctrl,
		speed:    speed,
		volume:   &effects.Volume{Streamer: toSpeaker, Base: 2},
	}
}

func Load(path string) (*Panel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	streamer, format, err := flac.Decode(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return New(format.SampleRate, streamer), nil
}

func (p *Panel) Play()        { speaker.Play(p.volume) }
func (p *Panel) Close() error { return p.streamer.Close() }

func (p *Panel) Ended() bool {
	speaker.Lock()
	defer speaker.Unlock()
	return p.streamer.Position() >= p.streamer.Len()
}

func (p *Panel) TogglePause() {
	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	speaker.Unlock()
}

func (p *Panel) SeekBy(d time.Duration) {
	speaker.Lock()
	pos := p.streamer.Position() + p.rate.N(d)
	_ = p.streamer.Seek(max(0, min(pos, p.streamer.Len()-1)))
	speaker.Unlock()
}

func (p *Panel) AdjustVolume(delta float64) {
	speaker.Lock()
	p.volume.Volume = max(minVolume, min(p.volume.Volume+delta, maxVolume))
	speaker.Unlock()
}

func (p *Panel) ScaleSpeed(factor float64) {
	speaker.Lock()
	p.speed.SetRatio(max(minSpeed, min(p.speed.Ratio()*factor, maxSpeed)))
	speaker.Unlock()
}

func (p *Panel) State() State {
	speaker.Lock()
	defer speaker.Unlock()
	return State{
		Position: p.rate.D(p.streamer.Position()),
		Length:   p.rate.D(p.streamer.Len()),
		Volume:   p.volume.Volume,
		Speed:    p.speed.Ratio(),
		Paused:   p.ctrl.Paused,
	}
}

func ClearSpeaker() { speaker.Clear() }
