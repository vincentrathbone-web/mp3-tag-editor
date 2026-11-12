package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds everything the window can call. Its exported methods become
// TypeScript functions in frontend/wailsjs/go/main/App.
type App struct {
	ctx     context.Context
	unsaved atomic.Int32
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// beforeClose asks before closing when some files still have unsaved changes.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	n := a.unsaved.Load()
	if n == 0 {
		return false
	}
	files := "file has"
	if n > 1 {
		files = "files have"
	}
	answer, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Unsaved changes",
		Message:       fmt.Sprintf("%d %s unsaved changes. Close without saving?", n, files),
		DefaultButton: "No",
	})
	return err != nil || answer != "Yes"
}

// SetUnsavedCount tells the app how many files have unsaved changes, for the close warning.
func (a *App) SetUnsavedCount(n int) { a.unsaved.Store(int32(n)) }

var mp3Filter = []runtime.FileFilter{{DisplayName: "MP3 files (*.mp3)", Pattern: "*.mp3"}}
var imageFilter = []runtime.FileFilter{{
	DisplayName: "Images (AVIF, WebP, PNG, JPG, GIF, BMP, TIFF)",
	Pattern:     "*.avif;*.webp;*.png;*.jpg;*.jpeg;*.jfif;*.gif;*.bmp;*.tif;*.tiff",
}}

// PickFiles shows the Windows file picker for MP3 files. Returns no tracks when cancelled.
func (a *App) PickFiles() ([]Track, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Add MP3 files", Filters: mp3Filter,
	})
	if err != nil || len(paths) == 0 {
		return []Track{}, err
	}
	return a.LoadPaths(paths)
}

// PickFolder shows the Windows folder picker and adds every MP3 inside it, including subfolders.
func (a *App) PickFolder() ([]Track, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Add a folder of MP3 files"})
	if err != nil || dir == "" {
		return []Track{}, err
	}
	return a.LoadPaths([]string{dir})
}

// LoadPaths reads files and folders (for example from drag and drop).
// It sends "load-progress" events while it works on large folders.
func (a *App) LoadPaths(paths []string) ([]Track, error) {
	files := expandPaths(paths)
	if len(files) == 0 {
		return []Track{}, nil
	}
	return loadTracks(files, func(done, total int) {
		if a.ctx != nil && total > 25 {
			runtime.EventsEmit(a.ctx, "load-progress", done, total)
		}
	}), nil
}

// GetCover returns the full cover image stored in a file, or nil if it has none.
func (a *App) GetCover(path string) (*Cover, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	tag, err := readID3v2(f, st.Size())
	if err != nil {
		return nil, err
	}
	i := coverFrameIndex(tag.Frames)
	if i < 0 {
		return nil, nil
	}
	p, _ := parsePicture(tag.Frames[i].Body)
	return describePicture(p, true), nil
}

// PickCover shows the Windows file picker for an image and converts it. Returns nil when cancelled.
func (a *App) PickCover() (*Cover, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose cover art", Filters: imageFilter})
	if err != nil || path == "" {
		return nil, err
	}
	return a.ConvertCoverFile(path)
}

// ConvertCoverFile converts an image file on disk (for example one dropped on the cover box).
func (a *App) ConvertCoverFile(path string) (*Cover, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return convertCover(data, filepath.Base(path))
}

// ConvertCoverData converts image bytes sent from the page (used when testing in a browser).
func (a *App) ConvertCoverData(name string, data string) (*Cover, error) {
	b, err := decodeDataURL(data)
	if err != nil {
		return nil, err
	}
	return convertCover(b, name)
}

// SaveRequest is one file's changes.
type SaveRequest struct {
	Path      string            `json:"path"`
	Tags      map[string]string `json:"tags"`
	Cover     string            `json:"cover"`     // "keep", "set" or "remove"
	CoverData string            `json:"coverData"` // the new JPEG as a data URL when Cover is "set"
}

// SaveTrack writes the changes as an ID3v2.3 tag and returns the file as it is now on disk.
func (a *App) SaveTrack(req SaveRequest) (Track, error) {
	change := CoverChange{Mode: req.Cover}
	switch req.Cover {
	case "keep", "remove":
	case "set":
		b, err := decodeDataURL(req.CoverData)
		if err != nil {
			return Track{}, err
		}
		change.JPEG = b
	default:
		return Track{}, errors.New("unknown cover action " + req.Cover)
	}
	if err := writeTags(req.Path, req.Tags, change); err != nil {
		return Track{}, err
	}
	return loadTrack(req.Path), nil
}
