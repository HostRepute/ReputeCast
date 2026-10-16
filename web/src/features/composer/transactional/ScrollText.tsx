import { useEffect, useRef, useState, type ReactNode } from 'react'

/**
 * A block of code or text that may scroll. While it scrolls it takes
 * keyboard focus, so it can be read without a mouse (WCAG 2.1.1), as the
 * kit's ScrollPre does sideways; this one also scrolls down.
 */
export function ScrollText({ className, label, children }: { className?: string; label?: string; children: ReactNode }) {
  const ref = useRef<HTMLPreElement>(null)
  const [over, setOver] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const check = () => setOver(el.scrollHeight > el.clientHeight + 1 || el.scrollWidth > el.clientWidth + 1)
    check()
    const ro = new ResizeObserver(check)
    ro.observe(el)
    const mo = new MutationObserver(check)
    mo.observe(el, { childList: true, subtree: true, characterData: true })
    return () => {
      ro.disconnect()
      mo.disconnect()
    }
  }, [])
  return (
    // A scrolling region is the exception to the rule: it must take focus.
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <pre ref={ref} className={className} aria-label={label} tabIndex={over ? 0 : undefined}>
      {children}
    </pre>
  )
}
