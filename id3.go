package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// rawFrame is one ID3v2 frame with its body already freed from unsynchronisation,
// grouping bytes and data-length indicators, so it can be written back as plain ID3v2.3.
type rawFrame struct {
	ID   string
	Body []byte
}

// id3v2Tag is what was found at the start of a file.
type id3v2Tag struct {
	Version byte  // 2, 3 or 4; 0 when the file has no ID3v2 tag
	Size    int64 // bytes the tag takes up at the start of the file (header, frames, padding, footer)
	Frames  []rawFrame
}

// id3v1Tag is the fixed 128-byte tag some files carry at the very end.
type id3v1Tag struct {
	Present bool
	Minor   int // 0 for ID3v1, 1 for ID3v1.1 (has a track number)
	Fields  map[string]string
}

var frameIDPattern = regexp.MustCompile(`^[A-Z0-9]{3,4}$`)

func syncsafe(b []byte) int64 {
	return int64(b[0]&0x7f)<<21 | int64(b[1]&0x7f)<<14 | int64(b[2]&0x7f)<<7 | int64(b[3]&0x7f)
}

func putSyncsafe(n int) []byte {
	return []byte{byte(n >> 21 & 0x7f), byte(n >> 14 & 0x7f), byte(n >> 7 & 0x7f), byte(n & 0x7f)}
}

// deunsync reverses ID3 unsynchronisation: every 0xFF 0x00 pair becomes 0xFF.
func deunsync(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		out = append(out, b[i])
		if b[i] == 0xFF && i+1 < len(b) && b[i+1] == 0x00 {
			i++
		}
	}
	return out
}

// readID3v2 reads the ID3v2 tag (v2.2, v2.3 or v2.4) at the start of r.
func readID3v2(r io.ReaderAt, fileSize int64) (*id3v2Tag, error) {
	hdr := make([]byte, 10)
	if _, err := r.ReadAt(hdr, 0); err != nil || string(hdr[:3]) != "ID3" {
		return &id3v2Tag{}, nil
	}
	ver, flags := hdr[3], hdr[5]
	size := syncsafe(hdr[6:10])
	t := &id3v2Tag{Version: ver, Size: 10 + size}
	if ver == 4 && flags&0x10 != 0 {
		t.Size += 10 // footer
	}
	if ver < 2 || ver > 4 {
		return t, fmt.Errorf("unknown ID3v2.%d tag", ver)
	}
	if 10+size > fileSize {
		return t, errors.New("the ID3 tag is damaged (it claims to be larger than the file)")
	}
	body := make([]byte, size)
	if _, err := r.ReadAt(body, 10); err != nil {
		return t, err
	}
	if ver < 4 && flags&0x80 != 0 {
		body = deunsync(body) // whole-tag unsynchronisation (v2.2 / v2.3)
	}
	pos := 0
	if ver >= 3 && flags&0x40 != 0 && len(body) >= 4 { // skip extended header
		if ver == 3 {
			pos = 4 + int(binary.BigEndian.Uint32(body[:4]))
		} else {
			pos = int(syncsafe(body[:4]))
		}
	}

	for {
		if ver == 2 {
			if pos+6 > len(body) {
				break
			}
			id := string(body[pos : pos+3])
			n := int(body[pos+3])<<16 | int(body[pos+4])<<8 | int(body[pos+5])
			pos += 6
			if !frameIDPattern.MatchString(id) || pos+n > len(body) {
				break
			}
			fb := body[pos : pos+n]
			pos += n
			newID, ok := v22Frames[id]
			if !ok {
				continue // no v2.3 equivalent we know of
			}
			if id == "PIC" {
				fb = convertV22Picture(fb)
			}
			t.Frames = append(t.Frames, rawFrame{newID, append([]byte(nil), fb...)})
			continue
		}

		if pos+10 > len(body) {
			break
		}
		id := string(body[pos : pos+4])
		if !frameIDPattern.MatchString(id) {
			break // reached padding
		}
		var n int
		if ver == 4 {
			n = int(syncsafe(body[pos+4 : pos+8]))
		} else {
			n = int(binary.BigEndian.Uint32(body[pos+4 : pos+8]))
		}
		format := body[pos+9]
		pos += 10
		if n < 0 || pos+n > len(body) {
			break
		}
		fb := append([]byte(nil), body[pos:pos+n]...)
		pos += n

		if ver == 4 {
			if format&0x0C != 0 {
				continue // compressed or encrypted: cannot be carried over, dropped
			}
			skip := 0
			if format&0x40 != 0 {
				skip++ // group id
			}
			if format&0x01 != 0 {
				skip += 4 // data length indicator
			}
			if skip > len(fb) {
				continue
			}
			fb = fb[skip:]
			if format&0x02 != 0 {
				fb = deunsync(fb)
			}
		} else {
			if format&0xC0 != 0 {
				continue // compressed or encrypted: cannot be carried over, dropped
			}
			if format&0x20 != 0 && len(fb) > 0 {
				fb = fb[1:] // group id
			}
		}
		t.Frames = append(t.Frames, rawFrame{id, fb})
	}
	return t, nil
}

