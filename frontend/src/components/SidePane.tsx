import { FIELD, FIELDS } from '../fields'
import { IconImage, IconRows, IconWarn } from '../icons'
import { Editing, Row, fieldChanged, fmtBytes, fmtDuration, isDirty, plural, shownCover, val } from '../model'

interface Props {
  targets: Row[]
  hasFiles: boolean
  editing: Editing | null
  coverBusy: boolean
  fullCover: (r: Row) => string | undefined
  renderEditor: (key: string, initial: string) => React.ReactNode
  onEdit: (key: string) => void
  onPickCover: () => void
  onRemoveCover: () => void
}

export default function SidePane(p: Props) {
  const t = p.targets
  if (!t.length) {
    return (
      <div className="side-empty">
        <IconRows />
        <div>{p.hasFiles ? 'Select a file to see its cover and details.' : 'Add music to get started.'}</div>
        <div className="hint">Ctrl+click or Shift+click selects several files, so you can edit them together.</div>
      </div>
    )
  }
  const one = t.length === 1 ? t[0] : null
  const covers = t.map(shownCover)
  const same = covers.every(c => (c?.thumb ?? '') === (covers[0]?.thumb ?? '') && (c?.bytes ?? 0) === (covers[0]?.bytes ?? 0))
  const cover = same ? covers[0] : null
  const anyCover = covers.some(Boolean)
  const big = cover ? (cover.dataUrl || p.fullCover(t[0]) || cover.thumb) : ''

  const shown = FIELDS.filter(f => t.some(r => val(r, f.key)) || (p.editing?.where === 'pane' && p.editing.key === f.key))
  const empty = FIELDS.filter(f => !shown.includes(f))

  return (
    <>
      <section>
        <div className="sh"><h3>Cover art</h3><span>{one ? 'Front cover' : `Applies to ${plural(t.length, 'file')}`}</span></div>
        <div
          id="cover-drop"
          data-drop="cover"
          className={'cover' + (cover ? ' has' : '')}
          tabIndex={0}
          role="button"
          aria-label="Add or replace cover art"
          onClick={p.onPickCover}
          onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); p.onPickCover() } }}
        >
          {cover ? (
            <>
              <img src={big} alt="Cover art" />
              <div className="hover">Click or drop an image to replace</div>
            </>
          ) : (
            <div className="ph">
              <IconImage />
              <div>{!same ? <>These files have different covers.<br />Drop an image to give them all one cover.</> : 'Drop an image here or click to choose'}</div>
              <small>AVIF · WebP · PNG · JPG · GIF · BMP · TIFF</small>
            </div>
          )}
          {p.coverBusy && <div className="busy">Converting…</div>}
        </div>
        {cover && (
          <div className="cover-meta">
            <div>{cover.mime.replace('image/', '').toUpperCase()} · {cover.width} × {cover.height} · {fmtBytes(cover.bytes)}</div>
            {cover.source && <div className="from">from {cover.source.format} {cover.source.width} × {cover.source.height} · {fmtBytes(cover.source.bytes)}</div>}
            {t.some(r => r.cover.mode === 'set') && <div className="from">Saved as baseline JPEG, max 500 × 500</div>}
          </div>
        )}
        <div className="cover-actions">
          <button className="btn small" onClick={p.onPickCover} disabled={p.coverBusy}><IconImage />{anyCover ? 'Replace…' : 'Choose image…'}</button>
          {anyCover && <button className="btn small ghost" onClick={p.onRemoveCover}>Remove cover</button>}
        </div>
      </section>

      {one?.track.problem && (
        <div className="problem"><IconWarn /><span>{one.track.problem}</span></div>
      )}

      <section>
        <div className="sh"><h3>File</h3></div>
        {one ? <FileProps r={one} /> : <ManyProps rows={t} />}
      </section>

      <section>
        <div className="sh"><h3>Tags</h3><span>{shown.length} filled · click a value to edit</span></div>
        <div className="tags">
          {shown.map(f => {
            const vals = new Set(t.map(r => val(r, f.key)))
            const changed = t.some(r => fieldChanged(r, f.key))
            const editingThis = p.editing?.where === 'pane' && p.editing.key === f.key
            return (
              <div className="tagrow" key={f.key}>
                <div className="lab">{f.label}<span className="fid">{f.frame}</span></div>
                {editingThis ? (
                  <div>{p.renderEditor(f.key, vals.size === 1 ? [...vals][0] : '')}</div>
                ) : (
                  <div
                    className={'val' + (changed ? ' changed' : '')}
                    tabIndex={0}
                    onMouseDown={e => { e.preventDefault(); p.onEdit(f.key) }}
                    onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); p.onEdit(f.key) } }}
                  >
                    {vals.size === 1
                      ? ([...vals][0] || <span className="mixed">Empty</span>)
                      : <span className="mixed">Mixed values ({vals.size})</span>}
                  </div>
                )}
              </div>
            )
          })}
        </div>
        {empty.length > 0 && (
          <div className="add-field">
            <select id="add-field" aria-label="Add a tag field" value="" onChange={e => e.target.value && p.onEdit(e.target.value)}>
              <option value="">+ Add a tag field…</option>
              {empty.map(f => <option key={f.key} value={f.key}>{f.label} ({FIELD[f.key].frame})</option>)}
            </select>
          </div>
        )}
      </section>
    </>
  )
}

function FileProps({ r }: { r: Row }) {
  const t = r.track
  const rows: [string, React.ReactNode, string?][] = [
    ['Name', t.name],
    ['Extension', '.' + t.ext],
    ['Size', fmtBytes(t.size)],
    ['Full path', t.path, 'mono'],
    ['Duration', fmtDuration(t.duration)],
    ['Bitrate', t.bitrate ? `${t.bitrate} kbps${t.vbr ? ' VBR' : ''}` : '—'],
    ['Sample rate', t.sampleRate ? `${t.sampleRate / 1000} kHz` : '—'],
    ['Channels', t.channels || '—'],
    ['Format', t.codec || '—'],
    ['Tag', <>{t.tagVersion || 'No tag yet'}{isDirty(r) && <span className="note">Saves as ID3v2.3</span>}</>],
    ['Modified', t.modified || '—'],
  ]
  return (
    <dl className="props">
      {rows.map(([k, v, c]) => (
        <div className="prop" key={k}><dt>{k}</dt><dd className={c}>{v}</dd></div>
      ))}
    </dl>
  )
}

function ManyProps({ rows }: { rows: Row[] }) {
  const size = rows.reduce((a, r) => a + (r.track.size || 0), 0)
  const dur = rows.reduce((a, r) => a + (r.track.duration || 0), 0)
  const folders = [...new Set(rows.map(r => r.track.folder))]
  return (
    <dl className="props">
      <div className="prop"><dt>Files</dt><dd>{rows.length}</dd></div>
      <div className="prop"><dt>Total size</dt><dd>{fmtBytes(size)}</dd></div>
      <div className="prop"><dt>Total time</dt><dd>{fmtDuration(dur)}</dd></div>
      <div className="prop"><dt>Folder{folders.length > 1 ? 's' : ''}</dt><dd className="mono">{folders.map(f => <div key={f}>{f}</div>)}</dd></div>
    </dl>
  )
}
