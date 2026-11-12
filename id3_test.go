package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// fakeMP3 returns n silent MPEG-1 Layer III frames (128 kbps, 44.1 kHz, joint stereo).
func fakeMP3(n int) []byte {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		f := make([]byte, 417)
		copy(f, []byte{0xFF, 0xFB, 0x90, 0x64})
		b.Write(f)
	}
	return b.Bytes()
}

func v24Frame(id string, body []byte) []byte {
	out := []byte(id)
	out = append(out, putSyncsafe(len(body))...)
	out = append(out, 0, 0)
	return append(out, body...)
}

// v24Tag builds an ID3v2.4 tag with UTF-8 text, the kind many modern tools write.
func v24Tag(frames ...[]byte) []byte {
	body := bytes.Join(frames, nil)
	out := []byte{'I', 'D', '3', 4, 0, 0}
	out = append(out, putSyncsafe(len(body))...)
	return append(out, body...)
}

func TestRoundTripV24ToV23(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.mp3")
	audio := fakeMP3(200)
	tag := v24Tag(
		v24Frame("TIT2", append([]byte{3}, "Café Ünïcode ☕"...)),
		v24Frame("TPE1", append([]byte{3}, "Artist A\x00Artist B"...)),
		v24Frame("TDRC", append([]byte{3}, "2021-05-03"...)),
		v24Frame("TCON", append([]byte{3}, "(17)"...)),
		v24Frame("TXXX", append([]byte{3}, "MOOD\x00Calm"...)),
		v24Frame("COMM", append([]byte{3}, "eng\x00Old comment"...)),
		v24Frame("COMM", append([]byte{3}, "engiTunNORM\x00 000001"...)),
		v24Frame("TMOO", append([]byte{3}, "Calm"...)),
	)
	v1 := buildID3v1(map[string]string{"title": "old v1 title"})
	if err := os.WriteFile(path, append(append(tag, audio...), v1...), 0o644); err != nil {
		t.Fatal(err)
	}

	tr := loadTrack(path)
	if tr.Problem != "" {
		t.Fatalf("problem: %s", tr.Problem)
	}
	want := map[string]string{"title": "Café Ünïcode ☕", "artist": "Artist A; Artist B", "year": "2021", "genre": "Rock", "comment": "Old comment"}
	for k, v := range want {
		if tr.Tags[k] != v {
			t.Errorf("read %s = %q, want %q", k, tr.Tags[k], v)
		}
	}
	if tr.TagVersion != "ID3v2.4 + ID3v1.0" {
		t.Errorf("tag version %q", tr.TagVersion)
	}
	if tr.Bitrate != 128 || tr.SampleRate != 44100 || tr.Channels != "Joint stereo" {
		t.Errorf("audio info %+v", tr)
	}

	// A red PNG with transparency, larger than 500 px, becomes a 500×250 JPEG.
	img := image.NewNRGBA(image.Rect(0, 0, 1000, 500))
	for x := 0; x < 1000; x++ {
		for y := 0; y < 500; y++ {
			img.Set(x, y, color.NRGBA{255, 0, 0, uint8(x % 256)})
		}
	}
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	cov, err := convertCover(pngBuf.Bytes(), "red.png")
	if err != nil {
		t.Fatal(err)
	}
	if cov.Width != 500 || cov.Height != 250 || cov.Source.Format != "PNG" {
		t.Errorf("cover %dx%d from %s", cov.Width, cov.Height, cov.Source.Format)
	}
	jpegBytes, _ := decodeDataURL(cov.DataURL)
	if !isBaselineJPEG(jpegBytes) {
		t.Error("cover is not baseline JPEG")
	}

	tags := map[string]string{"title": "Neuer Titel ☕", "artist": "Artist A; Artist B", "year": "2022", "genre": "Rock", "track": "3/12", "comment": "New comment"}
	if _, err := (&App{}).SaveTrack(SaveRequest{Path: path, Tags: tags, Cover: "set", CoverData: cov.DataURL}); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)
	if raw[3] != 3 {
		t.Fatalf("saved as ID3v2.%d, want 2.3", raw[3])
	}
	if !bytes.Contains(raw, audio) {
		t.Fatal("audio data changed")
	}
	if !bytes.HasSuffix(raw, []byte{0}) && string(raw[len(raw)-128:len(raw)-125]) != "TAG" {
		t.Fatal("ID3v1 tag missing")
	}
	f, _ := os.Open(path)
	st, _ := f.Stat()
	saved, err := readID3v2(f, st.Size())
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int{}
	for _, fr := range saved.Frames {
		ids[fr.ID]++
		if len(fr.Body) > 0 && (fr.Body[0] == 2 || fr.Body[0] == 3) && fr.ID[0] == 'T' {
			t.Errorf("%s still uses a v2.4-only text encoding", fr.ID)
		}
	}
	if ids["TDRC"] != 0 || ids["TMOO"] != 0 {
		t.Errorf("v2.4-only frames left: %v", ids)
	}
	if ids["TXXX"] != 1 || ids["COMM"] != 2 || ids["APIC"] != 1 {
		t.Errorf("frames after save: %v", ids)
	}

	tr2 := loadTrack(path)
	for k, v := range tags {
		if tr2.Tags[k] != v {
			t.Errorf("after save %s = %q, want %q", k, tr2.Tags[k], v)
		}
	}
	if tr2.Cover == nil || tr2.Cover.Width != 500 {
		t.Errorf("cover after save: %+v", tr2.Cover)
	}
	if tr2.TagVersion != "ID3v2.3 + ID3v1.1" {
		t.Errorf("tag version after save %q", tr2.TagVersion)
	}

	// Removing the cover and clearing a field.
	delete(tags, "comment")
	if _, err := (&App{}).SaveTrack(SaveRequest{Path: path, Tags: tags, Cover: "remove"}); err != nil {
		t.Fatal(err)
	}
	tr3 := loadTrack(path)
	if tr3.Cover != nil || tr3.Tags["comment"] != "" {
		t.Errorf("after remove: cover %v comment %q", tr3.Cover, tr3.Tags["comment"])
	}
}

func TestNoTagFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bare.mp3")
	os.WriteFile(path, fakeMP3(50), 0o644)
	tr := loadTrack(path)
	if tr.TagVersion != "" || tr.Problem != "" || tr.Duration < 1.2 || tr.Duration > 1.4 {
		t.Fatalf("bare file: %+v", tr)
	}
	if _, err := (&App{}).SaveTrack(SaveRequest{Path: path, Tags: map[string]string{"title": "Hello"}, Cover: "keep"}); err != nil {
		t.Fatal(err)
	}
	if got := loadTrack(path); got.Tags["title"] != "Hello" || got.TagVersion != "ID3v2.3" {
		t.Fatalf("after save: %+v", got)
	}
}
