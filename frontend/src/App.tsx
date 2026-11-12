import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  ConvertCoverData, ConvertCoverFile, EventsOn, GetCover, IMAGE_EXT, LoadPaths, OnFileDrop, OnFileDropOff,
  PickCover, PickFiles, PickFolder, SaveTrack, SetUnsavedCount, errorText, isDesktop, readAsDataURL,
} from './api'
import ColumnsMenu from './components/ColumnsMenu'
import DropBlock from './components/DropBlock'
import EditInput from './components/EditInput'
import FileTable from './components/FileTable'
import SidePane from './components/SidePane'
import Toast, { ToastData } from './components/Toast'
import { DEFAULT_COLUMNS, FIELD, GENRES } from './fields'
import { IconClose, IconMusic, IconPlus, IconRemove, IconSave, IconSearch, IconUndo } from './icons'
import { Cover, Editing, Row, Track, fmtBytes, isDirty, plural, val } from './model'

const COLS_KEY = 'mp3te.columns'

function loadColumns(): string[] {
  try {
    const v = JSON.parse(localStorage.getItem(COLS_KEY) ?? 'null')
    if (Array.isArray(v) && v.every(k => typeof k === 'string' && FIELD[k])) return v
  } catch { /* use the default */ }
  return DEFAULT_COLUMNS
}

const coverKey = (r: Row) => `${r.track.path}|${r.track.modified}|${r.track.size}`

