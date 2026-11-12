import type { Component, Editor } from 'grapesjs'
import { http, type Schemas } from '@/api/client'

/**
 * The Countdown block (M59 G-21): an mj-image whose `rc-countdown`
 * attribute holds its settings and whose src is the GIF the server draws
 * when the email is opened (/t/cd/{token}.gif). The settings stay in the
 * source (internal/countdown reads them for the checks); the attribute is
 * stripped before compiling (stripRcAttrs, mjml.PrepareCompile).
 */
export type CountdownSettings = Schemas['CountdownSettings']
export type CountdownImage = Schemas['CountdownImage']

export const ATTR_COUNTDOWN = 'rc-countdown'

/** The languages the timer's labels come in. */
export const COUNTDOWN_LANGUAGES = ['en', 'de', 'es', 'fr', 'vi'] as const

/** Sizes in pixels: S, M, L. */
export const COUNTDOWN_SIZES: Record<CountdownSettings['size'], [number, number]> = { s: [360, 96], m: [480, 120], l: [600, 150] }

/** "2026-11-27T23:59" a week from now, in a time zone. */
function inAWeek(zone: string): string {
  const d = new Date(Date.now() + 7 * 24 * 3600 * 1000)
  try {
    const p = new Intl.DateTimeFormat('en-CA', { timeZone: zone, year: 'numeric', month: '2-digit', day: '2-digit' }).format(d)
    return `${p}T23:59`
  } catch {
    return `${d.toISOString().slice(0, 10)}T23:59`
  }
}

/** A new block's settings: a week from now at 23:59 in the workspace's zone. */
export function newCountdown(zone: string, language: string): CountdownSettings {
  const lang = (COUNTDOWN_LANGUAGES as readonly string[]).includes(language) ? (language as CountdownSettings['language']) : 'en'
  return {
    endsAt: inAWeek(zone || 'UTC'),
    timeZone: zone || 'UTC',
    units: 'dhms',
    language: lang,
    size: 'l',
    background: '#0F172A',
    digits: '#FFFFFF',
    labels: '#CBD5E1',
    ended: { mode: 'zeros' },
  }
}

/** The block's MJML; src is empty until the server makes the image. */
export function countdownMjml(s: CountdownSettings, img?: CountdownImage): string {
  const json = JSON.stringify(s).replace(/&/g, '&amp;').replace(/"/g, '&quot;')
  const [w] = COUNTDOWN_SIZES[s.size]
  const src = img?.url ?? ''
  const alt = (img?.alt ?? '').replace(/&/g, '&amp;').replace(/"/g, '&quot;')
  return `<mj-image ${ATTR_COUNTDOWN}="${json}" src="${src}" alt="${alt}" width="${w}px" padding="0px"></mj-image>`
}

export function isCountdown(c: Component | null | undefined): boolean {
  if (!c || (c.get('tagName') as string | undefined) !== 'mj-image') return false
  return (c.getAttributes() as Record<string, unknown>)[ATTR_COUNTDOWN] !== undefined
}

/** A block's settings, or null when its attribute isn't readable. */
export function countdownOf(c: Component): CountdownSettings | null {
  const raw = (c.getAttributes() as Record<string, unknown>)[ATTR_COUNTDOWN]
  if (typeof raw !== 'string') return null
  try {
    const v = JSON.parse(raw) as CountdownSettings
    return v && typeof v === 'object' && typeof v.endsAt === 'string' ? v : null
  } catch {
    return null
  }
}

/** The image of a set of settings (the same settings give the same image). */
export async function countdownImage(slug: string, s: CountdownSettings): Promise<CountdownImage> {
  const { data, error } = await http.POST('/api/v1/workspaces/{workspace}/countdowns', { params: { path: { workspace: slug } }, body: s })
  if (error || !data) throw error ?? new Error('countdown')
  return data
}

/** Puts settings and their image on a block (one undoable change). */
export function applyCountdown(c: Component, s: CountdownSettings, img: CountdownImage) {
  c.addAttributes({ [ATTR_COUNTDOWN]: JSON.stringify(s), src: img.url, alt: img.alt, width: `${img.width}px` })
}

/** Gives each Countdown block dropped or pasted without an image its image. */
export function loadCountdowns(editor: Editor, slug: () => string) {
  const pending = new WeakSet<Component>()
  const fill = (c: Component) => {
    if (!isCountdown(c) || pending.has(c)) return
    const src = String((c.getAttributes() as Record<string, unknown>).src ?? '')
    if (src.includes('/t/cd/')) return
    const s = countdownOf(c)
    if (!s) return
    pending.add(c)
    countdownImage(slug(), s)
      .then((img) => applyCountdown(c, s, img))
      .catch(() => pending.delete(c))
  }
  const walk = (c: Component) => {
    fill(c)
    c.components().forEach((x: Component) => walk(x))
  }
  editor.on('component:add', walk)
}
