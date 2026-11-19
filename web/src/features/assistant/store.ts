import { useSyncExternalStore } from 'react'

/**
 * The assistant panel's state outside it (assistant brief §3.1): open or
 * closed, its width on a desktop, the thread it shows and what waits in its
 * composer. It lives with the launcher in the first load; the panel itself
 * is a lazy chunk (loadPanel), fetched when the launcher is hovered or
 * focused, or on ⌘J.
 */
export interface AssistantUi {
  open: boolean
  /** The docked width on a desktop, PANEL_MIN–PANEL_MAX. */
  width: number
  view: 'thread' | 'list'
  /** The thread shown, by workspace; none: a new thread starts on sending. */
  threads: Record<string, string | undefined>
  /** Text put into the composer from outside (⌘K › Ask the assistant), unsent; nonce tells a new one. */
  draft: string
  draftNonce: number
  /** A confirmation waits in the closed panel: the launcher's dot. */
  waiting: boolean
  /**
   * Until when (ms since 1970) the launcher looks out for a call that waits,
   * after the panel closed while its turn was answering; 0: it doesn't.
   */
  watch: number
  /** Panels mounted (AppShell's slot): the launcher shows only where one is. */
  slots: number
  /** What a contextual button suggests, on the screen it was pressed on; nothing is attached until the person attaches it. */
  suggest?: AssistantSuggestion
  /** The panel's chunk: loaded (it renders at once), or failed to load. */
  panel?: 'ready' | 'failed'
}

/** A resource an "Ask the assistant" button suggests as context (a report, a draft campaign). */
export interface AssistantSuggestion {
  kind: 'report' | 'campaign'
  id: string
  /** The page it was suggested on: elsewhere the route's own suggestion stands. */
  path: string
}

export const PANEL_MIN = 360
export const PANEL_MAX = 640
export const PANEL_DEFAULT = 400

const WIDTH_KEY = 'rc.assistant.width'
const threadKey = (ws: string) => `rc.assistant.thread.${ws}`

function stored(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function store(key: string, value: string | undefined) {
  try {
    if (value === undefined) window.localStorage.removeItem(key)
    else window.localStorage.setItem(key, value)
  } catch {
    // Private mode or full storage: it lasts for this page only.
  }
}

export function clampWidth(w: number): number {
  return Math.min(PANEL_MAX, Math.max(PANEL_MIN, Math.round(w)))
}

let state: AssistantUi = {
  open: false,
  width: typeof window === 'undefined' ? PANEL_DEFAULT : clampWidth(Number(stored(WIDTH_KEY)) || PANEL_DEFAULT),
  view: 'thread',
  threads: {},
  draft: '',
  draftNonce: 0,
  waiting: false,
  watch: 0,
  slots: 0,
}
const listeners = new Set<() => void>()
/** What had focus when the panel opened: focus goes back there on close. */
let opener: HTMLElement | null = null

function set(patch: Partial<AssistantUi>) {
  state = { ...state, ...patch }
  for (const l of listeners) l()
}

function subscribe(l: () => void) {
  listeners.add(l)
  return () => {
    listeners.delete(l)
  }
}

export function useAssistant(): AssistantUi {
  return useSyncExternalStore(subscribe, () => state, () => state)
}

export function getAssistant(): AssistantUi {
  return state
}

let panel: Promise<unknown> | undefined
let panelModule: unknown
/**
 * Fetches the panel's chunk ahead of opening it. Once it is in, the slot
 * renders the panel at once, never through a Suspense fallback (whose
 * reveal React holds back for up to 300 ms).
 */
export function loadPanel() {
  panel ??= import('./Panel').then(
    (mod) => {
      panelModule = mod
      set({ panel: 'ready' })
      return mod
    },
    (e: unknown) => {
      panel = undefined
      set({ panel: 'failed' })
      throw e
    },
  )
  return panel
}

/** The panel's module, once loaded. */
export function loadedPanel(): unknown {
  return panelModule
}

/**
 * Opens the panel; with a draft, the composer holds it, unsent; with a
 * resource, the thread suggests it as context (a contextual button).
 */
export function openAssistant(opts: { draft?: string; suggest?: Omit<AssistantSuggestion, 'path'> } = {}) {
  void loadPanel().catch(() => {})
  if (!state.open) opener = document.activeElement instanceof HTMLElement && document.activeElement !== document.body ? document.activeElement : null
  const toThread = opts.draft !== undefined || opts.suggest !== undefined
  set({
    open: true,
    waiting: false,
    watch: 0,
    view: toThread ? 'thread' : state.view,
    suggest: opts.suggest ? { ...opts.suggest, path: window.location.pathname } : undefined,
    ...(opts.draft !== undefined ? { draft: opts.draft, draftNonce: state.draftNonce + 1 } : {}),
  })
}

/** How long the launcher looks out for a call that waits, after the panel closed mid-answer. */
const WATCH_FOR = 10 * 60_000

/**
 * Closes the panel and gives focus back to what opened it, or to the
 * launcher. A call that waits leaves the launcher's dot; a turn still
 * answering, a watch for one; neither when the thread goes on showing
 * on its full page.
 */
export function closeAssistant(opts: { refocus?: boolean; toPage?: boolean } = {}) {
  const was = opts.toPage ? undefined : activity
  set({ open: false, waiting: was === 'waiting', watch: was === 'answering' ? Date.now() + WATCH_FOR : 0 })
  if (opts.refocus === false) return
  const back = opener && opener.isConnected ? opener : document.querySelector<HTMLElement>('[data-assistant-launcher]')
  opener = null
  window.setTimeout(() => back?.focus({ preventScroll: true }))
}

export function toggleAssistant() {
  if (state.open) closeAssistant()
  else openAssistant()
}

export function setAssistantWidth(w: number) {
  const width = clampWidth(w)
  if (width === state.width) return
  set({ width })
  store(WIDTH_KEY, String(width))
}

/** What the open panel's turn does, while one runs in it: closing then leaves a dot, or a watch. */
let activity: 'answering' | 'waiting' | undefined
export function setAssistantActivity(now: typeof activity) {
  activity = now
}

/**
 * The notification a call waiting for its owner's confirmation leaves
 * (notify.KindAIAssistantConfirmation). Its link is the thread's full
 * page, /w/{workspace}/assistant/{thread}.
 */
export const WAITING_KIND = 'ai_assistant_confirmation'

/** The thread a waiting call's notification is about, or undefined for any other notification. */
export function waitingThread(n: { kind: string; link: string }): string | undefined {
  return n.kind === WAITING_KIND ? /\/assistant\/([^/?#]+)/.exec(n.link)?.[1] : undefined
}

export function setAssistantView(view: AssistantUi['view']) {
  set({ view })
}

/** The thread a workspace's panel shows (kept for the next visit). */
export function threadOf(ws: string): string | undefined {
  if (ws in state.threads) return state.threads[ws]
  return stored(threadKey(ws)) ?? undefined
}

export function setAssistantThread(ws: string, thread: string | undefined) {
  set({ threads: { ...state.threads, [ws]: thread }, view: 'thread' })
  store(threadKey(ws), thread)
}

let taken = 0
/** The outside draft, while no composer has taken it yet. */
export function untakenDraft(): string {
  return state.draftNonce !== taken ? state.draft : ''
}
/** A composer took the outside draft of this nonce. */
export function markDraftTaken(nonce: number) {
  taken = nonce
}

export function mountSlot(): () => void {
  set({ slots: state.slots + 1 })
  return () => set({ slots: state.slots - 1, open: state.slots - 1 > 0 && state.open })
}