export default function App() {
  // rowsRef / selectedRef always hold the latest values, so event handlers never act on stale data.
  const [rows, setRowsState] = useState<Row[]>([])
  const rowsRef = useRef<Row[]>([])
  const setRows = useCallback((fn: (rs: Row[]) => Row[]) => {
    const next = fn(rowsRef.current)
    if (next !== rowsRef.current) { rowsRef.current = next; setRowsState(next) }
  }, [])

  const [selected, setSelectedState] = useState<Set<number>>(new Set())
  const selectedRef = useRef<Set<number>>(new Set())
  const setSelected = useCallback((s: Set<number>) => { selectedRef.current = s; setSelectedState(s) }, [])
  const anchor = useRef<number | null>(null)

  const [editing, setEditingState] = useState<Editing | null>(null)
  const editingRef = useRef<Editing | null>(null)
  const setEditing = useCallback((e: Editing | null) => { editingRef.current = e; setEditingState(e) }, [])
  const editValue = useRef('')
  const token = useRef(0)
  const nextId = useRef(1)

  const [sort, setSort] = useState<{ key: string | null; dir: 1 | -1 }>({ key: null, dir: 1 })
  const [filter, setFilter] = useState('')
  const [cols, setColsState] = useState<string[]>(loadColumns)
  const [dialog, setDialog] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)
  const busyRef = useRef(false)
  const [coverBusy, setCoverBusy] = useState(false)
  const [toast, setToast] = useState<ToastData | null>(null)
  const [fullCovers, setFullCovers] = useState<Record<string, string>>({})

  const showToast = useCallback((text: string, error = false, action?: ToastData['action']) =>
    setToast({ id: Date.now(), text, error, action }), [])
  const closeToast = useCallback(() => setToast(null), [])

  const setCols = (c: string[]) => {
    setColsState(c)
    try { localStorage.setItem(COLS_KEY, JSON.stringify(c)) } catch { /* not important */ }
    const ed = editingRef.current
    if (ed?.where === 'table' && !c.includes(ed.key)) setEditing(null)
  }

  /* ---------- Visible rows (filter + sort) ---------- */
  const visible = useMemo(() => {
    let list = rows
    if (filter) {
      const q = filter.toLowerCase()
      list = list.filter(r => [r.track.name, r.tags.title, r.tags.artist, r.tags.album, r.tags.albumArtist, r.tags.genre]
        .some(v => (v ?? '').toLowerCase().includes(q)))
    }
    if (sort.key) {
      const k = sort.key
      const num = k !== 'name' && FIELD[k].kind === 'num'
      const get = (r: Row) => (k === 'name' ? r.track.name : val(r, k))
      list = [...list].sort((a, b) => {
        const x = get(a), y = get(b)
        if (!x && y) return 1
        if (x && !y) return -1
        const d = num ? (parseFloat(x) || 0) - (parseFloat(y) || 0) : x.localeCompare(y, undefined, { numeric: true, sensitivity: 'base' })
        return d * sort.dir
      })
    }
    return list
  }, [rows, filter, sort])
  const visibleRef = useRef(visible)
  visibleRef.current = visible

  const targets = useMemo(() => rows.filter(r => selected.has(r.id)), [rows, selected])
  const sideRef = useRef<HTMLElement>(null)
  useEffect(() => { sideRef.current?.scrollTo({ top: 0 }) }, [selected])
  const dirtyCount = useMemo(() => rows.filter(isDirty).length, [rows])

  /* ---------- Editing ---------- */
  const commitEdit = useCallback(() => {
    const ed = editingRef.current
    if (!ed) return
    setEditing(null)
    const v = editValue.current.trim()
    setRows(rs => {
      const tg = rs.filter(r => ed.ids.includes(r.id))
      const mixed = ed.where === 'pane' && new Set(tg.map(r => val(r, ed.key))).size > 1
      if (mixed && v === '') return rs // left empty: every file keeps its own value
      let changed = false
      const next = rs.map(r => {
        if (!ed.ids.includes(r.id) || val(r, ed.key) === v) return r
        changed = true
        const tags = { ...r.tags }
        if (v) tags[ed.key] = v
        else delete tags[ed.key]
        return { ...r, tags }
      })
      return changed ? next : rs
    })
  }, [setEditing, setRows])

  const beginEdit = useCallback((ids: number[], key: string, where: Editing['where']) => {
    if (busyRef.current || !ids.length) return
    token.current++
    setEditing({ ids, key, where, token: token.current })
  }, [setEditing])

  const onEditBlur = useCallback((t: number) => {
    if (editingRef.current?.token === t) commitEdit()
  }, [commitEdit])

  const moveEdit = (ed: Editing, dx: number, dy: number) => {
    const list = visibleRef.current
    let r = list.findIndex(x => x.id === ed.ids[0])
    let c = cols.indexOf(ed.key) + dx
    r += dy
    if (c >= cols.length) { c = 0; r++ }
    if (c < 0) { c = cols.length - 1; r-- }
    if (r < 0 || r >= list.length || c < 0) return
    const id = list[r].id
    setSelected(new Set([id]))
    anchor.current = id
    beginEdit([id], cols[c], 'table')
  }

  const onEditKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    const ed = editingRef.current
    if (!ed) return
    if (e.key === 'Enter') {
      e.preventDefault()
      commitEdit()
      if (ed.where === 'table') moveEdit(ed, 0, 1)
    } else if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      setEditing(null)
    } else if (e.key === 'Tab' && ed.where === 'table') {
      e.preventDefault()
      commitEdit()
      moveEdit(ed, e.shiftKey ? -1 : 1, 0)
    }
  }

  const editor = (key: string, initial: string) => editing && (
    <EditInput key={editing.token} initial={initial} fieldKey={key} token={editing.token}
      valueRef={editValue} onKeyDown={onEditKey} onBlur={onEditBlur} />
  )

  /* ---------- Selection ---------- */
  const onCellMouseDown = (e: React.MouseEvent, row: Row, key: string | null) => {
    if (e.button !== 0 || (e.target as HTMLElement).closest('input.edit')) return
    e.preventDefault()
    commitEdit()
    const id = row.id
    if (e.shiftKey && anchor.current != null) {
      const list = visibleRef.current
      const a = list.findIndex(r => r.id === anchor.current), b = list.findIndex(r => r.id === id)
      const s = new Set(e.ctrlKey || e.metaKey ? selectedRef.current : [])
      list.slice(Math.min(a, b), Math.max(a, b) + 1).forEach(r => s.add(r.id))
      setSelected(s)
      return
    }
    if (e.ctrlKey || e.metaKey) {
      const s = new Set(selectedRef.current)
      if (s.has(id)) s.delete(id)
      else s.add(id)
      anchor.current = id
      setSelected(s)
      return
    }
    setSelected(new Set([id]))
    anchor.current = id
    if (key) beginEdit([id], key, 'table')
  }

  const onSort = (k: string) => {
    commitEdit()
    setSort(s => (s.key !== k ? { key: k, dir: 1 } : s.dir === 1 ? { key: k, dir: -1 } : { key: null, dir: 1 }))
  }

  /* ---------- Adding music ---------- */
  const addTracks = useCallback((tracks: Track[], fromDrop: boolean) => {
    if (!tracks.length) {
      if (fromDrop) showToast('No MP3 files were found in what you dropped.', true)
      return
    }
    const known = new Set(rowsRef.current.map(r => r.track.path.toLowerCase()))
    const fresh = tracks.filter(t => !known.has(t.path.toLowerCase()))
    setDialog(false)
    if (!fresh.length) { showToast('Those files are already in the list.'); return }
    const added: Row[] = fresh.map(t => ({ id: nextId.current++, track: t, tags: { ...(t.tags ?? {}) }, cover: { mode: 'keep' } }))
    setRows(rs => [...rs, ...added])
    setSelected(new Set([added[0].id]))
    anchor.current = added[0].id
    const problems = fresh.filter(t => t.problem).length
    showToast(`Added ${plural(fresh.length, 'file')}.` +
      (problems ? ` ${plural(problems, 'file')} can't be saved. Select a file marked with ⚠ to see why.` : ''))
  }, [setRows, setSelected, showToast])

  const runLoad = useCallback(async (load: () => Promise<Track[]>, fromDrop = false) => {
    if (busyRef.current) return
    commitEdit()
    busyRef.current = true
    setBusy('Reading files…')
    try {
      addTracks((await load()) ?? [], fromDrop)
    } catch (e) {
      showToast(errorText(e), true)
    } finally {
      busyRef.current = false
      setBusy(null)
    }
  }, [addTracks, commitEdit, showToast])

  const addFolder = () => runLoad(PickFolder)
  const addFiles = () => runLoad(PickFiles)

  useEffect(() => EventsOn('load-progress', (done: number, total: number) =>
    setBusy(`Reading ${done} of ${total} files…`)), [])

  /* ---------- Cover art ---------- */
  const applyCover = useCallback(async (get: () => Promise<Cover | null>) => {
    const ids = [...selectedRef.current]
    if (!ids.length) { showToast('Select one or more files first, then add the cover.', true); return }
    commitEdit()
    setCoverBusy(true)
    try {
      const c = await get()
      if (!c) return
      setRows(rs => rs.map(r => (ids.includes(r.id) ? { ...r, cover: { mode: 'set', cover: c } } : r)))
      showToast(`Cover ready: JPEG ${c.width} × ${c.height}, ${fmtBytes(c.bytes)}, for ${plural(ids.length, 'file')}. Save to write it.`)
    } catch (e) {
      showToast(errorText(e), true)
    } finally {
      setCoverBusy(false)
    }
  }, [commitEdit, setRows, showToast])

  const removeCover = () => {
    commitEdit()
    const ids = selectedRef.current
    setRows(rs => rs.map(r => {
      if (!ids.has(r.id)) return r
      return { ...r, cover: r.track.cover ? { mode: 'remove' } : { mode: 'keep' } }
    }))
  }

  // Load the full-size cover for the side pane (the table only gets small thumbnails).
  useEffect(() => {
    const first = targets[0]
    if (!first || first.cover.mode !== 'keep' || !first.track.cover || fullCovers[coverKey(first)]) return
    let alive = true
    GetCover(first.track.path)
      .then(c => { if (alive && c?.dataUrl) setFullCovers(m => ({ ...m, [coverKey(first)]: c.dataUrl! })) })
      .catch(() => { /* the thumbnail is shown instead */ })
    return () => { alive = false }
  }, [targets, fullCovers])

  /* ---------- Drag and drop ---------- */
  const handlers = useRef({ runLoad, applyCover, showToast })
  handlers.current = { runLoad, applyCover, showToast }

  useEffect(() => {
    if (isDesktop) {
      // In the app window Wails tells us the real paths of dropped files and folders.
      OnFileDrop((x, y, paths) => {
        const h = handlers.current
        const el = document.elementFromPoint(x, y) as HTMLElement | null
        if (el?.closest('[data-drop="cover"]')) {
          const img = paths.find(p => IMAGE_EXT.test(p))
          if (img) h.applyCover(() => ConvertCoverFile(img))
          else h.showToast('That isn’t an image. Drop an AVIF, WebP, PNG, JPG, GIF, BMP or TIFF file.', true)
        } else {
          h.runLoad(() => LoadPaths(paths), true)
        }
      }, true)
      return () => OnFileDropOff()
    }
    // In a normal browser we only get file contents, not paths: enough for cover art.
    const over = (e: DragEvent) => { if (e.dataTransfer?.types.includes('Files')) e.preventDefault() }
    const drop = async (e: DragEvent) => {
      if (!e.dataTransfer?.types.includes('Files')) return
      e.preventDefault()
      const h = handlers.current
      const file = [...e.dataTransfer.files].find(f => IMAGE_EXT.test(f.name) || f.type.startsWith('image/'))
      if ((e.target as HTMLElement).closest?.('[data-drop="cover"]') && file) {
        h.applyCover(async () => ConvertCoverData(file.name, await readAsDataURL(file)))
        return
      }
      h.showToast('A browser can’t see where dropped files are stored. Use Add folder or Add file(s), or drop music in the desktop window.', true)
    }
    window.addEventListener('dragover', over)
    window.addEventListener('drop', drop)
    return () => { window.removeEventListener('dragover', over); window.removeEventListener('drop', drop) }
  }, [])

  // Developer helper (dev mode only): window.__mp3te.addPaths(['D:\\Music']) adds files without the picker.
  useEffect(() => {
    if (!import.meta.env.DEV) return
    (window as any).__mp3te = { addPaths: (p: string[]) => handlers.current.runLoad(() => LoadPaths(p), true) }
  }, [])

  /* ---------- Save, revert, remove ---------- */
  const saveRows = async (ids: number[] | null) => {
    if (busyRef.current) return
    commitEdit()
    const list = rowsRef.current.filter(r => (ids === null || ids.includes(r.id)) && isDirty(r))
    if (!list.length) return
    busyRef.current = true
    let ok = 0
    const failed: string[] = []
    for (const [i, r] of list.entries()) {
      setBusy(`Saving ${i + 1} of ${list.length}…`)
      const cur = rowsRef.current.find(x => x.id === r.id)
      if (!cur) continue
      try {
        const t = await SaveTrack({
          path: cur.track.path,
          tags: cur.tags,
          cover: cur.cover.mode,
          coverData: cur.cover.mode === 'set' ? cur.cover.cover.dataUrl ?? '' : '',
        })
        setRows(rs => rs.map(x => (x.id === r.id ? { ...x, track: t, tags: { ...(t.tags ?? {}) }, cover: { mode: 'keep' } } : x)))
        ok++
      } catch (e) {
        failed.push(`${cur.track.name}: ${errorText(e)}`)
      }
    }
    busyRef.current = false
    setBusy(null)
    if (failed.length) {
      showToast(`Saved ${ok} of ${list.length}. Couldn't save ${failed[0]}${failed.length > 1 ? ` (and ${failed.length - 1} more)` : ''}`, true)
    } else {
      showToast(`Saved ${plural(ok, 'file')} as ID3v2.3.`)
    }
  }

  const revert = () => {
    commitEdit()
    const ids = selectedRef.current
    const n = rowsRef.current.filter(r => ids.has(r.id) && isDirty(r)).length
    setRows(rs => rs.map(r => (ids.has(r.id) && isDirty(r) ? { ...r, tags: { ...(r.track.tags ?? {}) }, cover: { mode: 'keep' } } : r)))
    showToast(`Undid the unsaved changes in ${plural(n, 'file')}.`)
  }

  const remove = () => {
    commitEdit()
    const ids = selectedRef.current
    const before = rowsRef.current
    const gone = before.filter(r => ids.has(r.id))
    const unsaved = gone.filter(isDirty).length
    setRows(rs => rs.filter(r => !ids.has(r.id)))
    setSelected(new Set())
    showToast(
      `Removed ${plural(gone.length, 'file')} from the list${unsaved ? `, including ${unsaved} with unsaved changes` : ''}. The files on disk are untouched.`,
      false,
      {
        label: 'Undo',
        // Put the removed files back in their old places, keeping any edits made since.
        run: () => setRows(rs => {
          const current = new Map(rs.map(r => [r.id, r]))
          const restored = before.flatMap(r => current.get(r.id) ?? (ids.has(r.id) ? [r] : []))
          const later = rs.filter(r => !before.some(b => b.id === r.id))
          return [...restored, ...later]
        }),
      },
    )
  }

  /* ---------- Keyboard + window ---------- */
  const keys = useRef({ saveAll: () => saveRows(null), selectAll: () => {} })
  keys.current = {
    saveAll: () => saveRows(null),
    selectAll: () => setSelected(new Set(visibleRef.current.map(r => r.id))),
  }
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const typing = /INPUT|SELECT|TEXTAREA/.test((e.target as HTMLElement).tagName)
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') { e.preventDefault(); keys.current.saveAll() }
      else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a' && !typing) { e.preventDefault(); keys.current.selectAll() }
      else if (e.key === 'Escape') setDialog(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  useEffect(() => { SetUnsavedCount(dirtyCount).catch(() => { /* not running under Wails */ }) }, [dirtyCount])
  useEffect(() => {
    if (isDesktop) return // the Go side asks before closing
    const warn = (e: BeforeUnloadEvent) => { if (dirtyCount) e.preventDefault() }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirtyCount])

  const selDirty = targets.some(isDirty)
  const saving = busy?.startsWith('Saving')

  return (
    <>
      <div className="app">
        <header className="toolbar">
          <div className="brand">
            <div className="mark"><IconMusic /></div>
            <span className="brand-name">MP3 Tag Editor</span>
          </div>
          <button className="btn primary" onClick={() => { commitEdit(); setDialog(true) }} disabled={!!busy}><IconPlus />Add music</button>
          <span className="sep" />
          <button className="btn" onClick={() => saveRows([...selected])} disabled={!selDirty || !!busy} title="Save the selected files">
            <IconSave />Save
          </button>
          <button className={'btn' + (dirtyCount ? ' primary' : '')} onClick={() => saveRows(null)} disabled={!dirtyCount || !!busy} title="Save every changed file (Ctrl+S)">
            Save all <span className="badge">{dirtyCount}</span>
          </button>
          <button className="btn ghost" onClick={revert} disabled={!selDirty || !!busy} title="Undo unsaved changes on the selected files">
            <IconUndo />Revert
          </button>
          <span className="spacer" />
          <label className="search">
            <IconSearch />
            <input type="search" placeholder="Filter files" value={filter} onChange={e => { commitEdit(); setFilter(e.target.value.trim()) }} />
          </label>
          <ColumnsMenu cols={cols} onChange={setCols} />
          <button className="btn ghost" onClick={remove} disabled={!selected.size || !!busy} title="Remove the selected files from the list. Files on disk are not touched.">
            <IconRemove />Remove
          </button>
        </header>

        <main className="main">
          <section className="panel table-panel" aria-label="Files">
            {rows.length ? (
              <FileTable rows={visible} cols={cols} selected={selected} editing={editing} sort={sort} filter={filter}
                renderEditor={(r, k) => editor(k, val(r, k))} onCellMouseDown={onCellMouseDown} onSort={onSort} />
            ) : (
              <div className="empty"><div className="empty-inner"><DropBlock onAddFolder={addFolder} onAddFiles={addFiles} disabled={!!busy} /></div></div>
            )}
          </section>
          <aside className="panel side" ref={sideRef} aria-label="Cover art and file properties">
            <SidePane
              targets={targets}
              hasFiles={rows.length > 0}
              editing={editing}
              coverBusy={coverBusy}
              fullCover={r => fullCovers[coverKey(r)]}
              renderEditor={editor}
              onEdit={k => { commitEdit(); beginEdit([...selectedRef.current], k, 'pane') }}
              onPickCover={() => !coverBusy && applyCover(PickCover)}
              onRemoveCover={removeCover}
            />
          </aside>
        </main>

        <footer className="status">
          <div>
            <b>{plural(rows.length, 'file')}</b> · {selected.size} selected ·{' '}
            <span className={dirtyCount ? 'unsaved' : ''}><b>{dirtyCount}</b> unsaved</span>
            {busy && <span className="busy-text"> · {busy}</span>}
          </div>
          <div className="right">
            <span>Writes ID3v2.3</span>
            <span>Covers: JPEG ≤ 500×500</span>
            {!isDesktop && <span>Browser mode: drop music in the desktop window</span>}
          </div>
        </footer>
        <div className="drop-catch"><div>Drop to add music</div></div>
      </div>

      {dialog && (
        <div className="overlay" onMouseDown={e => { if (e.target === e.currentTarget) setDialog(false) }}>
          <div className="dialog" role="dialog" aria-modal="true" aria-labelledby="add-title">
            <div className="dialog-head">
              <h2 id="add-title">Add music</h2>
              <button className="icon-btn" aria-label="Close" onClick={() => setDialog(false)}><IconClose /></button>
            </div>
            <DropBlock onAddFolder={addFolder} onAddFiles={addFiles} disabled={!!busy} />
            {busy && !saving && <div className="dialog-busy">{busy}</div>}
          </div>
        </div>
      )}

      <Toast toast={toast} onClose={closeToast} />
      <datalist id="genres">{GENRES.map(g => <option key={g} value={g} />)}</datalist>
    </>
  )
}
