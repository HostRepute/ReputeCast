export type DiffLine = { kind: 'same' | 'add' | 'del' | 'fold'; text: string }

/**
 * Line diff of two texts (longest common subsequence), for comparing
 * versions. Unchanged runs longer than `context * 2` lines are folded into
 * one "same" line that says how many lines it hides.
 */
export function diffLines(a: string, b: string, context = 3): DiffLine[] {
  const x = a.split('\n')
  const y = b.split('\n')
  // Trim the common head and tail first: versions usually differ in a few places.
  let head = 0
  while (head < x.length && head < y.length && x[head] === y[head]) head++
  let tail = 0
  while (tail < x.length - head && tail < y.length - head && x[x.length - 1 - tail] === y[y.length - 1 - tail]) tail++
  const xs = x.slice(head, x.length - tail)
  const ys = y.slice(head, y.length - tail)
  const n = xs.length
  const m = ys.length
  const lcs: Uint16Array[] = Array.from({ length: n + 1 }, () => new Uint16Array(m + 1))
  for (let i = n - 1; i >= 0; i--) for (let j = m - 1; j >= 0; j--) lcs[i]![j] = xs[i] === ys[j] ? lcs[i + 1]![j + 1]! + 1 : Math.max(lcs[i + 1]![j]!, lcs[i]![j + 1]!)
  const middle: DiffLine[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (xs[i] === ys[j]) {
      middle.push({ kind: 'same', text: xs[i]! })
      i++
      j++
    } else if (lcs[i + 1]![j]! >= lcs[i]![j + 1]!) middle.push({ kind: 'del', text: xs[i++]! })
    else middle.push({ kind: 'add', text: ys[j++]! })
  }
  while (i < n) middle.push({ kind: 'del', text: xs[i++]! })
  while (j < m) middle.push({ kind: 'add', text: ys[j++]! })
  const all: DiffLine[] = [...x.slice(0, head).map((t) => ({ kind: 'same' as const, text: t })), ...middle, ...y.slice(y.length - tail).map((t) => ({ kind: 'same' as const, text: t }))]
  // Fold long unchanged runs.
  const out: DiffLine[] = []
  let run: DiffLine[] = []
  const flush = (last: boolean) => {
    const first = out.length === 0
    if (run.length > context * 2 + 1) {
      const keepHead = first ? 0 : context
      const keepTail = last ? 0 : context
      out.push(...run.slice(0, keepHead))
      out.push({ kind: 'fold', text: `… ${run.length - keepHead - keepTail} unchanged lines` })
      out.push(...run.slice(run.length - keepTail))
    } else out.push(...run)
    run = []
  }
  for (const l of all) {
    if (l.kind === 'same') run.push(l)
    else {
      flush(false)
      out.push(l)
    }
  }
  flush(true)
  return out
}
