import type { Data, TestData, Variable, VarType } from './types'
import { PREVIEW_TO } from './types'

/** A variable name the API accepts: letters, digits and underscores, not starting with a digit. */
export const NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/

/** Names ReputeCast reserves (bound at render time, or Liquid's own). */
export const RESERVED = new Set(['to', 'unsubscribe_url', 'preferences_url', 'forloop', 'tablerowloop'])

export function newVariable(name: string, type: VarType = 'text'): Variable {
  return { name, type, required: false, sensitive: false, sample: '' }
}

/** The sample as the value a call would send (numbers, yes/no, lists and objects parsed). */
export function sampleValue(v: Variable): unknown {
  const s = v.sample ?? ''
  if (s === '') return undefined
  switch (v.type) {
    case 'number': {
      const n = Number(s)
      return s.trim() !== '' && Number.isFinite(n) ? n : s
    }
    case 'boolean':
      return s === 'true' ? true : s === 'false' ? false : s
    case 'list':
    case 'object':
      try {
        return JSON.parse(s) as unknown
      } catch {
        return s
      }
  }
  return s
}

/** Sample values of every variable that has one. */
export function sampleData(vars: Variable[]): Data {
  const out: Data = {}
  for (const v of vars) {
    const val = sampleValue(v)
    if (val !== undefined) out[v.name] = val
  }
  return out
}

/** The data a preview renders with: the chosen test data set, or the samples. */
export function dataFor(sets: TestData[], name: string, vars: Variable[]): Data {
  const set = sets.find((s) => s.name === name)
  return set ? set.data : sampleData(vars)
}

/** A parsed test data set, or where its JSON breaks (`notObject`: valid JSON that isn't an object). */
export type JsonResult = { ok: true; data: Data } | { ok: false; line: number; message: string; notObject?: boolean }

/**
 * Where JSON stops being valid, as an offset (-1 when it is valid). The
 * browser's JSON.parse messages don't always say where.
 */
export function jsonErrorAt(s: string): number {
  let i = 0
  const fail = (): never => {
    throw i
  }
  const ws = () => {
    while (i < s.length && ' \t\n\r'.includes(s[i]!)) i++
  }
  const string = () => {
    i++
    while (i < s.length && s[i] !== '"') {
      if (s[i] === '\n') fail()
      if (s[i] === '\\') i++
      i++
    }
    if (s[i] !== '"') fail()
    i++
  }
  const value = (): void => {
    ws()
    const c = s[i]
    if (c === '{') {
      i++
      ws()
      if (s[i] === '}') {
        i++
        return
      }
      for (;;) {
        ws()
        if (s[i] !== '"') fail()
        string()
        ws()
        if (s[i] !== ':') fail()
        i++
        value()
        ws()
        if (s[i] === ',') {
          i++
          continue
        }
        if (s[i] === '}') {
          i++
          return
        }
        fail()
      }
    }
    if (c === '[') {
      i++
      ws()
      if (s[i] === ']') {
        i++
        return
      }
      for (;;) {
        value()
        ws()
        if (s[i] === ',') {
          i++
          continue
        }
        if (s[i] === ']') {
          i++
          return
        }
        fail()
      }
    }
    if (c === '"') return string()
    const m = /^(?:-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null)/.exec(s.slice(i))
    if (!m) fail()
    i += m![0].length
  }
  try {
    value()
    ws()
    if (i < s.length) fail()
    return -1
  } catch (at) {
    return typeof at === 'number' ? at : 0
  }
}

/** Parses a test data set: a JSON object. Errors carry their line. */
export function parseJsonObject(text: string): JsonResult {
  try {
    const v = JSON.parse(text) as unknown
    if (!v || typeof v !== 'object' || Array.isArray(v)) return { ok: false, line: 1, message: '', notObject: true }
    return { ok: true, data: v as Data }
  } catch (e) {
    const at = jsonErrorAt(text)
    return { ok: false, line: text.slice(0, Math.max(0, at)).split('\n').length, message: e instanceof Error ? e.message : String(e) }
  }
}

export function formatJson(data: Data): string {
  return JSON.stringify(data, null, 2)
}

/** Value at a dotted path with indexes: "items[0].name". */
export function lookup(data: Data, path: string[]): unknown {
  let cur: unknown = data
  for (const p of path) {
    if (cur === null || cur === undefined) return undefined
    if (Array.isArray(cur)) {
      if (p === 'size') return cur.length
      if (p === 'first') {
        cur = cur[0]
        continue
      }
      if (p === 'last') {
        cur = cur[cur.length - 1]
        continue
      }
      const i = Number(p)
      cur = Number.isInteger(i) ? cur[i] : undefined
      continue
    }
    if (typeof cur === 'object') {
      cur = (cur as Record<string, unknown>)[p]
      continue
    }
    if (typeof cur === 'string' && p === 'size') return cur.length
    return undefined
  }
  return cur
}

/** One `{{ … }}` output of a Liquid source, with its offsets. */
export interface Output {
  expr: string
  start: number
  end: number
}

export function liquidOutputs(src: string): Output[] {
  const out: Output[] = []
  const re = /\{\{-?\s*([\s\S]*?)\s*-?\}\}/g
  let m: RegExpExecArray | null
  while ((m = re.exec(src))) out.push({ expr: m[1] ?? '', start: m.index, end: m.index + m[0].length })
  return out
}

/** Splits on `|` outside quotes. */
function splitPipes(expr: string): string[] {
  const parts: string[] = []
  let cur = ''
  let quote = ''
  for (const ch of expr) {
    if (quote) {
      if (ch === quote) quote = ''
      cur += ch
      continue
    }
    if (ch === '"' || ch === "'") {
      quote = ch
      cur += ch
      continue
    }
    if (ch === '|') {
      parts.push(cur)
      cur = ''
      continue
    }
    cur += ch
  }
  parts.push(cur)
  return parts.map((p) => p.trim())
}

