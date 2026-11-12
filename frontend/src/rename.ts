// Builds new file names from tags with a format like "[Title] - [#] - [Artist]".
import { FIELDS } from './fields'
import { Row, val } from './model'

type Part = { text: string } | { field: string; label: string }

export interface ParsedFormat {
  parts: Part[]
  error: string
}

// Extra names people use for a field, on top of each field's own label ("Album artist", "BPM"…).
const ALIASES: Record<string, string> = {
  '#': 'track', 'track #': 'track', 'track number': 'track', 'song title': 'title', 'song': 'title',
  'name': 'title', 'band': 'albumArtist', 'disc #': 'disc', 'disc number': 'disc', 'date': 'year',
}
const NAMES: Record<string, string> = {
  ...Object.fromEntries(FIELDS.map(f => [f.label.toLowerCase(), f.key])),
  ...Object.fromEntries(FIELDS.map(f => [f.key.toLowerCase(), f.key])),
  ...ALIASES,
  'filename': 'filename', 'file name': 'filename',
}

// The placeholders offered as buttons in the dialog.
export const CHIPS = ['Title', 'Artist', 'Album', '#', 'Album artist', 'Year', 'Disc', 'Genre', 'Filename']

export const CHIP_HELP: Record<string, string> = {
  '#': 'Track number with two digits, e.g. 03',
  'Disc': 'Disc number, e.g. 1',
  'Filename': 'The current file name',
}

export function parseFormat(format: string): ParsedFormat {
  const parts: Part[] = []
  const re = /\[([^\]]*)\]/g
  let last = 0
  for (let m = re.exec(format); m; m = re.exec(format)) {
    if (m.index > last) parts.push({ text: format.slice(last, m.index) })
    const label = m[1].trim()
    const field = NAMES[label.toLowerCase()]
    if (!field) return { parts, error: `“[${label}]” isn’t a tag name. Use one of the buttons below.` }
    parts.push({ field, label })
    last = re.lastIndex
  }
  const rest = format.slice(last)
  if (/[[\]]/.test(rest)) return { parts, error: 'A square bracket isn’t closed. Write fields like [Title].' }
  if (rest) parts.push({ text: rest })
  if (!parts.some(p => 'field' in p)) return { parts, error: 'Add at least one field, such as [Title].' }
  return { parts, error: '' }
}

// "3/12" → "03"; "1/2" → "1" for discs.
const numberPart = (v: string) => (v.match(/\d+/)?.[0] ?? '').replace(/^0+(?=\d)/, '')

function fieldText(r: Row, field: string): string {
  if (field === 'filename') return r.track.name.replace(/\.[^.]+$/, '')
  const v = val(r, field).replace(/\s+/g, ' ').trim()
  if (field === 'track') { const n = numberPart(v); return n ? n.padStart(2, '0') : '' }
  if (field === 'disc') return numberPart(v)
  if (field === 'year') return v.match(/\d{4}/)?.[0] ?? v
  return v
}

// Makes text safe for a Windows file name. Mirrors checkFileName in rename.go.
export function cleanName(s: string): string {
  return s
    .replace(/\s*:\s+/g, ' - ') // "Artist: Title" → "Artist - Title"
    .replace(/[\\/:|]/g, '-')
    .replace(/[<>"?*\x00-\x1f]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/[. ]+$/, '')
    .replace(/^(con|prn|aux|nul|com\d|lpt\d)$/i, '_$1')
    .slice(0, 240)
}

export interface NewName {
  name: string // without extension
  missing: string[] // labels of fields that were empty
}

export function buildName(r: Row, parts: Part[]): NewName {
  const missing: string[] = []
  const text = parts.map(p => {
    if ('text' in p) return p.text
    const v = fieldText(r, p.field)
    if (!v) missing.push(p.label)
    return v
  }).join('')
  // When a field is empty, tidy what's left around it: "Title -  - Artist" → "Title - Artist",
  // " - Artist" → "Artist", "Title ()" → "Title".
  const tidy = !missing.length ? text : text
    .replace(/\(\s*\)|\[\s*\]|\{\s*\}/g, '')
    .replace(/(\s+[-–_]\s+)(?:[-–_]\s+)+/g, '$1')
    .replace(/^[\s\-–_]+|[\s\-–_]+$/g, '')
  const name = cleanName(tidy)
  // Only dashes and dots left means every field was empty.
  return { name: /[\p{L}\p{N}]/u.test(name) ? name : '', missing: [...new Set(missing)] }
}

export const extOf = (name: string) => name.match(/\.[^.]+$/)?.[0] ?? ''
