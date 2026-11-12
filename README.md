# MP3 Tag Editor

A small dark-themed Windows app for editing MP3 tags and cover art.
Built with [Wails](https://wails.io) (Go) and React + TypeScript + Vite.

## What it does

- Add music with **drag & drop** (files or whole folders, including subfolders), **Add folder…** or **Add file(s)…**.
- Edit tags in a table. Click a cell to edit it: **Enter** moves down, **Tab** moves right, **Esc** cancels.
- **Columns** shows any of the 28 ID3 fields. Title, Artist and Album are shown by default.
- Select several files (Ctrl+click / Shift+click) to edit a field or set a cover for all of them at once.
- Cover art from AVIF, WebP, PNG, JPG, GIF, BMP or TIFF is converted to a **baseline JPEG, at most 500×500**.
- The right pane shows the cover, file details (size, full path, duration, bitrate…) and every filled-in tag.
- Nothing is written until you click **Save** / **Save all** (Ctrl+S). Tags are saved as **ID3v2.3**.
  An existing ID3v1 tag at the end of the file is updated as well.

## Requirements

- Go and the Wails CLI (installed in `D:\Dev\toolshed`)
- Node.js and npm
- WebView2 (built into Windows 11)

## Run it while developing

```bash
wails dev
```

This opens the app window and also serves it to a browser at http://localhost:34115,
where the buttons still call the real Go code. Dropping music only works in the app window,
because a browser can't see where dropped files are stored.

## Build the .exe

```bash
wails build -trimpath -ldflags "-s -w"
```

The app is written to `build\bin\MP3 Tag Editor.exe`.

## Tests

```bash
go test .
```

To create a folder of silent sample MP3s for trying the app:

```powershell
$env:MP3TE_SAMPLES_DIR = "D:\temp\mp3-samples"; go test -run TestMakeSamples .
```

## Project layout

| File | What it does |
| --- | --- |
| `app.go` | Functions the window can call (pick files, load, convert cover, save) |
| `id3.go` | Reads ID3v1/v2.2/v2.3/v2.4 tags and writes clean ID3v2.3 |
| `mpeg.go` | Reads duration, bitrate and sample rate from the audio |
| `cover.go` | Converts images to the 500×500 JPEG cover |
| `track.go` | Loads files and folders |
| `fields.go` | The list of tag fields (keep in step with `frontend/src/fields.ts`) |
| `frontend/src` | The React screen |