export interface ParsedOutput {
  /** The variable the output reads, or "" for a literal. */
  root: string
  path: string[]
  /** The argument of a `default:` filter. */
  fallback?: string
}

/** Reads `name.path[0] | default: "x"` from an output's expression. */
export function parseOutput(expr: string): ParsedOutput {
  const [head = '', ...filters] = splitPipes(expr)
  let fallback: string | undefined
  for (const f of filters) {
    const m = /^default\s*:\s*(?:"([^"]*)"|'([^']*)'|([\w.-]+))/.exec(f)
    if (m) fallback = m[1] ?? m[2] ?? m[3]
  }
  const m = /^([A-Za-z_][\w-]*)((?:\.[\w-]+|\[\d+\]|\["[^"]*"\])*)$/.exec(head)
  if (!m) return { root: '', path: [], fallback }
  const path: string[] = []
  const rest = m[2] ?? ''
  const re = /\.([\w-]+)|\[(\d+)\]|\["([^"]*)"\]/g
  let p: RegExpExecArray | null
  while ((p = re.exec(rest))) path.push(p[1] ?? p[2] ?? p[3] ?? '')
  return { root: m[1]!, path, fallback }
}

/** Whether a Liquid source reads a variable: in an output, or in an if, unless, case or for tag. */
export function usesVariable(src: string, name: string): boolean {
  if (!src || !src.includes(name)) return false
  if (liquidOutputs(src).some((o) => parseOutput(o.expr).root === name)) return true
  const n = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return new RegExp(`\\{%-?\\s*(?:if|elsif|unless|case|when|for\\s+\\w+\\s+in)\\s+[^%]*?(?<![\\w.])${n}(?![\\w])`).test(src)
}

/** Names a template creates itself: loop items, assigns and captures. */
export function localsOf(src: string): Set<string> {
  const out = new Set<string>()
  const re = /\{%-?\s*(?:for\s+([A-Za-z_]\w*)\s+in|assign\s+([A-Za-z_]\w*)\s*=|capture\s+([A-Za-z_]\w*)|tablerow\s+([A-Za-z_]\w*)\s+in)/g
  let m: RegExpExecArray | null
  while ((m = re.exec(src))) out.add(m[1] ?? m[2] ?? m[3] ?? m[4] ?? '')
  out.delete('')
  return out
}

/** How a value shows on the canvas: text as is, lists by their size. */
export function display(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    const s = JSON.stringify(value)
    return s.length > 48 ? `${s.slice(0, 47)}…` : s
  } catch {
    return ''
  }
}

/** The chip of one output on the canvas: what it prints with the preview data. */
export interface TokenView {
  text: string
  /** '' a value, 's' a sensitive value, 'e' undeclared or a missing required value, 'm' prints nothing. */
  tone: '' | 's' | 'e' | 'm'
  /** The variable it reads ("" for a literal or a local). */
  name: string
}

export interface TokenContext {
  declared: Map<string, Variable>
  data: Data
  locals: Set<string>
}

export function tokenView(expr: string, ctx: TokenContext): TokenView {
  const p = parseOutput(expr)
  if (!p.root) return { text: expr, tone: '', name: '' }
  const shown = p.path.length ? `${p.root}.${p.path.join('.')}` : p.root
  if (p.root === 'to') {
    const v = p.path[0] === 'name' ? PREVIEW_TO.name : PREVIEW_TO.email
    return { text: v, tone: '', name: 'to' }
  }
  if (RESERVED.has(p.root)) return { text: shown, tone: '', name: p.root }
  if (ctx.locals.has(p.root)) return { text: shown, tone: '', name: '' }
  const v = ctx.declared.get(p.root)
  if (!v) return { text: p.root, tone: 'e', name: p.root }
  const value = lookup(ctx.data, [p.root, ...p.path])
  const tone = v.sensitive ? 's' : ''
  if (value !== undefined && value !== null && value !== '') return { text: display(value), tone, name: p.root }
  if (p.fallback !== undefined) return { text: p.fallback, tone, name: p.root }
  if (v.default) return { text: v.default, tone, name: p.root }
  if (v.required) return { text: p.root, tone: 'e', name: p.root }
  return { text: p.root, tone: 'm', name: p.root }
}

/** Renames a variable in every test data set. */
export function renameInData(sets: TestData[], from: string, to: string): TestData[] {
  return sets.map((s) => {
    if (!(from in s.data)) return s
    const data: Data = {}
    for (const [k, v] of Object.entries(s.data)) data[k === from ? to : k] = v
    return { ...s, data }
  })
}

/** Replaces `{{ from` outputs with `{{ to` in a Liquid source. */
export function renameInSource(src: string, from: string, to: string): string {
  const re = new RegExp(`(\\{\\{-?\\s*)${from.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?=[\\s.|\\[}-])`, 'g')
  return src.replace(re, `$1${to}`)
}

/** Removes every `{{ name … }}` output from a Liquid source. */
export function removeOutputs(src: string, name: string): string {
  return src.replace(/\{\{-?\s*([\s\S]*?)\s*-?\}\}/g, (all, expr: string) => (parseOutput(expr).root === name ? '' : all))
}

/** A unique test data name: "New user", then "New user 2". */
export function uniqueName(base: string, taken: string[]): string {
  if (!taken.includes(base)) return base
  for (let i = 2; ; i++) {
    const n = `${base} ${i}`
    if (!taken.includes(n)) return n
  }
}
