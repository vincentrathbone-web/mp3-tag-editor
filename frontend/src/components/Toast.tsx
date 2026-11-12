import { useEffect } from 'react'

export interface ToastData {
  id: number
  text: string
  error?: boolean
  action?: { label: string; run: () => void }
}

export default function Toast({ toast, onClose }: { toast: ToastData | null; onClose: () => void }) {
  useEffect(() => {
    if (!toast) return
    const t = setTimeout(onClose, toast.action ? 7000 : toast.error ? 8000 : 4200)
    return () => clearTimeout(t)
  }, [toast, onClose])
  if (!toast) return null
  return (
    <div className={'toast' + (toast.error ? ' err' : '')} role="status">
      <span className={toast.error ? 't-err' : 't-ok'}>{toast.error ? '✕' : '✓'}</span>
      <span>{toast.text}</span>
      {toast.action && <button onClick={() => { toast.action!.run(); onClose() }}>{toast.action.label}</button>}
      <button className="t-close" aria-label="Dismiss" onClick={onClose}>✕</button>
    </div>
  )
}