// convertV22Picture turns a v2.2 PIC body (3-letter image format) into a v2.3 APIC body (MIME type).
func convertV22Picture(b []byte) []byte {
	if len(b) < 5 {
		return b
	}
	mime := "image/jpeg"
	if strings.EqualFold(string(b[1:4]), "PNG") {
		mime = "image/png"
	}
	out := []byte{b[0]}
	out = append(out, mime...)
	out = append(out, 0)
	return append(out, b[4:]...)
}

// readID3v1 reads the 128-byte ID3v1 tag at the end of the file, if there is one.
func readID3v1(r io.ReaderAt, fileSize int64) id3v1Tag {
	t := id3v1Tag{Fields: map[string]string{}}
	if fileSize < 128 {
		return t
	}
	b := make([]byte, 128)
	if _, err := r.ReadAt(b, fileSize-128); err != nil || string(b[:3]) != "TAG" {
		return t
	}
	t.Present = true
	str := func(x []byte) string {
		if i := bytes.IndexByte(x, 0); i >= 0 {
			x = x[:i]
		}
		return strings.TrimSpace(decodeLatin1(x))
	}
	set := func(k, v string) {
		if v != "" {
			t.Fields[k] = v
		}
	}
	set("title", str(b[3:33]))
	set("artist", str(b[33:63]))
	set("album", str(b[63:93]))
	set("year", str(b[93:97]))
	if b[125] == 0 && b[126] != 0 {
		t.Minor = 1
		set("comment", str(b[97:125]))
		set("track", strconv.Itoa(int(b[126])))
	} else {
		set("comment", str(b[97:127]))
	}
	if g := int(b[127]); g < len(id3v1Genres) {
		set("genre", id3v1Genres[g])
	}
	return t
}

// buildID3v1 makes a fresh 128-byte ID3v1.1 block from the tag values.
func buildID3v1(tags map[string]string) []byte {
	b := make([]byte, 128)
	copy(b, "TAG")
	put := func(dst []byte, v string) { copy(dst, encodeLatin1Lossy(v)) }
	put(b[3:33], tags["title"])
	put(b[33:63], tags["artist"])
	put(b[63:93], tags["album"])
	put(b[93:97], tags["year"])
	put(b[97:125], tags["comment"])
	if n, err := strconv.Atoi(strings.SplitN(tags["track"], "/", 2)[0]); err == nil && n > 0 && n < 256 {
		b[126] = byte(n)
	}
	b[127] = 255
	for i, g := range id3v1Genres {
		if strings.EqualFold(g, tags["genre"]) {
			b[127] = byte(i)
		}
	}
	return b
}

/* ---------- Text encoding ---------- */

func decodeLatin1(b []byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

func encodeLatin1Lossy(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF {
			r = '?'
		}
		out = append(out, byte(r))
	}
	return out
}

func decodeText(b []byte, enc byte) string {
	switch enc {
	case 1, 2:
		bigEndian := enc == 2
		if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
			bigEndian, b = false, b[2:]
		} else if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
			bigEndian, b = true, b[2:]
		}
		u := make([]uint16, 0, len(b)/2)
		for i := 0; i+1 < len(b); i += 2 {
			if bigEndian {
				u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
			} else {
				u = append(u, uint16(b[i+1])<<8|uint16(b[i]))
			}
		}
		return string(utf16.Decode(u))
	case 3:
		return string(b)
	default:
		return decodeLatin1(b)
	}
}

