package flacmeta

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/go-flac/go-flac/v2"
)

const (
	blockVorbisComment = 4
	blockPicture       = 6
	pictureFrontCover  = 3
)

type Metadata struct {
	Title        string
	Artist       string
	Album        string
	Date         string
	Duration     uint64
	SampleRate   uint32
	Channels     uint8
	BitDepth     uint8
	TotalSamples uint64
	AlbumArt     []byte
	AlbumArtMime string
}

func IsFlac(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".flac")
}

func ReadAll(paths []string) ([]*Metadata, error) {
	list := make([]*Metadata, 0, len(paths))
	for _, p := range paths {
		md, err := Read(p)
		if err != nil {
			return nil, err
		}
		list = append(list, md)
	}
	return list, nil
}

func Read(path string) (*Metadata, error) {
	file, err := flac.ParseFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	si, err := file.GetStreamInfo()
	if err != nil {
		return nil, err
	}
	md := &Metadata{
		SampleRate:   uint32(si.SampleRate),
		Channels:     uint8(si.ChannelCount),
		BitDepth:     uint8(si.BitDepth),
		TotalSamples: uint64(si.SampleCount),
	}
	if si.SampleRate > 0 {
		md.Duration = uint64(si.SampleCount) / uint64(si.SampleRate)
	}

	for _, block := range file.Meta {
		switch block.Type {
		case blockVorbisComment:
			md.parseVorbis(block.Data)
		case blockPicture:
			md.parsePicture(block.Data)
		}
	}
	return md, nil
}

// cursor reads length-prefixed fields; any overrun latches err and yields zero values.
type cursor struct {
	b   []byte
	err bool
}

func (c *cursor) take(n int) []byte {
	if c.err || n < 0 || n > len(c.b) {
		c.err = true
		return nil
	}
	out := c.b[:n]
	c.b = c.b[n:]
	return out
}

func (c *cursor) u32(order binary.ByteOrder) int {
	b := c.take(4)
	if b == nil {
		return 0
	}
	return int(order.Uint32(b))
}

func (md *Metadata) parseVorbis(data []byte) {
	c := &cursor{b: data}
	c.take(c.u32(binary.LittleEndian)) // vendor string
	n := c.u32(binary.LittleEndian)

	fields := map[string]*string{
		"TITLE":  &md.Title,
		"ARTIST": &md.Artist,
		"ALBUM":  &md.Album,
		"DATE":   &md.Date,
	}
	for i := 0; i < n; i++ {
		comment := c.take(c.u32(binary.LittleEndian))
		if c.err {
			return
		}
		key, value, ok := strings.Cut(string(comment), "=")
		if dst := fields[strings.ToUpper(key)]; ok && dst != nil {
			*dst = value
		}
	}
}

func (md *Metadata) parsePicture(data []byte) {
	c := &cursor{b: data}
	pictureType := c.u32(binary.BigEndian)
	mime := string(c.take(c.u32(binary.BigEndian)))
	c.take(c.u32(binary.BigEndian)) // description
	c.take(16)                      // width, height, depth, indexed colors
	img := c.take(c.u32(binary.BigEndian))
	if c.err {
		return
	}
	if pictureType == pictureFrontCover || md.AlbumArt == nil {
		md.AlbumArt = bytes.Clone(img)
		md.AlbumArtMime = mime
	}
}
