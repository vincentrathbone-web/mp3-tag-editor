package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.MP3")
	b := filepath.Join(dir, "b.mp3")
	os.WriteFile(a, fakeMP3(5), 0o644)
	os.WriteFile(b, fakeMP3(5), 0o644)

	got, err := renameFile(a, "Fog Bell - 03 - Harbor Lights")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Fog Bell - 03 - Harbor Lights.MP3"); got != want {
		t.Fatalf("got %s, want %s (the extension must be kept)", got, want)
	}
	if _, err := renameFile(got, "b"); err == nil {
		t.Fatal("renaming over an existing file must fail")
	}
	if _, err := renameFile(got, "fog bell - 03 - harbor lights"); err != nil {
		t.Fatalf("changing only the case must work: %v", err)
	}
	for _, bad := range []string{"", "  ", "a/b", "what?", "end.", "CON", "a:b"} {
		if _, err := renameFile(b, bad); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}