// cleanText removes trailing terminators and joins v2.4-style multiple values.
func cleanText(s string) string {
	s = strings.TrimRight(s, "\x00")
	parts := strings.Split(s, "\x00")
	out := parts[:0]
	for _, p := range parts {
		// A BOM can be left at the start of each value after splitting UTF-16 text.
		if p = strings.TrimSpace(strings.TrimPrefix(p, "\ufeff")); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "; ")
}

// splitTerminated splits b at the first string terminator for the encoding.
func splitTerminated(b []byte, enc byte) ([]byte, []byte) {
	if enc == 1 || enc == 2 {
		for i := 0; i+1 < len(b); i += 2 {
			if b[i] == 0 && b[i+1] == 0 {
				return b[:i], b[i+2:]
			}
		}
		return b, nil
	}
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return b[:i], b[i+1:]
	}
	return b, nil
}

// pickEncoding picks the most compact ID3v2.3 encoding that can hold all strings:
// ISO-8859-1 when possible, otherwise UTF-16 with BOM.
func pickEncoding(strs ...string) byte {
	for _, s := range strs {
		for _, r := range s {
			if r > 0xFF {
				return 1
			}
		}
	}
	return 0
}

func encodeText(s string, enc byte) []byte {
	if enc == 0 {
		return encodeLatin1Lossy(s)
	}
	u := utf16.Encode([]rune(s))
	out := make([]byte, 2, 2+len(u)*2)
	out[0], out[1] = 0xFF, 0xFE
	for _, c := range u {
		out = append(out, byte(c), byte(c>>8))
	}
	return out
}

func terminator(enc byte) []byte {
	if enc == 1 {
		return []byte{0, 0}
	}
	return []byte{0}
}

/* ---------- Frame parsing ---------- */

type pictureInfo struct {
	Mime string
	Type byte
	Data []byte
}

func parseTextFrame(b []byte) string {
	if len(b) < 1 {
		return ""
	}
	return cleanText(decodeText(b[1:], b[0]))
}

// parseLangFrame parses COMM and USLT bodies: encoding, language, description, text.
func parseLangFrame(b []byte) (lang, desc, text string, ok bool) {
	if len(b) < 4 {
		return "", "", "", false
	}
	enc := b[0]
	d, rest := splitTerminated(b[4:], enc)
	return string(b[1:4]), cleanText(decodeText(d, enc)), strings.TrimRight(decodeText(rest, enc), "\x00"), true
}

func parsePicture(b []byte) (pictureInfo, bool) {
	if len(b) < 4 {
		return pictureInfo{}, false
	}
	enc := b[0]
	mime, rest := splitTerminated(b[1:], 0)
	if len(rest) < 1 {
		return pictureInfo{}, false
	}
	ptype := rest[0]
	_, data := splitTerminated(rest[1:], enc)
	m := strings.ToLower(string(mime))
	if m == "" || !strings.Contains(m, "/") {
		m = "image/" + strings.TrimPrefix(m, "image/")
		if m == "image/" || m == "image/jpg" {
			m = "image/jpeg"
		}
	}
	return pictureInfo{m, ptype, data}, len(data) > 0
}

// coverFrameIndex returns the index of the frame shown as "the cover":
// the front cover (type 3), else an "other" picture (type 0), else the first picture.
func coverFrameIndex(frames []rawFrame) int {
	best, bestRank := -1, 9
	for i, f := range frames {
		if f.ID != "APIC" {
			continue
		}
		p, ok := parsePicture(f.Body)
		if !ok {
			continue
		}
		rank := 2
		switch p.Type {
		case 3:
			rank = 0
		case 0:
			rank = 1
		}
		if rank < bestRank {
			best, bestRank = i, rank
		}
	}
	return best
}

var genreRef = regexp.MustCompile(`^\((\d+)\)(.*)$`)

