package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/image/draw"

	"mmp/internal/kitty"
)

// refreshArt rebuilds the cached cover. On kitty-protocol terminals it also
// returns the command that uploads the full-resolution image.
func (m *Model) refreshArt() tea.Cmd {
	m.art = ""
	art := m.albums[m.selAlbum].Tracks[0].Meta.AlbumArt
	if len(art) == 0 {
		return nil
	}

	width := artCols
	if m.width < artCols {
		width = 6
	}

	if kitty.Supported() {
		if placeholders, upload := kitty.Art(art, width); placeholders != "" {
			m.art = placeholders
			return func() tea.Msg {
				os.Stdout.WriteString(upload)
				return nil
			}
		}
	}
	m.art = halfBlockArt(art, width)
	return nil
}

// halfBlockArt renders an image as 24-bit ANSI, two pixel rows per terminal row.
func halfBlockArt(data []byte, width int) string {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil || width <= 0 {
		return ""
	}
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return ""
	}

	height := max(1, width*b.Dy()/b.Dx())
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)

	var sb strings.Builder
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			top := dst.RGBAAt(x, y)
			bottom := color.RGBA{}
			if y+1 < height {
				bottom = dst.RGBAAt(x, y+1)
			}
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀",
				top.R, top.G, top.B, bottom.R, bottom.G, bottom.B)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return sb.String()
}
