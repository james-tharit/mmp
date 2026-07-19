package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/cmplx"
	"os"

	"github.com/gopxl/beep/v2/flac"
)

const (
	fftSize  = 2048 // ~21 Hz bins at 44.1 kHz
	specCols = 400  // windows sampled across the track
)

// fft transforms x in place. len(x) must be a power of two.
func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j |= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for l := 2; l <= n; l <<= 1 {
		wl := cmplx.Rect(1, -2*math.Pi/float64(l))
		for i := 0; i < n; i += l {
			w := complex(1, 0)
			for k := 0; k < l/2; k++ {
				u, v := x[i+k], x[i+k+l/2]*w
				x[i+k], x[i+k+l/2] = u+v, u-v
				w *= wl
			}
		}
	}
}

// cutoffFrom returns the highest frequency (Hz) still carrying real energy,
// given per-bin peak magnitudes. ponytail: fixed -60 dB-below-peak threshold,
// good enough to separate a brickwall from a natural rolloff.
func cutoffFrom(peak []float64, sampleRate int) int {
	maxMag := 0.0
	for _, p := range peak {
		maxMag = math.Max(maxMag, p)
	}
	if maxMag == 0 {
		return 0
	}
	thresh := maxMag * 0.001 // -60 dB
	for b := len(peak) - 1; b >= 0; b-- {
		if peak[b] > thresh {
			return b * sampleRate / (2 * len(peak))
		}
	}
	return 0
}

// verdict guesses the source encoding from where the spectrum stops.
func verdict(cutoffHz, sampleRate int) string {
	nyquist := sampleRate / 2
	khz := float64(cutoffHz) / 1000
	// Lossy codecs brickwall between ~15 and ~21 kHz. Content reaching Nyquist,
	// or surviving past 22 kHz at all, is a natural rolloff — not a codec.
	if cutoffHz >= nyquist*9/10 || cutoffHz > 22000 {
		return fmt.Sprintf("LOSSLESS — content up to %.1f kHz (Nyquist %.1f kHz)", khz, float64(nyquist)/1000)
	}
	guess := "low-bitrate lossy"
	switch {
	case cutoffHz >= 19500:
		guess = "lossy, ~320 kbps"
	case cutoffHz >= 18500:
		guess = "lossy, ~256 kbps"
	case cutoffHz >= 17500:
		guess = "lossy, ~192 kbps"
	case cutoffHz >= 15500:
		guess = "lossy, ~128 kbps"
	}
	return fmt.Sprintf("LOSSY (transcode?) — brickwall at %.1f kHz, %s", khz, guess)
}

type spectrumResult struct {
	Img      *image.RGBA
	CutoffHz int
	Verdict  string
	MaxHz    int
}

// analyzeSpectrum decodes windows spread across the file and renders a
// spectrogram plus a lossy/lossless guess.
func analyzeSpectrum(path string) (*spectrumResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	streamer, format, err := flac.Decode(f)
	if err != nil {
		return nil, err
	}
	defer streamer.Close()

	total := streamer.Len()
	if total < fftSize {
		return nil, fmt.Errorf("track too short to analyze")
	}
	sr := int(format.SampleRate)

	// Columns are read strictly forward, discarding the gaps: seeking in a FLAC
	// without a seektable costs seconds, sequential decode costs milliseconds.
	cols := min(total/fftSize, specCols)
	hop := total / cols

	bins := fftSize / 2
	img := image.NewRGBA(image.Rect(0, 0, cols, bins))
	peak := make([]float64, bins)
	buf := make([][2]float64, fftSize)
	x := make([]complex128, fftSize)
	skip := make([][2]float64, 16384)

	hann := make([]float64, fftSize)
	for i := range hann {
		hann[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
	}

	for c := 0; c < cols; c++ {
		if c > 0 {
			for left := hop - fftSize; left > 0; {
				got, ok := streamer.Stream(skip[:min(left, len(skip))])
				left -= got
				if !ok {
					break
				}
			}
		}
		for n := 0; n < fftSize; {
			got, ok := streamer.Stream(buf[n:])
			n += got
			if !ok {
				for ; n < fftSize; n++ {
					buf[n] = [2]float64{}
				}
			}
		}
		for i := range buf {
			x[i] = complex((buf[i][0]+buf[i][1])/2*hann[i], 0)
		}
		fft(x)
		for b := 0; b < bins; b++ {
			mag := cmplx.Abs(x[b]) / float64(bins)
			peak[b] = math.Max(peak[b], mag)
			db := 20 * math.Log10(mag+1e-12)
			r, g, bl := heat((db + 100) / 100)
			// Write straight into the pixel buffer; img.Set is ~5x slower.
			o := (bins-1-b)*img.Stride + c*4
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, bl, 255
		}
	}

	cutoff := cutoffFrom(peak, sr)
	// Mark the cutoff with a horizontal line.
	if row := bins - 1 - cutoff*2*bins/sr; row >= 0 && row < bins {
		for c := 0; c < cols-1; c += 6 {
			img.Set(c, row, color.RGBA{0, 255, 255, 255})
			img.Set(c+1, row, color.RGBA{0, 255, 255, 255})
		}
	}
	return &spectrumResult{Img: img, CutoffHz: cutoff, Verdict: verdict(cutoff, sr), MaxHz: sr / 2}, nil
}

var heatStops = [][3]float64{{0, 0, 8}, {50, 10, 90}, {180, 35, 70}, {250, 190, 60}, {255, 255, 230}}

// heat maps 0..1 onto a dark→purple→red→yellow→white ramp.
func heat(t float64) (uint8, uint8, uint8) {
	t = math.Min(math.Max(t, 0), 1) * float64(len(heatStops)-1)
	i := int(t)
	if i >= len(heatStops)-1 {
		i = len(heatStops) - 2
	}
	f := t - float64(i)
	a, b := heatStops[i], heatStops[i+1]
	return uint8(a[0] + (b[0]-a[0])*f), uint8(a[1] + (b[1]-a[1])*f), uint8(a[2] + (b[2]-a[2])*f)
}