// normalizeGenre turns ID3v1-style genre references like "(17)" or "17" into names.
func normalizeGenre(s string) string {
	if m := genreRef.FindStringSubmatch(s); m != nil {
		if strings.TrimSpace(m[2]) != "" {
			return strings.TrimSpace(m[2])
		}
		s = m[1]
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < len(id3v1Genres) {
		return id3v1Genres[n]
	}
	switch s {
	case "(RX)", "RX":
		return "Remix"
	case "(CR)", "CR":
		return "Cover"
	}
	return s
}

// tagValues extracts the editable field values from the frames.
func tagValues(frames []rawFrame) map[string]string {
	out := map[string]string{}
	byFrame := map[string]string{}
	for _, f := range Fields {
		byFrame[f.Frame] = f.Key
	}
	for _, f := range frames {
		switch {
		case f.ID == "COMM":
			if _, desc, text, ok := parseLangFrame(f.Body); ok && desc == "" && out["comment"] == "" {
				out["comment"] = strings.TrimSpace(text)
			}
		case f.ID == "USLT":
			if _, _, text, ok := parseLangFrame(f.Body); ok && out["lyrics"] == "" {
				out["lyrics"] = strings.TrimSpace(text)
			}
		case f.ID == "WOAR":
			out["url"] = cleanText(decodeLatin1(f.Body))
		case f.ID[0] == 'T' && f.ID != "TXXX":
			v := parseTextFrame(f.Body)
			if v == "" {
				continue
			}
			if k, ok := v24Aliases[f.ID]; ok {
				if out[k] == "" && len(v) >= 4 {
					out[k] = v[:4]
				}
				continue
			}
			if k, ok := byFrame[f.ID]; ok {
				if k == "genre" {
					v = normalizeGenre(v)
				}
				out[k] = v
			}
		}
	}
	for k, v := range out {
		if v == "" {
			delete(out, k)
		}
	}
	return out
}

/* ---------- Writing ---------- */

// toV23 rewrites a frame body so it is valid ID3v2.3: UTF-8 and UTF-16BE text
// (only allowed in v2.4) is re-encoded as ISO-8859-1 or UTF-16 with BOM.
func toV23(f rawFrame) (rawFrame, bool) {
	if v24Only[f.ID] {
		return f, false
	}
	b := f.Body
	if len(b) == 0 {
		return f, false
	}
	enc := b[0]
	needs := enc == 2 || enc == 3
	switch {
	case f.ID[0] == 'T' && f.ID != "TXXX":
		if needs {
			s := cleanText(decodeText(b[1:], enc))
			e := pickEncoding(s)
			f.Body = append([]byte{e}, encodeText(s, e)...)
		}
	case f.ID == "TXXX" || f.ID == "WXXX":
		if needs {
			d, rest := splitTerminated(b[1:], enc)
			desc := decodeText(d, enc)
			if f.ID == "WXXX" {
				e := pickEncoding(desc)
				f.Body = append(append(append([]byte{e}, encodeText(desc, e)...), terminator(e)...), rest...)
			} else {
				val := cleanText(decodeText(rest, enc))
				e := pickEncoding(desc, val)
				f.Body = append(append(append([]byte{e}, encodeText(desc, e)...), terminator(e)...), encodeText(val, e)...)
			}
		}
	case f.ID == "COMM" || f.ID == "USLT":
		if needs {
			lang, desc, text, ok := parseLangFrame(b)
			if !ok {
				return f, false
			}
			f.Body = langFrameBody(lang, desc, text)
		}
	case f.ID == "APIC":
		if needs {
			p, ok := parsePicture(b)
			if !ok {
				return f, false
			}
			f.Body = pictureBody(p.Mime, p.Type, p.Data)
		}
	}
	return f, true
}

func textFrame(id, value string) rawFrame {
	e := pickEncoding(value)
	return rawFrame{id, append([]byte{e}, encodeText(value, e)...)}
}

func langFrameBody(lang, desc, text string) []byte {
	if len(lang) != 3 {
		lang = "eng"
	}
	e := pickEncoding(desc, text)
	b := append([]byte{e}, lang...)
	b = append(b, encodeText(desc, e)...)
	b = append(b, terminator(e)...)
	return append(b, encodeText(text, e)...)
}

func pictureBody(mime string, ptype byte, data []byte) []byte {
	b := append([]byte{0}, mime...)
	b = append(b, 0, ptype, 0) // mime terminator, picture type, empty description
	return append(b, data...)
}

// CoverChange says what to do with the cover when saving.
type CoverChange struct {
	Mode string // "keep", "set" or "remove"
	JPEG []byte // the new front cover when Mode is "set"
}

// buildFrames merges the edited values into the existing frames and returns a v2.3 frame list.
func buildFrames(existing []rawFrame, tags map[string]string, cover CoverChange) []rawFrame {
	managed := map[string]bool{"TDRC": true, "TDOR": true, "WOAR": true, "USLT": true}
	for _, f := range Fields {
		managed[f.Frame] = true
	}
	coverIdx := -1
	if cover.Mode != "keep" {
		coverIdx = coverFrameIndex(existing)
	}

	var head, keep, pics []rawFrame
	for _, f := range Fields {
		v := strings.TrimSpace(tags[f.Key])
		if v == "" {
			continue
		}
		switch f.Frame {
		case "COMM":
			head = append(head, rawFrame{"COMM", langFrameBody("eng", "", v)})
		case "USLT":
			head = append(head, rawFrame{"USLT", langFrameBody("eng", "", v)})
		case "WOAR":
			head = append(head, rawFrame{"WOAR", encodeLatin1Lossy(v)})
		default:
			head = append(head, textFrame(f.Frame, v))
		}
	}

	for i, f := range existing {
		if f.ID == "COMM" {
			// Only the plain comment (empty description) is edited; other comments
			// (for example iTunes' iTunNORM data) are kept.
			if _, desc, _, ok := parseLangFrame(f.Body); ok && desc == "" {
				continue
			}
		} else if managed[f.ID] {
			continue
		}
		if f.ID == "APIC" && cover.Mode != "keep" {
			if i == coverIdx {
				continue
			}
			if p, ok := parsePicture(f.Body); ok && p.Type == 3 {
				continue // a file has only one front cover
			}
		}
		g, ok := toV23(f)
		if !ok {
			continue
		}
		if g.ID == "APIC" {
			pics = append(pics, g)
		} else {
			keep = append(keep, g)
		}
	}
	if cover.Mode == "set" && len(cover.JPEG) > 0 {
		pics = append([]rawFrame{{"APIC", pictureBody("image/jpeg", 3, cover.JPEG)}}, pics...)
	}
	// Pictures go last so players can read the text fields without loading the images.
	return append(append(head, keep...), pics...)
}

func encodeTagV23(frames []rawFrame) []byte {
	if len(frames) == 0 {
		return nil
	}
	const padding = 1024 // lets other tools make small edits without rewriting the file
	var body bytes.Buffer
	for _, f := range frames {
		body.WriteString(f.ID)
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(f.Body)))
		body.Write(n[:])
		body.Write([]byte{0, 0})
		body.Write(f.Body)
	}
	body.Write(make([]byte, padding))
	out := []byte{'I', 'D', '3', 3, 0, 0}
	out = append(out, putSyncsafe(body.Len())...)
	return append(out, body.Bytes()...)
}

