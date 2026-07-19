package main

import (
	"math"
	"testing"
)

func TestFFTFindsTone(t *testing.T) {
	const sr = 44100
	x := make([]complex128, fftSize)
	for i := range x {
		x[i] = complex(math.Sin(2*math.Pi*5000*float64(i)/sr), 0)
	}
	fft(x)
	best, bestMag := 0, 0.0
	for b := 0; b < fftSize/2; b++ {
		if m := cmplxAbs(x[b]); m > bestMag {
			best, bestMag = b, m
		}
	}
	if got := best * sr / fftSize; math.Abs(float64(got-5000)) > 30 {
		t.Fatalf("peak bin at %d Hz, want ~5000", got)
	}
}

func cmplxAbs(c complex128) float64 { return math.Hypot(real(c), imag(c)) }

func TestCutoffAndVerdict(t *testing.T) {
	const sr = 44100
	bins := fftSize / 2
	// Brickwalled at 16 kHz.
	peak := make([]float64, bins)
	for b := range peak {
		if b*sr/(2*bins) < 16000 {
			peak[b] = 1
		}
	}
	if c := cutoffFrom(peak, sr); c < 15800 || c > 16000 {
		t.Fatalf("cutoff %d, want ~16000", c)
	}
	if v := verdict(cutoffFrom(peak, sr), sr); v[:5] != "LOSSY" {
		t.Fatalf("got %q, want LOSSY", v)
	}

	// Energy all the way to Nyquist.
	for b := range peak {
		peak[b] = 1
	}
	if v := verdict(cutoffFrom(peak, sr), sr); v[:8] != "LOSSLESS" {
		t.Fatalf("got %q, want LOSSLESS", v)
	}

	// Hi-res 96 kHz file rolling off naturally at 41 kHz: not a codec brickwall.
	if v := verdict(41000, 96000); v[:8] != "LOSSLESS" {
		t.Fatalf("got %q, want LOSSLESS", v)
	}
}
