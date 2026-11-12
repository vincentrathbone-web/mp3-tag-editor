import { useMemo, useRef, useState } from 'react'
import { IconClose, IconWarn } from '../icons'
import { Row, plural } from '../model'
import { CHIPS, CHIP_HELP, buildName, extOf, parseFormat } from '../rename'

export interface RenameItem { id: number; name: string }

interface Props {
  rows: Row[] // the selected files, in table order
  otherPaths: Set<string> // lower-case paths of the files in the list that aren't selected
  format: string
  busy: string | null
  onClose: () => void
  onRename: (items: RenameItem[], format: string) => void
}

const folderOf = (p: string) => p.slice(0, Math.max(p.lastIndexOf('\\'), p.lastIndexOf('/')) + 1)

export default function RenameDialog({ rows, otherPaths, format: initial, busy, onClose, onRename }: Props) {
  const [format, setFormat] = useState(initial)
  const input = useRef<HTMLInputElement>(null)
  const parsed = useMemo(() => parseFormat(format), [format])

  const preview = useMemo(() => {
    if (parsed.error) return []
    const list = rows.map(r => {
      const { name, missing } = buildName(r, parsed.parts)
      const next = name ? name + extOf(r.track.name) : ''
      return { r, name, next, missing, problem: name ? '' : 'The new name would be empty' }
    })
    // Two files can't end up with the same name in the same folder.
    const seen = new Map<string, number>()
    for (const p of list) {
      if (!p.next) continue
      const key = (folderOf(p.r.track.path) + p.next).toLowerCase()
      seen.set(key, (seen.get(key) ?? 0) + 1)
    }
    for (const p of list) {
      if (!p.next || p.problem) continue
      const key = (folderOf(p.r.track.path) + p.next).toLowerCase()
      if ((seen.get(key) ?? 0) > 1) p.problem = 'Another selected file would get the same name'
      else if (otherPaths.has(key)) p.problem = 'Another file in the list already has this name'
    }
    return list
  }, [rows, parsed, otherPaths])

  const todo = preview.filter(p => !p.problem && p.next !== p.r.track.name)
  const problems = preview.filter(p => p.problem).length
  const same = preview.filter(p => !p.problem && p.next === p.r.track.name).length

  const insert = (chip: string) => {
    const el = input.current
    const token = `[${chip}]`
    const at = el?.selectionStart ?? format.length, end = el?.selectionEnd ?? at
    const next = format.slice(0, at) + token + format.slice(end)
    setFormat(next)
    requestAnimationFrame(() => { el?.focus(); el?.setSelectionRange(at + token.length, at + token.length) })
  }

  const ok = () => {
    if (busy || parsed.error || !todo.length) return
    onRename(todo.map(p => ({ id: p.r.id, name: p.name })), format)
  }

  let summary = ''
  if (!parsed.error) {
    summary = todo.length ? `${plural(todo.length, 'file')} will be renamed.` : 'No file names would change.'
    if (same) summary += ` ${same} already ${same === 1 ? 'has' : 'have'} this name.`
    if (problems) summary += ` ${problems} will be skipped.`
  }

  return (
    <div className="overlay" onMouseDown={e => { if (e.target === e.currentTarget && !busy) onClose() }}>
      <div className="dialog wide" role="dialog" aria-modal="true" aria-labelledby="rename-title">
        <div className="dialog-head">
          <h2 id="rename-title">Rename {plural(rows.length, 'file')} from tags</h2>
          <button className="icon-btn" aria-label="Close" onClick={onClose} disabled={!!busy}><IconClose /></button>
        </div>

        <label className="field-label" htmlFor="rename-format">New file name</label>
        <div className={'format-box' + (parsed.error ? ' bad' : '')}>
          <input
            id="rename-format"
            ref={input}
            autoFocus
            spellCheck={false}
            value={format}
            onChange={e => setFormat(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); ok() } }}
          />
          <span className="ext">.mp3</span>
        </div>
        <div className="chips" aria-label="Insert a field">
          {CHIPS.map(c => (
            <button key={c} type="button" className="chip" title={CHIP_HELP[c] ?? `Insert the ${c} tag`}
              onMouseDown={e => e.preventDefault()} onClick={() => insert(c)}>[{c}]</button>
          ))}
        </div>
        <p className="format-help">
          Write field names in square brackets. Any other text is kept as it is. [#] is the track number with two
          digits. The extension stays the same, and characters Windows doesn’t allow (such as : or ?) are replaced.
        </p>

        {parsed.error ? (
          <div className="format-error"><IconWarn />{parsed.error}</div>
        ) : (
          <div className="preview" aria-label="Preview">
            {preview.map(p => (
              <div key={p.r.id} className={'pv' + (p.problem ? ' bad' : p.next === p.r.track.name ? ' same' : '')}>
                <div className="pv-old" title={p.r.track.path}>{p.r.track.name}</div>
                <div className="pv-new">
                  <span className="arrow">→</span>
                  <span title={p.next}>{p.next || '—'}</span>
                </div>
                {(p.problem || p.missing.length > 0) && (
                  <div className="pv-note">
                    {p.problem || `Empty: ${p.missing.join(', ')}`}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        <div className="dialog-foot">
          <span className="summary">{busy ?? summary}</span>
          <button className="btn ghost" onClick={onClose} disabled={!!busy}>Cancel</button>
          <button className="btn primary" onClick={ok} disabled={!!busy || !!parsed.error || !todo.length}>
OK
          </button>
        </div>
      </div>
    </div>
  )
}
