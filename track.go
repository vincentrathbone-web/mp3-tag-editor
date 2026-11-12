package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Track is one MP3 file as the UI sees it.
type Track struct {
	Path       string            `json:"path"`
	Name       string            `json:"name"`
	Ext        string            `json:"ext"`
	Folder     string            `json:"folder"`
	Size       int64             `json:"size"`
	Modified   string            `json:"modified"`
	TagVersion string            `json:"tagVersion"`
	Duration   float64           `json:"duration"`
	Bitrate    int               `json:"bitrate"`
	VBR        bool              `json:"vbr"`
	SampleRate int               `json:"sampleRate"`
	Channels   string            `json:"channels"`
	Codec      string            `json:"codec"`
	Tags       map[string]string `json:"tags"`
	Cover      *Cover            `json:"cover"`
	// Problem explains why the file can't be saved. Empty when everything is fine.
	Problem string `json:"problem"`
}

// loadTrack reads everything the UI shows about one file.
func loadTrack(path string) Track {
	t := Track{
		Path:   path,
		Name:   filepath.Base(path),
		Ext:    strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		Folder: filepath.Dir(path),
		Tags:   map[string]string{},
	}
	f, err := os.Open(path)
	if err != nil {
		t.Problem = "Couldn't open the file: " + err.Error()
		return t
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		t.Problem = err.Error()
		return t
	}
	t.Size = st.Size()
	t.Modified = st.ModTime().Format("2006-01-02 15:04")

	v2, err := readID3v2(f, st.Size())
	if err != nil {
		t.Problem = "The ID3 tag can't be read safely: " + err.Error()
	}
	v1 := readID3v1(f, st.Size())

	var versions []string
	if v2.Version > 0 {
		versions = append(versions, fmt.Sprintf("ID3v2.%d", v2.Version))
		t.Tags = tagValues(v2.Frames)
		if i := coverFrameIndex(v2.Frames); i >= 0 {
			p, _ := parsePicture(v2.Frames[i].Body)
			t.Cover = describePicture(p, false)
		}
	}
	if v1.Present {
		versions = append(versions, fmt.Sprintf("ID3v1.%d", v1.Minor))
		for k, v := range v1.Fields { // fill gaps from the old tag, the ID3v2 values win
			if t.Tags[k] == "" {
				t.Tags[k] = v
			}
		}
	}
	t.TagVersion = strings.Join(versions, " + ")

	end := st.Size()
	if v1.Present {
		end -= 128
	}
	if a, ok := readAudioInfo(f, v2.Size, end); ok {
		t.Duration, t.Bitrate, t.VBR = a.Duration, a.Bitrate, a.VBR
		t.SampleRate, t.Channels = a.SampleRate, a.Channels
		t.Codec = fmt.Sprintf("%s Layer %s", a.Version, strings.Repeat("I", a.Layer))
	} else if t.Problem == "" {
		t.Problem = "No MP3 audio was found in this file."
	}
	return t
}

// expandPaths turns a mix of files and folders into a sorted list of MP3 files.
// Folders are searched including their subfolders.
func expandPaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if strings.EqualFold(filepath.Ext(p), ".mp3") && !seen[strings.ToLower(p)] {
			seen[strings.ToLower(p)] = true
			out = append(out, p)
		}
	}
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !st.IsDir() {
			add(p)
			continue
		}
		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				add(path)
			}
			return nil
		})
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

// loadTracks reads many files in parallel, keeping their order.
func loadTracks(paths []string, progress func(done, total int)) []Track {
	out := make([]Track, len(paths))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				out[i] = loadTrack(paths[i])
				mu.Lock()
				done++
				if progress != nil && (done%25 == 0 || done == len(paths)) {
					progress(done, len(paths))
				}
				mu.Unlock()
			}
		}()
	}
	for i := range paths {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}
