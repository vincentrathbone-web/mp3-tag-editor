package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"image/png"

	"github.com/gen2brain/avif"
)

func TestAVIFCover(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 800))
	for x := 0; x < 800; x++ {
		for y := 0; y < 800; y++ {
			img.Set(x, y, color.RGBA{uint8(x / 4), 120, uint8(y / 4), 255})
		}
	}
	var buf bytes.Buffer
	if err := avif.Encode(&buf, img, avif.Options{Quality: 60, Speed: 10}); err != nil {
		t.Fatal(err)
	}
	c, err := convertCover(buf.Bytes(), "art.avif")
	if err != nil {
		t.Fatal(err)
	}
	if c.Source.Format != "AVIF" || c.Width != 500 || c.Height != 500 {
		t.Fatalf("got %s %dx%d", c.Source.Format, c.Width, c.Height)
	}
}

func TestSmallBaselineJPEGKept(t *testing.T) {
	c1, err := convertCover(mustPNG(t, 300, 300), "small.png")
	if err != nil {
		t.Fatal(err)
	}
	jpg, _ := decodeDataURL(c1.DataURL)
	c2, err := convertCover(jpg, "small.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if c2.DataURL != c1.DataURL {
		t.Fatal("a small baseline JPEG should be kept byte for byte")
	}
}

func TestNotAnImage(t *testing.T) {
	if _, err := convertCover([]byte("hello"), "notes.txt"); err == nil {
		t.Fatal("expected an error")
	}
}

func mustPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
