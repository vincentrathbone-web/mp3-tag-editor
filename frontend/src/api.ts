// The Go functions the page can call, plus a few helpers about where the page runs.
export {
  ConvertCoverData, ConvertCoverFile, GetCover, LoadPaths, PickCover, PickFiles, PickFolder,
  RenameFile, SaveTrack, SetUnsavedCount,
} from '../wailsjs/go/main/App'
export { EventsOn, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime'

// True inside the Windows app window (WebView2). False when testing in a normal browser
// through `wails dev`, where drag and drop can't tell us where files are on disk.
export const isDesktop = !!(window as any).chrome?.webview

export const IMAGE_EXT = /\.(avif|webp|png|jpe?g|jfif|gif|bmp|tiff?)$/i

export function readAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onload = () => resolve(String(r.result))
    r.onerror = () => reject(r.error)
    r.readAsDataURL(file)
  })
}

export function errorText(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
