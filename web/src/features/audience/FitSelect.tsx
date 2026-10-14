import { useLayoutEffect, useRef, type SelectHTMLAttributes } from 'react'

/**
 * Sizes a select to its chosen option, the way a native select sizes to its
 * widest option (text + padding, border and arrow): a select may offer every
 * field, but a row should be as compact as on the board. A hidden native
 * select holding only that option is measured, so the width follows the
 * browser's own select metrics (they change between Chrome versions). The
 * width goes to --fitw; `.in.fit` uses it (audience.css, imports.css).
 */
export function FitSelect({ label, className, ...rest }: SelectHTMLAttributes<HTMLSelectElement> & { label: string }) {
  const ref = useRef<HTMLSelectElement>(null)
  useLayoutEffect(() => {
    const el = ref.current
    if (!el?.parentElement) return
    const fit = () => {
      const probe = document.createElement('select')
      probe.className = className ?? ''
      probe.setAttribute('aria-hidden', 'true')
      probe.tabIndex = -1
      probe.style.cssText = 'position:absolute;visibility:hidden;pointer-events:none;width:auto;min-width:0;max-width:none'
      probe.append(new Option(label))
      el.parentElement?.append(probe)
      el.style.setProperty('--fitw', `${probe.getBoundingClientRect().width}px`)
      probe.remove()
    }
    fit()
    void document.fonts?.ready.then(fit)
  }, [label, className])
  return <select ref={ref} className={`${className ?? ''} fit`} {...rest} />
}
