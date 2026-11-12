package main

import (
	"encoding/binary"
	"io"
)

// audioInfo is what we can tell about the MP3 audio itself.
type audioInfo struct {
	Duration   float64 // seconds
	Bitrate    int     // kbps (average for VBR files)
	VBR        bool
	SampleRate int // Hz
	Channels   string
	Layer      int
	Version    string
}

var bitrates = map[[2]int][]int{
	{1, 1}: {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
	{1, 2}: {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
	{1, 3}: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
	{2, 1}: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
	{2, 2}: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	{2, 3}: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
}

type mpegHeader struct {
	version    int // 1 = MPEG-1, 2 = MPEG-2, 25 = MPEG-2.5
	layer      int
	bitrate    int // kbps
	sampleRate int
	padding    int
	mono       bool
	chanMode   byte
}

func parseMPEGHeader(b []byte) (mpegHeader, bool) {
	if len(b) < 4 || b[0] != 0xFF || b[1]&0xE0 != 0xE0 {
		return mpegHeader{}, false
	}
	var h mpegHeader
	switch (b[1] >> 3) & 3 {
	case 0:
		h.version = 25
	case 2:
		h.version = 2
	case 3:
		h.version = 1
	default:
		return h, false
	}
	switch (b[1] >> 1) & 3 {
	case 1:
		h.layer = 3
	case 2:
		h.layer = 2
	case 3:
		h.layer = 1
	default:
		return h, false
	}
	bi, si := int(b[2]>>4), int((b[2]>>2)&3)
	if bi == 0 || bi == 15 || si == 3 {
		return h, false
	}
	tv := 1
	if h.version != 1 {
		tv = 2
	}
	h.bitrate = bitrates[[2]int{tv, h.layer}][bi]
	h.sampleRate = []int{44100, 48000, 32000}[si]
	if h.version == 2 {
		h.sampleRate /= 2
	} else if h.version == 25 {
		h.sampleRate /= 4
	}
	h.padding = int((b[2] >> 1) & 1)
	h.chanMode = b[3] >> 6
	h.mono = h.chanMode == 3
	return h, true
}

func (h mpegHeader) frameLength() int {
	if h.layer == 1 {
		return (12*h.bitrate*1000/h.sampleRate + h.padding) * 4
	}
	if h.layer == 3 && h.version != 1 {
		return 72*h.bitrate*1000/h.sampleRate + h.padding
	}
	return 144*h.bitrate*1000/h.sampleRate + h.padding
}

func (h mpegHeader) samplesPerFrame() int {
	switch {
	case h.layer == 1:
		return 384
	case h.layer == 3 && h.version != 1:
		return 576
	default:
		return 1152
	}
}

// readAudioInfo finds the first MPEG audio frame after the ID3v2 tag and works out
// duration and bitrate, using the Xing/Info or VBRI header when the encoder wrote one.
func readAudioInfo(r io.ReaderAt, start, end int64) (audioInfo, bool) {
	const scan = 256 * 1024
	buf := make([]byte, scan)
	n, _ := r.ReadAt(buf, start)
	buf = buf[:n]

	for i := 0; i+4 <= len(buf); i++ {
		h, ok := parseMPEGHeader(buf[i:])
		if !ok {
			continue
		}
		fl := h.frameLength()
		// Make sure the next frame header is also valid, to avoid false matches.
		if i+fl+4 <= len(buf) {
			if h2, ok2 := parseMPEGHeader(buf[i+fl:]); !ok2 || h2.sampleRate != h.sampleRate || h2.layer != h.layer {
				continue
			}
		}
		info := audioInfo{SampleRate: h.sampleRate, Layer: h.layer, Bitrate: h.bitrate}
		info.Channels = []string{"Stereo", "Joint stereo", "Dual channel", "Mono"}[h.chanMode]
		info.Version = map[int]string{1: "MPEG-1", 2: "MPEG-2", 25: "MPEG-2.5"}[h.version]
		audioBytes := end - (start + int64(i))

		frame := buf[i:]
		frames, size := int64(0), int64(0)
		if h.layer == 3 {
			side := 32
			if h.version == 1 && h.mono || h.version != 1 && !h.mono {
				side = 17
			} else if h.version != 1 && h.mono {
				side = 9
			}
			if x := 4 + side; x+16 <= len(frame) {
				tag := string(frame[x : x+4])
				if tag == "Xing" || tag == "Info" {
					flags := binary.BigEndian.Uint32(frame[x+4:])
					p := x + 8
					if flags&1 != 0 && p+4 <= len(frame) {
						frames = int64(binary.BigEndian.Uint32(frame[p:]))
						p += 4
					}
					if flags&2 != 0 && p+4 <= len(frame) {
						size = int64(binary.BigEndian.Uint32(frame[p:]))
					}
					info.VBR = tag == "Xing"
				}
			}
			if frames == 0 && 36+18 <= len(frame) && string(frame[36:40]) == "VBRI" {
				size = int64(binary.BigEndian.Uint32(frame[36+10:]))
				frames = int64(binary.BigEndian.Uint32(frame[36+14:]))
				info.VBR = true
			}
		}
		if frames > 0 {
			info.Duration = float64(frames) * float64(h.samplesPerFrame()) / float64(h.sampleRate)
			if size <= 0 {
				size = audioBytes
			}
			if info.Duration > 0 {
				info.Bitrate = int(float64(size)*8/info.Duration/1000 + 0.5)
			}
		} else if h.bitrate > 0 {
			info.Duration = float64(audioBytes) * 8 / float64(h.bitrate*1000)
		}
		return info, true
	}
	return audioInfo{}, false
}
