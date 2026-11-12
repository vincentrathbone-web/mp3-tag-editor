import { useEffect, useRef, useState } from 'react'
import { DEFAULT_COLUMNS, FIELDS } from '../fields'
import { IconColumns } from '../icons'

interface Props {
  cols: string[]
  onChange: (cols: string[]) => void
}

// The "Columns" button and its list of every ID3 field.
export default function ColumnsMenu({ cols, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const wrap = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => { if (!wrap.current?.contains(e.target as Node)) setOpen(false) }
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('mousedown', close)
    document.addEventListener('keydown', esc)
    return () => { document.removeEventListener('mousedown', close); document.removeEventListener('keydown', esc) }
  }, [open])

  const toggle = (key: string, on: boolean) =>
    onChange(on ? FIELDS.map(f => f.key).filter(k => k === key || cols.includes(k)) : cols.filter(k => k !== key))

  return (
    <div className="cols-wrap" ref={wrap}>
      <button className="btn" aria-expanded={open} onClick={() => setOpen(o => !o)}><IconColumns />Columns</button>
      {open && (
        <div className="pop">
          <div className="pop-head">
            <span>{cols.length} of {FIELDS.length} columns shown</span>
            <button onClick={() => onChange(DEFAULT_COLUMNS)}>Reset</button>
          </div>
          {FIELDS.map(f => (
            <label key={f.key}>
              <input type="checkbox" checked={cols.includes(f.key)} onChange={e => toggle(f.key, e.target.checked)} />
              {f.label}<span className="fid">{f.frame}</span>
            </label>
          ))}
        </div>
      )}
    </div>
  )
}
