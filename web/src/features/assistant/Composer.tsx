import { useEffect, useId, useLayoutEffect, useRef, type KeyboardEvent } from 'react'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { AsstIcon } from './icons'
import { m } from './messages'

export interface ComposerHandle {
  focus: () => void
}

/**
 * The composer (assistant brief §3.2): it grows with its text up to 40% of
 * the panel; Enter sends and Shift Enter adds a line, never while an input
 * method is composing a word (Telex, VNI, Japanese, Chinese, Korean commit
 * with Enter). While the assistant answers, Send becomes Stop.
 */
export function Composer({
  value,
  onChange,
  onSend,
  onStop,
  answering,
  blocked,
  placeholder,
  bound,
  handle,
}: {
  value: string
  onChange: (v: string) => void
  onSend: () => void
  onStop: () => void
  answering: boolean
  /** Why nothing can be sent here (a state of the assistant): Send says so. */
  blocked?: string
  placeholder: string
  /** The element whose height bounds the composer (the panel). */
  bound?: () => HTMLElement | null
  /** Receives what the panel may do to the composer (focus it). */
  handle?: (h: ComposerHandle | null) => void
}) {
  const t = useT()
  const area = useRef<HTMLTextAreaElement>(null)
  const id = useId()
  // A second click of Send right after it became Stop isn't a stop.
  const since = useRef(0)
  useLayoutEffect(() => {
    if (answering) since.current = Date.now()
  }, [answering])
  useEffect(() => {
    handle?.({
      focus: () => {
        const el = area.current
        if (!el) return
        el.focus({ preventScroll: true })
        el.setSelectionRange(el.value.length, el.value.length)
      },
    })
    return () => handle?.(null)
  }, [handle])
  // Grow with the text, up to 40% of the panel's height: measured only when
  // the text changes (measuring forces a layout of the panel, which a
  // streaming answer re-renders many times a second).
  const boundRef = useRef(bound)
  useLayoutEffect(() => {
    boundRef.current = bound
  })
  useLayoutEffect(() => {
    const el = area.current
    if (!el) return
    const limit = Math.max(44, (boundRef.current?.()?.clientHeight ?? window.innerHeight) * 0.4)
    el.style.height = 'auto'
    const next = Math.min(el.scrollHeight, limit)
    el.style.height = `${next}px`
    el.style.overflowY = el.scrollHeight > limit ? 'auto' : 'hidden'
  }, [value])
  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== 'Enter' || e.shiftKey) return
    // An input method uses Enter to commit a word: never send then.
    if (e.nativeEvent.isComposing || e.keyCode === 229) return
    e.preventDefault()
    if (!answering && !blocked) onSend()
  }
  return (
    <form
      className="asst-composer"
      onSubmit={(e) => {
        e.preventDefault()
        if (!answering && !blocked) onSend()
      }}
    >
      <label htmlFor={`${id}-draft`} className="vh">
        {t(m.composerLabel)}
      </label>
      <textarea
        ref={area}
        id={`${id}-draft`}
        rows={1}
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        aria-describedby={`${id}-hint`}
      />
      <div className="asst-composer-row">
        <span className="asst-hint" id={`${id}-hint`}>
          <span className="asst-hide-phone">{t(m.hint)}</span>
        </span>
        {/* One button that changes, so focus stays on it when Send becomes Stop. */}
        <Button
          type={answering ? 'button' : 'submit'}
          variant={answering ? 'default' : 'primary'}
          disabledReason={answering ? undefined : blocked}
          onClick={
            answering
              ? (e) => {
                  e.preventDefault()
                  if (Date.now() - since.current > 400) onStop()
                }
              : undefined
          }
        >
          <AsstIcon name={answering ? 'stop' : 'send'} />
          {answering ? t(m.stop) : t(m.send)}
        </Button>
      </div>
    </form>
  )
}
