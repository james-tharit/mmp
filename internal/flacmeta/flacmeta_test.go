package flacmeta

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func le32(n int) []byte { return binary.LittleEndian.AppendUint32(nil, uint32(n)) }
func be32(n int) []byte { return binary.BigEndian.AppendUint32(nil, uint32(n)) }

func TestParseVorbis(t *testing.T) {
	var b bytes.Buffer
	b.Write(le32(3))
	b.WriteString("vnd")
	comments := []string{"title=Song", "ARTIST=Band", "NOEQUALS", "Album=LP"}
	b.Write(le32(len(comments)))
	for _, c := range comments {
		b.Write(le32(len(c)))
		b.WriteString(c)
	}

	md := &Metadata{}
	md.parseVorbis(b.Bytes())
	if md.Title != "Song" || md.Artist != "Band" || md.Album != "LP" {
		t.Fatalf("got %+v", md)
	}

	md = &Metadata{}
	md.parseVorbis(b.Bytes()[:b.Len()-3]) // truncated: keep what parsed, no panic
	if md.Title != "Song" || md.Artist != "Band" {
		t.Fatalf("truncated: got %+v", md)
	}
}

func picture(kind int, img string) []byte {
	var b bytes.Buffer
	b.Write(be32(kind))
	b.Write(be32(len("image/png")))
	b.WriteString("image/png")
	b.Write(be32(0))
	b.Write(make([]byte, 16))
	b.Write(be32(len(img)))
	b.WriteString(img)
	return b.Bytes()
}

func TestParsePicturePrefersFrontCover(t *testing.T) {
	md := &Metadata{}
	md.parsePicture(picture(0, "other"))
	md.parsePicture(picture(pictureFrontCover, "front"))
	md.parsePicture(picture(0, "later"))
	if string(md.AlbumArt) != "front" || md.AlbumArtMime != "image/png" {
		t.Fatalf("got %q %q", md.AlbumArt, md.AlbumArtMime)
	}

	md = &Metadata{}
	md.parsePicture(picture(0, "img")[:30]) // truncated
	if md.AlbumArt != nil {
		t.Fatal("truncated picture must be ignored")
	}
}
