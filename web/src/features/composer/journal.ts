import { useMemo, useState } from 'react'
import type { ABTest } from './context'

/**
 * The browser journal of unsaved composer edits (CO-1). Every edit, every
 * keystroke in a text included, is written to this browser's storage within
 * a couple of seconds, so a tab that crashes or a laptop that dies mid-typing
 * loses at most those seconds. Reopening the campaign offers the edits back
 * (ComposerStates · Recovered edits: Restore them / Discard). A save that
 * catches up with the edits clears the journal.
 */

export interface JournalEntry {
  v: 1
  campaignId: string
  /** The language version edited; empty for the source. */
  language: string
  /** The version the edits started from. */
  baseVersion: number
  /** When the first unsaved edit was made, and the latest one. */
  startedAt: string
  updatedAt: string
  subject: string
  preheader: string
  mjml: string
  abTest: ABTest | null
}

/** What the journal keeps of the editor at one moment. */
export type JournalSnapshot = Pick<JournalEntry, 'baseVersion' | 'subject' | 'preheader' | 'mjml' | 'abTest'>

const PREFIX = 'rc-composer-journal:'

/** Entries older than this are dropped when the composer opens. */
const KEEP_MS = 14 * 24 * 3600_000

export function journalKey(userId: string, campaignId: string, language?: string): string {
  return `${PREFIX}${userId}:${campaignId}:${language || 'source'}`
}

function storage(): Storage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export function readJournal(key: string): JournalEntry | null {
  try {
    const raw = storage()?.getItem(key)
    if (!raw) return null
    const e = JSON.parse(raw) as Partial<JournalEntry>
    if (e.v !== 1 || typeof e.mjml !== 'string' || typeof e.subject !== 'string' || typeof e.baseVersion !== 'number' || !e.startedAt || !e.updatedAt) return null
    return { preheader: '', abTest: null, language: '', ...e } as JournalEntry
  } catch {
    return null
  }
}

/** Writes an entry; when storage is full, older journals make room. False when it couldn't be kept. */
export function writeJournal(key: string, entry: JournalEntry): boolean {
  const s = storage()
  if (!s) return false
  const raw = JSON.stringify(entry)
  try {
    s.setItem(key, raw)
    return true
  } catch {
    // Storage is full: drop the other journals, oldest first, and try again.
    for (const other of journalKeys().filter((k) => k !== key)) {
      try {
        s.removeItem(other)
        s.setItem(key, raw)
        return true
      } catch {
        /* keep making room */
      }
    }
    return false
  }
}

export function clearJournal(key: string) {
  try {
    storage()?.removeItem(key)
  } catch {
    /* storage unavailable */
  }
}

/** The journals in this browser, oldest first. */
function journalKeys(): string[] {
  const s = storage()
  if (!s) return []
  const out: { key: string; at: string }[] = []
  for (let i = 0; i < s.length; i++) {
    const key = s.key(i)
    if (!key?.startsWith(PREFIX)) continue
    out.push({ key, at: readJournal(key)?.updatedAt ?? '' })
  }
  return out.sort((a, b) => a.at.localeCompare(b.at)).map((x) => x.key)
}

/** Drops journals nobody came back to within two weeks. */
export function pruneJournals(now = Date.now()) {
  for (const key of journalKeys()) {
    const e = readJournal(key)
    if (!e || now - Date.parse(e.updatedAt) > KEEP_MS) clearJournal(key)
  }
}

/** Whether a journal holds something the loaded content doesn't. */
export function journalDiffers(e: JournalEntry, loaded: { subject: string; preheader: string; mjml: string; abTest?: ABTest | null }): boolean {
  const norm = (s: string) => s.replace(/\s+/g, ' ').trim()
  return (
    norm(e.mjml) !== norm(loaded.mjml) ||
    e.subject !== loaded.subject ||
    e.preheader !== loaded.preheader ||
    JSON.stringify(e.abTest ?? null) !== JSON.stringify(loaded.abTest ?? null)
  )
}

export interface JournalWriter {
  /** An edit happened: write within `delay`, and at least every `maxWait` while edits go on. */
  touch(): void
  /** Write now (the tab is being hidden or closed). */
  flush(): void
  /** The edits are saved: forget them. */
  clear(): void
  dispose(): void
}

/**
 * Writes the journal after edits: `delay` ms after the last one, and no
 * later than `maxWait` ms after the first one of a burst, so continuous
 * typing is written every `maxWait` ms.
 */
export function journalWriter(opts: {
  key: string
  campaignId: string
  language?: string
  snapshot: () => JournalSnapshot | null
  now?: () => Date
  delay?: number
  maxWait?: number
}): JournalWriter {
  const now = opts.now ?? (() => new Date())
  const delay = opts.delay ?? 800
  const maxWait = opts.maxWait ?? 2000
  let timer: ReturnType<typeof setTimeout> | undefined
  let firstPending: number | undefined
  let startedAt: string | undefined = readJournal(opts.key)?.startedAt
  let dirty = false
  const write = () => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
    firstPending = undefined
    if (!dirty) return
    const snap = opts.snapshot()
    if (!snap) return
    const at = now().toISOString()
    writeJournal(opts.key, { v: 1, campaignId: opts.campaignId, language: opts.language ?? '', startedAt: startedAt ?? at, updatedAt: at, ...snap })
  }
  return {
    touch() {
      dirty = true
      // The first unsaved edit since the last save starts the journal's span.
      startedAt ??= now().toISOString()
      const t = now().getTime()
      firstPending ??= t
      if (timer !== undefined) clearTimeout(timer)
      const wait = Math.max(0, Math.min(delay, firstPending + maxWait - t))
      timer = setTimeout(write, wait)
    },
    flush: write,
    clear() {
      if (timer !== undefined) clearTimeout(timer)
      timer = undefined
      firstPending = undefined
      dirty = false
      startedAt = undefined
      clearJournal(opts.key)
    },
    dispose() {
      if (timer !== undefined) clearTimeout(timer)
      timer = undefined
    },
  }
}

/**
 * The edits to offer back when the composer opens: the journal of the last
 * session moves aside (to `<key>:recovered`), so new edits don't write over
 * it before the person restores or discards it. Null when there is nothing
 * the loaded content doesn't already have.
 */
export function recoverJournal(key: string, loaded: { subject: string; preheader: string; mjml: string; abTest?: ABTest | null }): JournalEntry | null {
  const asideKey = `${key}:recovered`
  pruneJournals()
  // The newest unsaved edits win: a journal written after edits were offered
  // back (and left undecided) holds what the person did last.
  const main = readJournal(key)
  const aside = readJournal(asideKey)
  const found = main && journalDiffers(main, loaded) ? main : aside && journalDiffers(aside, loaded) ? aside : null
  clearJournal(key)
  clearJournal(asideKey)
  if (found) writeJournal(asideKey, found)
  return found
}

/** Forgets the edits offered back (restored or discarded). */
export function forgetRecovered(key: string) {
  clearJournal(`${key}:recovered`)
}

/**
 * The edits offered back when the composer opens (recoverJournal), read once;
 * set it to null once they are restored or discarded.
 */
export function useRecoveredJournal(key: string, loaded: { subject: string; preheader: string; mjml: string; abTest?: ABTest | null }, enabled: boolean) {
  return useState<JournalEntry | null>(() => (enabled ? recoverJournal(key, loaded) : null))
}

/** journalKey for a component (memoized, so effects keyed by it run once per campaign). */
export function useJournalKey(userId: string, campaignId: string, language?: string): string {
  return useMemo(() => journalKey(userId, campaignId, language), [userId, campaignId, language])
}
