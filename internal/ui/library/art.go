package library

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/Mercwri/innovade/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// ArtBaseURL is the official card image CDN, exported so other packages
// (e.g. the deckbuilder's PNG export) can fetch missing art without
// duplicating the HTTP/caching logic here.

const ArtBaseURL = "https://www.gundam-gcg.com/en/images/cards/card/"

// artDims computes the character dimensions of the art frame.
//
// A standard trading card is 63×88 mm. In half-block terminal coordinates
// each char-cell represents 2 square sub-pixels, giving H/W ≈ 0.698.
// We fill the available width (frameW) and derive height from the ratio.
// If the resulting height exceeds maxH we scale both axes down proportionally
// so the card shape is preserved.
func artDims(frameW, maxH int) (artW, artH int) {
	artW = frameW
	artH = max(1, int(float64(artW)*88/63/2+0.5))
	if maxH > 0 && artH > maxH {
		artH = maxH
		artW = max(1, int(float64(artH)*63/88*2+0.5))
	}
	return
}

// artCache stores decoded+resized images keyed by "path:WxH".
// Only successful loads are stored; misses are not cached so a download
// arriving later will be picked up on the next render.
var artCache sync.Map

// renderArtFrame returns the card art as a string of half-block Unicode
// characters (▀) with 24-bit ANSI colour codes, ready to embed in the
// detail panel. Falls back to a placeholder box when the image is absent.
//
// frameW is the inner content width; maxH caps the character height so the
// card text below remains visible (0 = no cap).
func renderArtFrame(imagePath string, frameW, maxH int) string {
	artW, artH := artDims(frameW, maxH)

	if imagePath == "" {
		return artPlaceholder(artW, artH)
	}

	img := resizedArt(imagePath, artW, artH*2) // *2: half-blocks = 2 sub-pixel rows per char
	if img == nil {
		return artPlaceholder(artW, artH)
	}

	return halfBlocks(img, artW, artH)
}

// resizedArt decodes and scales the image at path to (w × h) pixels.
// Returns nil if the file doesn't exist or can't be decoded.
func resizedArt(path string, w, h int) image.Image {
	key := fmt.Sprintf("%s:%dx%d", path, w, h)
	if v, ok := artCache.Load(key); ok {
		return v.(image.Image)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil // not cached — will retry on next render
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil
	}

	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	artCache.Store(key, dst)
	return dst
}

// halfBlocks converts img (w × h*2 pixels) to h rows of half-block chars.
func halfBlocks(img image.Image, charW, charH int) string {
	var sb strings.Builder
	for row := 0; row < charH; row++ {
		y0, y1 := row*2, row*2+1
		for x := 0; x < charW; x++ {
			tr, tg, tb, _ := img.At(x, y0).RGBA()
			br, bg, bb, _ := img.At(x, y1).RGBA()
			// RGBA() returns 16-bit; shift right by 8 to get 8-bit values.
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				tr>>8, tg>>8, tb>>8,
				br>>8, bg>>8, bb>>8)
		}
		sb.WriteString("\x1b[0m") // reset colours at end of each row
		if row < charH-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// artPlaceholder returns a bordered box the same size as the art frame.
func artPlaceholder(charW, charH int) string {
	// Width/Height() include padding but not the border itself (1 char each side).
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder).
		Width(charW-2).
		Height(charH-2).
		Align(lipgloss.Center, lipgloss.Center).
		Background(styles.BgBase).
		Foreground(styles.ColorBorder).
		Render("no image")
}

// revisionSuffix strips the "-r<timestamp>" cache-busting suffix that some
// imported card records carry (e.g. "GD05-111-r1783690232899.webp") — the CDN
// itself only ever hosts the unsuffixed name.
var revisionSuffix = regexp.MustCompile(`-r\d+$`)

// upstreamFilename maps a locally recorded image filename to the name the CDN
// actually serves. Besides the revision suffix above, a handful of entries
// (e.g. token cards) record a stale ".png" extension even though the CDN only
// hosts ".webp" for every card image; image.Decode sniffs content rather than
// relying on the on-disk extension, so normalizing here is safe.
func upstreamFilename(filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	base = revisionSuffix.ReplaceAllString(base, "")
	return base + ".webp"
}

// DownloadArt fetches filename from the official card image CDN and writes it
// atomically to localPath. The images directory is created if needed.
func DownloadArt(localPath, filename string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	resp, err := http.Get(ArtBaseURL + upstreamFilename(filename))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, filename)
	}

	// Write to a temp file then rename so we never leave a partial file.
	tmp, err := os.CreateTemp(filepath.Dir(localPath), "art-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, localPath)
}
