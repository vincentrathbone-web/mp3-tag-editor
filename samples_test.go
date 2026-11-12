package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// TestMakeSamples writes a small library of silent example MP3s for trying the app.
// It only runs when MP3TE_SAMPLES_DIR is set:
//
//	$env:MP3TE_SAMPLES_DIR = "D:\temp\samples"; go test -run TestMakeSamples
func TestMakeSamples(t *testing.T) {
	dir := os.Getenv("MP3TE_SAMPLES_DIR")
	if dir == "" {
		t.Skip("set MP3TE_SAMPLES_DIR to create sample files")
	}
	gradient := func(w, h int, a, b color.RGBA) []byte {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				f := float64(x+y) / float64(w+h)
				mix := func(p, q uint8) uint8 { return uint8(float64(p)*(1-f) + float64(q)*f) }
				img.Set(x, y, color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 255})
			}
		}
		var buf bytes.Buffer
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		return buf.Bytes()
	}
	ferry := gradient(500, 500, color.RGBA{14, 27, 43, 255}, color.RGBA{245, 165, 36, 255})
	glass := gradient(1200, 1200, color.RGBA{241, 214, 204, 255}, color.RGBA{143, 202, 197, 255})

	write := func(rel string, frames []rawFrame, v24 bool, audioFrames int) {
		path := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		tag := encodeTagV23(frames)
		if v24 && len(tag) > 0 {
			tag[3] = 4 // the frames below only use UTF-8/ISO text, sizes < 128 bytes stay valid
		}
		var b bytes.Buffer
		b.Write(tag)
		b.Write(fakeMP3(audioFrames))
		if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	utf8 := func(id, v string) rawFrame { return rawFrame{id, append([]byte{3}, v...)} }

	for i, title := range []string{"Lantern Wake", "Salt on the Railing", "Fog Bell", "Crossing at Four", "Pier 9"} {
		write(fmt.Sprintf(`Harbor Lights\Night Ferry\0%d - %s.mp3`, i+1, title), []rawFrame{
			textFrame("TIT2", title), textFrame("TPE1", "Harbor Lights"), textFrame("TALB", "Night Ferry"),
			textFrame("TPE2", "Harbor Lights"), textFrame("TRCK", fmt.Sprintf("%d/5", i+1)), textFrame("TYER", "2023"),
			textFrame("TCON", "Indie Folk"), textFrame("TCOM", "Ana Brandt"),
			{"APIC", pictureBody("image/jpeg", 3, ferry)},
		}, false, 1500+i*200)
	}
	for i, title := range []string{"Prism Hours", "Cold Bloom", "Static Garden"} {
		frames := []rawFrame{
			utf8("TIT2", title), utf8("TPE1", "Mira Okafor"), utf8("TALB", "Glass Season"),
			utf8("TRCK", fmt.Sprint(i+1)), utf8("TDRC", "2021-03-12"), utf8("TCON", "Electronic"),
			utf8("TBPM", fmt.Sprint(118+i*3)), utf8("TMOO", "Bright"),
		}
		if i == 0 {
			frames = append(frames, rawFrame{"APIC", pictureBody("image/jpeg", 3, glass)})
		}
		// Only the first file is written with the big cover, so the tag size stays small for the others.
		write(fmt.Sprintf(`Mira Okafor\Glass Season\0%d %s.mp3`, i+1, title), frames, i != 0, 1800)
	}
	for i, title := range []string{"Rust Belt Lullaby", "Copper Hours", "Low Light Freight", "Tin Roof Weather"} {
		write(fmt.Sprintf(`The Lowland Static\Copper Hours\%02d %s.mp3`, i+1, title), []rawFrame{
			textFrame("TIT2", title), textFrame("TPE1", "The Lowland Static"), textFrame("TALB", "Copper Hours"),
			textFrame("TRCK", fmt.Sprintf("%d/4", i+1)), textFrame("TYER", "2019"), textFrame("TCON", "Alternative"),
		}, false, 2100+i*150)
	}
	write(`Downloads\track07.mp3`, nil, false, 900)
	write(`Downloads\Demo 3 (final) FINAL.mp3`, []rawFrame{textFrame("TIT2", "Demo 3 (final) FINAL"), textFrame("TPE1", "Unknown artist")}, false, 700)
	t.Logf("sample files written to %s", dir)
}
