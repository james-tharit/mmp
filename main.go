package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	goflac "github.com/go-flac/go-flac/v2"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

type audioPanel struct {
	sampleRate beep.SampleRate
	streamer   beep.StreamSeeker
	ctrl       *beep.Ctrl
	resampler  *beep.Resampler
	volume     *effects.Volume
}

// FLACMetadata holds extracted FLAC metadata
type FLACMetadata struct {
	Title        string
	Artist       string
	Album        string
	Date         string
	Duration     uint64
	SampleRate   uint32
	Channels     uint8
	BitDepth     uint8
	TotalSamples uint64
}

func newAudioPanel(sampleRate beep.SampleRate, streamer beep.StreamSeeker) (*audioPanel, error) {
	loopStreamer, err := beep.Loop2(streamer)
	if err != nil {
		return nil, err
	}

	ctrl := &beep.Ctrl{Streamer: loopStreamer}
	resampler := beep.ResampleRatio(4, 1, ctrl)
	volume := &effects.Volume{Streamer: resampler, Base: 2}
	return &audioPanel{sampleRate, streamer, ctrl, resampler, volume}, nil
}

func (ap *audioPanel) play() {
	speaker.Play(ap.volume)
}

// getMetadata extracts FLAC metadata from a file
func getMetadata(filePath string) (*FLACMetadata, error) {
	// Parse FLAC file using go-flac library
	file, err := goflac.ParseFile(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metadata := &FLACMetadata{}

	// Extract StreamInfo metadata
	si, err := file.GetStreamInfo()
	if err == nil && si != nil {
		metadata.SampleRate = uint32(si.SampleRate)
		metadata.Channels = uint8(si.ChannelCount)
		metadata.BitDepth = uint8(si.BitDepth)
		metadata.TotalSamples = uint64(si.SampleCount)
		// Calculate duration in seconds
		if si.SampleRate > 0 {
			metadata.Duration = uint64(si.SampleCount) / uint64(si.SampleRate)
		}
	}

	// Parse metadata blocks for vorbis comments and other tags
	for _, block := range file.Meta {
		// BlockType 4 is VorbisComment
		if block.Type == 4 && len(block.Data) > 4 {
			// Simple VorbisComment parsing (basic implementation)
			// Format: 4 bytes vendor string length, then vendor string, then comment count
			vendorLen := int(block.Data[0]) | int(block.Data[1])<<8 | int(block.Data[2])<<16 | int(block.Data[3])<<24
			pos := 4 + vendorLen

			if pos+4 <= len(block.Data) {
				commentCount := int(block.Data[pos]) | int(block.Data[pos+1])<<8 | int(block.Data[pos+2])<<16 | int(block.Data[pos+3])<<24
				pos += 4

				// Parse comments
				for i := 0; i < commentCount && pos < len(block.Data); i++ {
					if pos+4 > len(block.Data) {
						break
					}
					commentLen := int(block.Data[pos]) | int(block.Data[pos+1])<<8 | int(block.Data[pos+2])<<16 | int(block.Data[pos+3])<<24
					pos += 4

					if pos+commentLen > len(block.Data) {
						break
					}

					comment := string(block.Data[pos : pos+commentLen])
					pos += commentLen

					// Parse key=value format
					for j := 0; j < len(comment); j++ {
						if comment[j] == '=' {
							key := comment[:j]
							value := comment[j+1:]

							switch key {
							case "TITLE":
								metadata.Title = value
							case "ARTIST":
								metadata.Artist = value
							case "ALBUM":
								metadata.Album = value
							case "DATE":
								metadata.Date = value
							}
							break
						}
					}
				}
			}
		}
	}

	return metadata, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s song.flac\n", os.Args[0])
		os.Exit(1)
	}

	// Open and decode the audio file
	f, err := os.Open(os.Args[1])
	if err != nil {
		report(err)
	}
	defer f.Close()

	streamer, format, err := flac.Decode(f)
	if err != nil {
		report(err)
	}
	defer streamer.Close()

	// Initialize speaker
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/30))
	defer speaker.Close()

	// Create audio panel with the beep engine
	ap, err := newAudioPanel(format.SampleRate, streamer)
	if err != nil {
		report(err)
	}

	ap.play()

	// Get metadata
	metadata, err := getMetadata(os.Args[1])
	if err != nil {
		report(err)
	}
	fmt.Printf("Title: %s\nArtist: %s\nAlbum: %s\nDate: %s\nDuration: %d seconds\nSample Rate: %d Hz\nChannels: %d\nBit Depth: %d\n",
		metadata.Title, metadata.Artist, metadata.Album, metadata.Date, metadata.Duration, metadata.SampleRate, metadata.Channels, metadata.BitDepth)

	// Create and run BubbleTea app
	model := NewUIModel(os.Args[1], ap)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		report(err)
	}
}

func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
