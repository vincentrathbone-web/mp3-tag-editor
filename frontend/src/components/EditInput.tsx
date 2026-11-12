import { useEffect, useRef } from 'react'
import { FIELD } from '../fields'

interface Props {
  initial: string
  fieldKey: string
  token: number
  valueRef: React.MutableRefObject<string>
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onBlur: (token: number) => void
}

// The text box shown while a value is being edited, in the table or in the side pane.
export default function EditInput({ initial, fieldKey, token, valueRef, onKeyDown, onBlur }: Props) {
  const ref = useRef<HTMLInputElement>(null)
  useEffect(() => {
    valueRef.current = initial
    ref.current?.focus()
    ref.current?.select()
    // Only when a new edit starts.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])
  return (
    <input
      ref={ref}
      className="edit"
      defaultValue={initial}
      list={fieldKey === 'genre' ? 'genres' : undefined}
      spellCheck={false}
      aria-label={FIELD[fieldKey]?.label}
      onChange={e => { valueRef.current = e.target.value }}
      onKeyDown={onKeyDown}
      onBlur={() => onBlur(token)}
    />
  )
}
