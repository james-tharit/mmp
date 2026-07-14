package main

import (
	"strings"

	"github.com/go-flac/go-flac/v2"
)

// extracted FLAC metadata for display in the UI
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
	AlbumArt     []byte // Raw image bytes
	AlbumArtMime string // e.g., "image/jpeg" or "image/png"
}

// Helper function to check for .flac extension
func isFlac(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".flac")
}

func getAllMetadata(filePaths []string) ([]*FLACMetadata, error) {
	var metadataList []*FLACMetadata
	for _, filePath := range filePaths {
		if isFlac(filePath) {
			metadata, err := getMetadata(filePath)
			if err != nil {
				return nil, err
			}
			metadataList = append(metadataList, metadata)
		}
	}
	return metadataList, nil
}

// getMetadata extracts FLAC metadata from a file
func getMetadata(filePath string) (*FLACMetadata, error) {
	// Parse FLAC file using go-flac library
	file, err := flac.ParseFile(filePath)
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
							// Vorbis comment field names are case-insensitive per spec.
							key := strings.ToUpper(comment[:j])
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

		// Inside: for _, block := range file.Meta {

		// BlockType 6 is PICTURE
		if block.Type == 6 && len(block.Data) > 32 {
			pos := 0

			// 1. Picture Type (4 bytes) - Value 3 is usually "Front Cover"
			pictureType := int(block.Data[pos])<<24 | int(block.Data[pos+1])<<16 | int(block.Data[pos+2])<<8 | int(block.Data[pos+3])
			pos += 4

			// 2. MIME type length (4 bytes)
			mimeLen := int(block.Data[pos])<<24 | int(block.Data[pos+1])<<16 | int(block.Data[pos+2])<<8 | int(block.Data[pos+3])
			pos += 4

			if pos+mimeLen > len(block.Data) {
				continue
			}
			mimeType := string(block.Data[pos : pos+mimeLen])
			pos += mimeLen

			// 3. Description length (4 bytes)
			descLen := int(block.Data[pos])<<24 | int(block.Data[pos+1])<<16 | int(block.Data[pos+2])<<8 | int(block.Data[pos+3])
			pos += 4

			if pos+descLen > len(block.Data) {
				continue
			}
			// Skip the description string, we don't strictly need it
			pos += descLen

			// 4. Dimensions & Color depth (16 bytes total: Width, Height, Depth, Indexed Colors)
			pos += 16

			// 5. Picture data length (4 bytes)
			if pos+4 > len(block.Data) {
				continue
			}
			dataLen := int(block.Data[pos])<<24 | int(block.Data[pos+1])<<16 | int(block.Data[pos+2])<<8 | int(block.Data[pos+3])
			pos += 4

			if pos+dataLen <= len(block.Data) {
				// Extract the raw image bytes
				imgData := make([]byte, dataLen)
				copy(imgData, block.Data[pos:pos+dataLen])

				// Prioritize Front Cover (Type 3), or take the first available image if not set yet
				if pictureType == 3 || metadata.AlbumArt == nil {
					metadata.AlbumArt = imgData
					metadata.AlbumArtMime = mimeType
				}
			}
		}
	}

	return metadata, nil
}