// writeTags saves the new tag to path. The new file is first written next to the
// original and then swapped in, so a failure never leaves a half-written MP3.
func writeTags(path string, tags map[string]string, cover CoverChange) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil {
		return err
	}
	old, err := readID3v2(src, st.Size())
	if err != nil {
		return fmt.Errorf("can't safely rewrite this file's tag: %w", err)
	}
	v1 := readID3v1(src, st.Size())

	tagBytes := encodeTagV23(buildFrames(old.Frames, tags, cover))
	audioEnd := st.Size()
	if v1.Present {
		audioEnd -= 128
	}
	if audioEnd < old.Size {
		return errors.New("the file is too short to contain audio")
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".mp3tag-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(tagBytes); err != nil {
		return err
	}
	if _, err = io.Copy(tmp, io.NewSectionReader(src, old.Size, audioEnd-old.Size)); err != nil {
		return err
	}
	if v1.Present {
		// Keep the old-style tag in step so older players show the same information.
		if _, err = tmp.Write(buildID3v1(tags)); err != nil {
			return err
		}
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	src.Close()
	if err = os.Rename(tmpName, path); err != nil {
		return friendlyWriteError(err)
	}
	ok = true
	return nil
}

func friendlyWriteError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "used by another process") || strings.Contains(msg, "Access is denied") {
		return errors.New("the file is open in another program or is read-only. Close it there and try again")
	}
	return err
}
