import { main } from '../wailsjs/go/models'
import { FIELDS } from './fields'

export type Track = main.Track
export type Cover = main.Cover
export type Tags = Record<string, string>

// What the user wants to do with the cover; nothing happens to the file until Save.
export type CoverEdit = { mode: 'keep' } | { mode: 'set'; cover: Cover } | { mode: 'remove' }

// One line in the table: the file as it is on disk (track) plus unsaved edits.
export interface Row {
  id: number
  track: Track
  tags: Tags
  cover: CoverEdit
}

export interface Editing {
  ids: number[]
  key: string
  where: 'table' | 'pane'
  token: number
}

export const val = (r: Row, k: string) => r.tags[k] ?? ''
export const fieldChanged = (r: Row, k: string) => (r.tags[k] ?? '') !== (r.track.tags?.[k] ?? '')
export const isDirty = (r: Row) => r.cover.mode !== 'keep' || FIELDS.some(f => fieldChanged(r, f.key))

export function shownCover(r: Row): Cover | null {
  if (r.cover.mode === 'set') return r.cover.cover
  if (r.cover.mode === 'remove') return null
  return r.track.cover ?? null
}

export const fmtBytes = (b?: number | null) =>
  b == null ? '—' : b < 1024 ? `${b} B` : b < 1048576 ? `${Math.round(b / 1024)} KB` : `${(b / 1048576).toFixed(1)} MB`

export const fmtDuration = (s?: number | null) => {
  if (!s) return '—'
  const total = Math.round(s)
  const h = Math.floor(total / 3600), m = Math.floor((total % 3600) / 60), sec = String(total % 60).padStart(2, '0')
  return h ? `${h}:${String(m).padStart(2, '0')}:${sec}` : `${m}:${sec}`
}

export const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? '' : 's'}`
