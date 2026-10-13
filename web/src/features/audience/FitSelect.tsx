import { useLayoutEffect, useRef, type SelectHTMLAttributes } from 'react'

let measureCtx: CanvasRenderingContext2D | null = null

/**
 * Sizes a select to its chosen option, the way a native select sizes to its
 * widest option (text + padding, border and arrow): a select may offer every
 * field, but a row should be as compact as on the board. The width goes to
 * --fitw; `.in.fit` uses it (audience.css, imports.css).
 */
export function FitSelect({ label, className, ...rest }: SelectHTMLAttributes<HTMLSelectElement> & { label: string }) {
  const ref = useRef<HTMLSelectElement>(null)
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const fit = () => {
      measureCtx ??= document.createElement('canvas').getContext('2d')
      if (!measureCtx) return
      measureCtx.font = getComputedStyle(el).font
      // Chrome sizes a native select to floor(text width) + 39px (padding,
      // border and arrow): 97.4px of text makes a 136px select on the board.
      el.style.setProperty('--fitw', `${Math.floor(measureCtx.measureText(label).width) + 39}px`)
    }
    fit()
    void document.fonts?.ready.then(fit)
  }, [label])
  return <select ref={ref} className={`${className ?? ''} fit`} {...rest} />
}
