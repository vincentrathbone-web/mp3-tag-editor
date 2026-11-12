import { useEffect, useRef } from 'react'
import { FIELD } from '../fields'

interface Props {
  initial: string
  fieldKey: string
  token: number
  placeholder?: string
  /** How many files the value is for, when it is more than one. */
  count?: number
  valueRef: React.MutableRefObject<string>
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onBlur: (token: number) => void
}

// The text box shown while a value is being edited, in the table or in the side pane.
export default function EditInput({ initial, fieldKey, token, placeholder, count, valueRef, onKeyDown, onBlur }: Props) {
  const ref = useRef<HTMLInputElement>(null)
  useEffect(() => {
    valueRef.current = initial
    ref.current?.focus()
    ref.current?.select()
    // Only when a new edit starts.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])
  const input = (
    <input
      ref={ref}
      className={'edit' + (count ? ' bulk' : '')}
      defaultValue={initial}
      placeholder={placeholder}
      list={fieldKey === 'genre' ? 'genres' : undefined}
      spellCheck={false}
      aria-label={FIELD[fieldKey]?.label + (count ? ` for ${count} files` : '')}
      onChange={e => { valueRef.current = e.target.value }}
      onKeyDown={onKeyDown}
      onBlur={() => onBlur(token)}
    />
  )
  if (!count) return input
  return <div className="edit-wrap">{input}<span className="edit-count">{count} files</span></div>
}
