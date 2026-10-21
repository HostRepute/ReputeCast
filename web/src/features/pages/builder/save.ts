/**
 * The draft's save state (PageBuilderStates · Autosave, saved and a
 * conflict). Autosave runs every 30 seconds and on every block added,
 * moved or deleted; Save and ⌘S save at once. A newer version saved
 * meanwhile by someone else is a conflict: keeping yours saves on top of
 * theirs (force), loading theirs replaces the canvas.
 */

/** Someone else's newer version (409 version_conflict's meta). */
export interface Theirs {
  version: number
  by: string
  at: string
}

export type SaveStatus =
  /** The draft is saved: `auto` when the last save was an autosave, `by` who saved it. */
  | { kind: 'saved'; version: number; at: string; by: string; auto: boolean }
  /** Edits not saved yet, since the save at `at`. */
  | { kind: 'dirty'; version: number; at?: string }
  | { kind: 'saving'; version: number; at?: string }
  | { kind: 'conflict'; version: number; theirs: Theirs }
  | { kind: 'offline'; version: number; since: string }
  | { kind: 'error'; version: number; message: string }

export type SaveEvent =
  /** The canvas or the blocks changed. */
  | { type: 'edit' }
  | { type: 'start' }
  /** Saved as `version`; `dirtyAgain` when edits came in while it saved. */
  | { type: 'saved'; version: number; at: string; by: string; manual: boolean; dirtyAgain: boolean }
  | { type: 'conflict'; theirs: Theirs }
  | { type: 'offline'; at: string }
  | { type: 'failed'; message: string }
  /** The draft was (re)loaded from the server. */
  | { type: 'loaded'; version: number; at: string; by: string; auto: boolean }
  /** A version was made outside the canvas (the page's settings): the next save goes on top of it. */
  | { type: 'version'; version: number }

function lastAt(s: SaveStatus): string | undefined {
  return s.kind === 'saved' || s.kind === 'dirty' || s.kind === 'saving' ? s.at : undefined
}

export function saveReducer(s: SaveStatus, e: SaveEvent): SaveStatus {
  switch (e.type) {
    case 'edit':
      // A conflict or an outage stays until it's resolved; a save in flight reports when it ends.
      if (s.kind === 'saving' || s.kind === 'conflict' || s.kind === 'offline' || s.kind === 'dirty') return s
      return { kind: 'dirty', version: s.version, at: lastAt(s) }
    case 'start':
      // Retries while offline stay offline (the card keeps when it began) until a save gets through.
      if (s.kind === 'offline') return s
      return { kind: 'saving', version: s.version, at: lastAt(s) }
    case 'saved':
      return e.dirtyAgain ? { kind: 'dirty', version: e.version, at: e.at } : { kind: 'saved', version: e.version, at: e.at, by: e.by, auto: !e.manual }
    case 'conflict':
      return { kind: 'conflict', version: s.version, theirs: e.theirs }
    case 'offline':
      return { kind: 'offline', version: s.version, since: s.kind === 'offline' ? s.since : e.at }
    case 'failed':
      return { kind: 'error', version: s.version, message: e.message }
    case 'loaded':
      return { kind: 'saved', version: e.version, at: e.at, by: e.by, auto: e.auto }
    case 'version':
      return e.version > s.version ? { ...s, version: e.version } : s
  }
}

/** Edits that would be lost on leaving. */
export function isUnsaved(s: SaveStatus): boolean {
  return s.kind === 'dirty' || s.kind === 'saving' || s.kind === 'offline' || s.kind === 'error' || s.kind === 'conflict'
}

/** The other person's version from a 409 version_conflict problem's meta. */
export function theirsFrom(meta: Readonly<Record<string, unknown>> | undefined): Theirs {
  const m = meta ?? {}
  const v = Number(m.version)
  return { version: Number.isFinite(v) ? v : 0, by: typeof m.savedBy === 'string' ? m.savedBy : '', at: typeof m.savedAt === 'string' ? m.savedAt : '' }
}

/** A network failure or a gateway in between: retry later, keep the edits. */
export function isOffline(e: unknown): boolean {
  if (e instanceof TypeError) return true
  const status = (e as { status?: number } | null)?.status
  return status === 502 || status === 503 || status === 504
}

/** When the timers save: every 30 seconds while there are edits, and soon after a block was added, moved or deleted. */
export const AUTOSAVE_MS = 30_000
export const STRUCTURAL_MS = 1_500
export const OFFLINE_RETRY_MS = 10_000

/** The project saved with the HTML: the blocks are the HTML, the editor reads nothing else. */
export const PROJECT = { editor: 'grapesjs', format: 'reputecast-page', v: 1 } as const
