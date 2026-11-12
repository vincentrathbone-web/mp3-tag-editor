import { IconFile, IconFolder, IconMusic } from '../icons'

interface Props {
  onAddFolder: () => void
  onAddFiles: () => void
  disabled?: boolean
}

// The drag & drop area with the two "Add" buttons under it.
export default function DropBlock({ onAddFolder, onAddFiles, disabled }: Props) {
  return (
    <div>
      <div className="dropzone" data-drop="music">
        <div className="dz-icon"><IconMusic /></div>
        <div className="dz-title">Drag MP3 files or folders here</div>
        <div className="dz-sub">Only .mp3 files are added. Dropped folders are searched, including their subfolders.</div>
      </div>
      <div className="drop-actions">
        <button className="btn" onClick={onAddFolder} disabled={disabled}><IconFolder />Add folder…</button>
        <button className="btn" onClick={onAddFiles} disabled={disabled}><IconFile />Add file(s)…</button>
      </div>
    </div>
  )
}
