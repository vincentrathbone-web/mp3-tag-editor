import { FIELD } from '../fields'
import { IconImage, IconWarn } from '../icons'
import { Editing, Row, fieldChanged, isDirty, shownCover, val } from '../model'

interface Props {
  rows: Row[]
  cols: string[]
  selected: Set<number>
  editing: Editing | null
  sort: { key: string | null; dir: 1 | -1 }
  filter: string
  renderEditor: (r: Row, key: string) => React.ReactNode
  onCellMouseDown: (e: React.MouseEvent, row: Row, key: string | null) => void
  onSort: (key: string) => void
}

export default function FileTable({ rows, cols, selected, editing, sort, filter, renderEditor, onCellMouseDown, onSort }: Props) {
  const arrow = (k: string) => (sort.key === k ? <span className="arrow">{sort.dir > 0 ? '▲' : '▼'}</span> : null)
  return (
    <table>
      <thead>
        <tr>
          <th className="c-file" onMouseDown={e => { e.preventDefault(); onSort('name') }}>File{arrow('name')}</th>
          {cols.map(k => (
            <th key={k} onMouseDown={e => { e.preventDefault(); onSort(k) }}>
              {FIELD[k].label}<span className="fid">{FIELD[k].frame}</span>{arrow(k)}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.length === 0 && (
          <tr className="nomatch"><td colSpan={cols.length + 1}>No files match “{filter}”.</td></tr>
        )}
        {rows.map(r => {
          const cover = shownCover(r)
          return (
            <tr key={r.id} className={selected.has(r.id) ? 'sel' : ''}>
              <td className="c-file" onMouseDown={e => onCellMouseDown(e, r, null)}>
                <div className="filecell" title={r.track.path}>
                  <span className="thumb">{cover?.thumb ? <img src={cover.thumb} alt="" /> : <IconImage />}</span>
                  <span className="fname">{r.track.name}</span>
                  {r.track.problem && <span className="warn" title={r.track.problem}><IconWarn /></span>}
                  {isDirty(r) && <span className="dot" title="Unsaved changes" />}
                </div>
              </td>
              {cols.map(k => {
                const cls = [FIELD[k].kind, fieldChanged(r, k) ? 'changed' : ''].join(' ')
                const isEditing = editing?.where === 'table' && editing.ids[0] === r.id && editing.key === k
                const v = val(r, k)
                return (
                  <td key={k} className={cls} onMouseDown={e => onCellMouseDown(e, r, k)}>
                    {isEditing ? renderEditor(r, k) : (
                      <div className="cell" title={v}>{v ? v.replace(/\n/g, ' ⏎ ') : <span className="none">—</span>}</div>
                    )}
                  </td>
                )
              })}
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}
