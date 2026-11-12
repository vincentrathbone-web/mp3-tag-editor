package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"strings"

	// Image formats the cover art can be made from.
	_ "image/gif"
	_ "image/png"

	_ "github.com/gen2brain/avif"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	xdraw "golang.org/x/image/draw"
)

const (
	coverMaxSide = 500 // covers are saved at most 500×500
	coverQuality = 90
	thumbSide    = 64
	maxPixels    = 100_000_000 // refuse absurdly large images instead of running out of memory
)

// Cover describes a picture for the UI. DataURL is only filled in when the full image is needed.
type Cover struct {
	DataURL string       `json:"dataUrl,omitempty"`
	Thumb   string       `json:"thumb"`
	Mime    string       `json:"mime"`
	Width   int          `json:"width"`
	Height  int          `json:"height"`
	Bytes   int          `json:"bytes"`
	Source  *CoverSource `json:"source,omitempty"`
}

// CoverSource describes the image the user picked, before conversion.
type CoverSource struct {
	Format string `json:"format"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int    `json:"bytes"`
}

// convertCover turns any supported image into a baseline JPEG no larger than 500×500.
func convertCover(data []byte, name string) (*Cover, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("“%s” isn't an image this app can read. Use AVIF, WebP, PNG, JPG, GIF, BMP or TIFF", name)
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, fmt.Errorf("“%s” is too large (%d × %d pixels)", name, cfg.Width, cfg.Height)
	}
	src := &CoverSource{Format: strings.ToUpper(format), Width: cfg.Width, Height: cfg.Height, Bytes: len(data)}
	if src.Format == "JPEG" {
		src.Format = "JPG"
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("couldn't read “%s”: %v", name, err)
	}

	var out []byte
	w, h := cfg.Width, cfg.Height
	if format == "jpeg" && w <= coverMaxSide && h <= coverMaxSide && isBaselineJPEG(data) {
		out = data // already a small, compatible JPEG: keep it exactly as it is
	} else {
		w, h = fitInside(w, h, coverMaxSide)
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		// JPEG has no transparency, so transparent areas become white.
		xdraw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, xdraw.Src)
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Over, nil)
		var buf bytes.Buffer
		// Go's encoder always writes baseline (not progressive) JPEG, which every player can show.
		if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: coverQuality}); err != nil {
			return nil, err
		}
		out = buf.Bytes()
	}
	return &Cover{
		DataURL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(out),
		Thumb:   makeThumb(img),
		Mime:    "image/jpeg",
		Width:   w, Height: h, Bytes: len(out),
		Source: src,
	}, nil
}

// describePicture builds a Cover for a picture already inside an MP3.
func describePicture(p pictureInfo, full bool) *Cover {
	c := &Cover{Mime: p.Mime, Bytes: len(p.Data)}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(p.Data)); err == nil {
		c.Width, c.Height = cfg.Width, cfg.Height
		if cfg.Width*cfg.Height <= maxPixels {
			if img, _, err := image.Decode(bytes.NewReader(p.Data)); err == nil {
				c.Thumb = makeThumb(img)
			}
		}
	}
	if full {
		c.DataURL = "data:" + p.Mime + ";base64," + base64.StdEncoding.EncodeToString(p.Data)
	}
	return c
}

func makeThumb(img image.Image) string {
	b := img.Bounds()
	w, h := fitInside(b.Dx(), b.Dy(), thumbSide)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, xdraw.Src)
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	var buf bytes.Buffer
	if jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}) != nil {
		return ""
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func fitInside(w, h, max int) (int, int) {
	if w <= max && h <= max {
		return w, h
	}
	if w >= h {
		return max, maxInt(1, h*max/w)
	}
	return maxInt(1, w*max/h), max
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// isBaselineJPEG reports whether the JPEG uses baseline encoding (SOF0/SOF1),
// which older players and car stereos need. Progressive JPEGs (SOF2) return false.
func isBaselineJPEG(b []byte) bool {
	for i := 2; i+4 <= len(b); {
		if b[i] != 0xFF {
			return false
		}
		m := b[i+1]
		switch {
		case m == 0xC0 || m == 0xC1:
			return true
		case m >= 0xC2 && m <= 0xCF && m != 0xC4 && m != 0xC8 && m != 0xCC:
			return false
		case m == 0xFF: // fill byte
			i++
			continue
		case m == 0xD8 || (m >= 0xD0 && m <= 0xD7) || m == 0x01:
			i += 2
			continue
		}
		i += 2 + int(b[i+2])<<8 + int(b[i+3])
	}
	return false
}

// decodeDataURL accepts "data:...;base64,XXXX" or plain base64.
func decodeDataURL(s string) ([]byte, error) {
	if i := strings.Index(s, ","); strings.HasPrefix(s, "data:") && i >= 0 {
		s = s[i+1:]
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("the image data was damaged on the way")
	}
	return b, nil
}
