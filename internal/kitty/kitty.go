package kitty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/draw"
)

// Supported reports whether the terminal speaks the kitty graphics protocol
// with unicode placeholders (kitty, ghostty). tmux/screen swallow the APC and ssh has no shared filesystem.
func Supported() bool {
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return false
	}
	return os.Getenv("KITTY_WINDOW_ID") != "" ||
		os.Getenv("TERM") == "xterm-kitty" ||
		os.Getenv("TERM_PROGRAM") == "ghostty"
}

// First entries of kitty's rowcolumn-diacritics table; index = row/column number.
var kittyDiacritics = []rune{
	0x305, 0x30D, 0x30E, 0x310, 0x312, 0x33D, 0x33E, 0x33F, 0x346, 0x34A,
	0x34B, 0x34C, 0x350, 0x351, 0x352, 0x357, 0x35B, 0x363, 0x364, 0x365,
	0x366, 0x367, 0x368, 0x369, 0x36A, 0x36B, 0x36C, 0x36D, 0x36E, 0x36F,
	0x483, 0x484, 0x485, 0x486, 0x487, 0x592, 0x593, 0x594, 0x595, 0x597,
}

const kittyImageID = 0x9867C5 // doubles as the placeholder foreground color (24-bit id)

// Art decodes the cover, sizes it to `cols` cells wide, and returns the
// placeholder text block plus the escape sequence that uploads the image.
func Art(artBytes []byte, cols int) (placeholders, upload string) {
	img, _, err := image.Decode(bytes.NewReader(artBytes))
	if err != nil || cols <= 0 {
		return "", ""
	}
	cols = min(cols, len(kittyDiacritics))
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return "", ""
	}
	// cells are ~1:2, so rows = cols * h / w / 2
	rows := max(1, min((cols*b.Dy()+b.Dx())/(2*b.Dx()), len(kittyDiacritics)))

	// Downscale big covers: the terminal only needs ~16px/cell horizontally.
	w := min(b.Dx(), 800)
	dst := image.NewRGBA(image.Rect(0, 0, w, w*b.Dy()/b.Dx()))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	var buf bytes.Buffer
	if png.Encode(&buf, dst) != nil {
		return "", ""
	}

	// Hand the PNG over by temp file (t=t: terminal reads then deletes it) so the
	// escape sequence is tiny and a single write can't be torn by a repaint.
	// The "tty-graphics-protocol" name part is required by kitty for t=t.
	f, err := os.CreateTemp("", "mmp-tty-graphics-protocol-*.png")
	if err != nil {
		return "", ""
	}
	_, werr := f.Write(buf.Bytes())
	if f.Close() != nil || werr != nil {
		os.Remove(f.Name())
		return "", ""
	}
	// a=T transmit+display, U=1 virtual placement, f=100 PNG, q=2 no replies.
	up := fmt.Sprintf("\x1b_Ga=T,U=1,t=t,f=100,q=2,i=%d,c=%d,r=%d;%s\x1b\\",
		kittyImageID, cols, rows, base64.StdEncoding.EncodeToString([]byte(f.Name())))

	var ph strings.Builder
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", kittyImageID>>16&255, kittyImageID>>8&255, kittyImageID&255)
	for r := 0; r < rows; r++ {
		ph.WriteString(fg)
		for c := 0; c < cols; c++ {
			ph.WriteRune(0x10EEEE)
			ph.WriteRune(kittyDiacritics[r])
			ph.WriteRune(kittyDiacritics[c])
		}
		ph.WriteString("\x1b[39m\n")
	}
	return ph.String(), up
}
