package main

import (
	"fmt"
	"time"

	vlc "github.com/adrg/libvlc-go/v3"
)

// VLCPlayer wraps libvlc functionality and keeps VLC logic isolated
type VLCPlayer struct {
	player *vlc.Player
}

// NewVLCPlayer initializes and returns a new VLC player instance
func NewVLCPlayer() (*VLCPlayer, error) {
	if err := vlc.Init("--no-video", "--quiet"); err != nil {
		return nil, fmt.Errorf("VLC init failed: %w", err)
	}

	player, err := vlc.NewPlayer()
	if err != nil {
		vlc.Release()
		return nil, fmt.Errorf("player creation failed: %w", err)
	}

	return &VLCPlayer{player: player}, nil
}

// LoadTrack loads a media file and returns its metadata
func (vp *VLCPlayer) LoadTrack(path string) (TrackInfo, error) {
	media, err := vp.player.LoadMediaFromPath(path)
	if err != nil {
		return TrackInfo{}, fmt.Errorf("failed to load media: %w", err)
	}

	return extractMetadata(media), nil
}

// Play starts playback
func (vp *VLCPlayer) Play() error {
	return vp.player.Play()
}

// Pause pauses playback
func (vp *VLCPlayer) Pause() error {
	return vp.player.SetPause(true)
}

// Resume resumes playback
func (vp *VLCPlayer) Resume() error {
	return vp.player.SetPause(false)
}

// Stop stops playback
func (vp *VLCPlayer) Stop() error {
	return vp.player.Stop()
}

// Release cleans up VLC resources
func (vp *VLCPlayer) Release() {
	vp.player.Release()
	vlc.Release()
}

// extractMetadata extracts metadata from a VLC media object
func extractMetadata(media *vlc.Media) TrackInfo {
	// Let VLC parse the local media file
	err := media.ParseWithOptions(0, vlc.MediaParseLocal)
	if err != nil {
		fmt.Println("VLC parsing error:", err)
	}

	// Wait up to 500ms for background parsing to complete
	for i := 0; i < 10; i++ {
		state, _ := media.ParseStatus()
		if state == vlc.MediaParseDone {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Extract metadata fields
	title, _ := media.Meta(vlc.MediaTitle)
	artist, _ := media.Meta(vlc.MediaArtist)
	album, _ := media.Meta(vlc.MediaAlbum)

	// Apply sensible defaults if fields are empty
	if title == "" {
		title = "Unknown Title"
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = "Unknown Album"
	}

	return TrackInfo{
		Title:  title,
		Artist: artist,
		Album:  album,
		Format: "M4A/FLAC Engine Track",
	}
}
