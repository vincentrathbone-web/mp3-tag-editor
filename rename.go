package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	badNameChars  = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	reservedNames = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`)
)

// checkFileName says why name (without extension) can't be used as a Windows file name.
func checkFileName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New("the new name is empty")
	case badNameChars.MatchString(name):
		return errors.New(`a file name can't contain < > : " / \ | ? *`)
	case strings.HasSuffix(name, ".") || strings.HasSuffix(name, " "):
		return errors.New("a file name can't end with a dot or a space")
	case reservedNames.MatchString(name):
		return fmt.Errorf("%q is a name Windows keeps for itself", name)
	case utf8.RuneCountInString(name) > 240:
		return errors.New("the new name is too long")
	}
	return nil
}

// renameFile gives the file a new name in the same folder. The extension stays as it is.
func renameFile(path, newName string) (string, error) {
	if err := checkFileName(newName); err != nil {
		return "", err
	}
	dst := filepath.Join(filepath.Dir(path), newName+filepath.Ext(path))
	if dst == path {
		return path, nil
	}
	src, err := os.Stat(path)
	if err != nil {
		return "", errors.New("the file is no longer there")
	}
	// A file with that name already exists, unless only the upper/lower case is changing.
	if other, err := os.Stat(dst); err == nil && !os.SameFile(src, other) {
		return "", fmt.Errorf("a file called %q already exists in that folder", filepath.Base(dst))
	}
	if err := os.Rename(path, dst); err != nil {
		return "", friendlyWriteError(err)
	}
	return dst, nil
}

// RenameFile renames an MP3 to newName (without extension) and returns the file as it is now on disk.
func (a *App) RenameFile(path, newName string) (Track, error) {
	dst, err := renameFile(path, newName)
	if err != nil {
		return Track{}, err
	}
	return loadTrack(dst), nil
}
