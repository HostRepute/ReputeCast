const thousands = (n: number) => n.toLocaleString('en-US')

/**
 * A number that counts up without moving anything around it: the final
 * value reserves the width (hidden), the animated value is drawn over it,
 * and screen readers only ever hear the final value.
 */
export function Count({ final, p, live, format = thousands }: { final: number; p: number; live: boolean; format?: (n: number) => string }) {
  const shown = format(Math.round(final * p))
  const end = format(final)
  return (
    <span className="count" data-live={live ? '' : undefined}>
      <span className="count-w" aria-hidden="true">
        {end}
      </span>
      <span className="count-v" aria-hidden="true">
        {shown}
      </span>
      <span className="vh">{end}</span>
    </span>
  )
}

export const percent = (n: number) => `${n}%`
