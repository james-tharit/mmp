package kitty

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKittyArt(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 100, 100)))
	t.Cleanup(func() { // kitty normally deletes the t=t file
		fs, _ := filepath.Glob(filepath.Join(os.TempDir(), "mmp-tty-graphics-protocol-*.png"))
		for _, f := range fs {
			os.Remove(f)
		}
	})
	ph, up := Art(b.Bytes(), 24)
	if lines := strings.Count(ph, "\n"); lines != 12 {
		t.Errorf("rows = %d, want 12", lines)
	}
	if !strings.HasPrefix(up, "\x1b_Ga=T,U=1,t=t,f=100") || !strings.HasSuffix(up, "\x1b\\") {
		t.Errorf("bad upload sequence: %.40q", up)
	}
}
