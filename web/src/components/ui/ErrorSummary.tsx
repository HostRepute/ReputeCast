import { useEffect, useRef } from 'react'

export interface FormError {
  /** The field's id (or name) to move focus to. */
  field: string
  /** What to fix, as a short sentence: "Reply-to address needs a domain". */
  message: string
}

function fieldElement(field: string): HTMLElement | null {
  return document.getElementById(field) ?? document.querySelector<HTMLElement>(`[name="${CSS.escape(field)}"]`)
}

/** Moves focus to a field and scrolls it into view. */
export function focusField(field: string): boolean {
  const el = fieldElement(field)
  if (!el) return false
  el.scrollIntoView?.({ block: 'center' })
  el.focus({ preventScroll: true })
  return true
}

/**
 * Form-level summary, shown on submit when several fields fail (Feedback
 * board). It takes focus when it appears, is announced once, and each link
 * moves focus to its field. Errors also stay next to their fields.
 */
export function ErrorSummary({ errors, title, autoFocus = true }: { errors: FormError[]; title?: string; autoFocus?: boolean }) {
  const ref = useRef<HTMLDivElement>(null)
  const signature = errors.map((e) => e.field).join('|')
  useEffect(() => {
    if (autoFocus && errors.length) ref.current?.focus()
    // re-focus when a new submit produces a different set of errors
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [signature, autoFocus])
  if (!errors.length) return null
  const n = errors.length
  return (
    <div ref={ref} className="errsum" role="alert" tabIndex={-1} aria-labelledby="errsum-title">
      <h2 className="est" id="errsum-title">
        {title ?? `${n === 1 ? '1 thing' : `${n} things`} to fix before saving`}
      </h2>
      <ul>
        {errors.map((e) => (
          <li key={e.field}>
            <a
              href={`#${e.field}`}
              onClick={(ev) => {
                if (focusField(e.field)) ev.preventDefault()
              }}
            >
              {e.message}
            </a>
          </li>
        ))}
      </ul>
    </div>
  )
}
