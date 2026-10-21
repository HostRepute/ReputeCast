import { countText, type Count } from './segments'

/**
 * How the composer counts a text while it's typed, as the server does
 * (internal/texting/compose.go): the locked prefix and opt-out text around
 * the body, each link at a short link's length, each field at its fallback.
 */

/** A short link's code: go.acme.io/k3P9xq. */
export const CODE_LENGTH = 6

/** Public shorteners are filtered by carriers: texts never carry them. */
export const PUBLIC_SHORTENERS = ['bit.ly', 'tinyurl.com', 't.co', 'goo.gl', 'ow.ly', 'is.gd', 'buff.ly', 'rebrand.ly', 'cutt.ly', 'shorturl.at', 'rb.gy', 'tiny.cc', 's.id', 't.ly', 'lnkd.in']

const linkRe = /(?:https?:\/\/[^\s]+|\b(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}\/[^\s]*)/gi
const tokenRe = /\{\{\s*(?:contact\.)?([a-zA-Z_][\w.]*)\s*(?:\|\s*default:\s*(?:"([^"]*)"|'([^']*)')\s*)?\}\}/g

export interface Link {
  url: string
  host: string
  shortener: boolean
  start: number
  end: number
}

/** The links of a body, as the server finds them. */
export function findLinks(body: string): Link[] {
  const out: Link[] = []
  for (const mt of body.matchAll(linkRe)) {
    const start = mt.index ?? 0
    const u = mt[0].replace(/[.,;:!?)]+$/, '')
    if (start > 0 && body[start - 1] === '@') continue
    let host: string
    try {
      host = new URL(u.includes('://') ? u : `https://${u}`).hostname.toLowerCase().replace(/^www\./, '')
    } catch {
      continue
    }
    if (!host) continue
    out.push({ url: u, host, shortener: PUBLIC_SHORTENERS.includes(host), start, end: start + u.length })
  }
  return out
}

/** A field in the body: its name and fallback. */
export interface FieldUse {
  key: string
  fallback: string
  hasFallback: boolean
}

export function findFields(body: string): FieldUse[] {
  const out: FieldUse[] = []
  for (const mt of body.matchAll(tokenRe)) {
    const fb = mt[2] ?? mt[3]
    out.push({ key: mt[1]!, fallback: fb ?? '', hasFallback: fb !== undefined })
  }
  return out
}

/** The body with links shortened on domain (placeholder codes). */
export function withShortLinks(body: string, domain: string): string {
  if (!domain) return body
  let out = ''
  let last = 0
  for (const l of findLinks(body)) {
    if (l.shortener) continue
    out += body.slice(last, l.start) + `${domain}/${'x'.repeat(CODE_LENGTH)}`
    last = l.end
  }
  return out + body.slice(last)
}

/** "Acme: … Reply STOP to opt out". */
export function fullText(prefix: string, body: string, optOut: string): string {
  return [prefix, body.trim(), optOut]
    .map((p) => p.trim())
    .filter(Boolean)
    .join(' ')
}

/** Every field at its fallback (or empty). */
export function renderFallbacks(body: string): string {
  return body.replace(tokenRe, (_all, _key, a?: string, b?: string) => a ?? b ?? '')
}

/** Every field at a value, for "Hi Christopher!". */
export function renderWith(body: string, values: Record<string, string>): string {
  return body.replace(tokenRe, (_all, key: string, a?: string, b?: string) => values[key] || (a ?? b ?? ''))
}

export interface Measured {
  count: Count
  /** The length without the personalized parts. */
  base: number
  fields: FieldUse[]
  links: Link[]
}

/** Counts a text as the composer shows it. */
export function measure(prefix: string, body: string, optOut: string, domain: string): Measured {
  const short = withShortLinks(body, domain)
  const count = countText(fullText(prefix, renderFallbacks(short), optOut))
  const base = countText(fullText(prefix, short.replace(tokenRe, ''), optOut)).characters
  return { count, base, fields: findFields(body), links: findLinks(body) }
}
