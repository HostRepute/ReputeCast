import { useEffect, useRef } from 'react'

/**
 * App-wide keyboard shortcuts (Feedback board). Every shortcut has a visible
 * button equivalent; none is required.
 *
 *   ⌘/Ctrl K       Search everything      (works while typing)
 *   ⌘/Ctrl S       Save version           (works while typing)
 *   ⌘/Ctrl Enter   Open Send Readiness    (works while typing)
 *   N              New campaign
 *   G then O/C/U/A/F/D/P/S   go to a section
 *   ?              This sheet
 *   Esc            Close panel or dialog  (handled by each dialog)
 *
 * Single-key shortcuts never fire while focus is in a text field, an editor
 * or a dialog, and never with Alt or Ctrl/⌘ held.
 */
export type GoTarget = 'overview' | 'campaigns' | 'automations' | 'audience' | 'forms' | 'deliverability' | 'providers' | 'settings'

export type ShortcutAction =
  | { type: 'search' }
  | { type: 'save' }
  | { type: 'readiness' }
  | { type: 'new-campaign' }
  | { type: 'help' }
  | { type: 'go'; to: GoTarget }

/** G then … */
export const GO_KEYS: Record<string, { to: GoTarget; label: string }> = {
  o: { to: 'overview', label: 'Overview' },
  c: { to: 'campaigns', label: 'Campaigns' },
  u: { to: 'automations', label: 'Automations' },
  a: { to: 'audience', label: 'Audience' },
  f: { to: 'forms', label: 'Forms & pages' },
  d: { to: 'deliverability', label: 'Deliverability' },
  p: { to: 'providers', label: 'Sending providers' },
  s: { to: 'settings', label: 'Settings' },
}

/** How long the second key of "G then O" may take. */
export const SEQUENCE_MS = 1500

export interface KeyLike {
  key: string
  metaKey: boolean
  ctrlKey: boolean
  altKey: boolean
  shiftKey: boolean
  target: EventTarget | null
}

/** True when keys typed here are text, not commands. */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) return false
  if (target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement) return true
  if (target instanceof HTMLInputElement) {
    return !['checkbox', 'radio', 'button', 'submit', 'reset', 'range', 'color', 'file', 'image'].includes(target.type)
  }
  if (target instanceof HTMLElement && target.isContentEditable) return true
  return Boolean(target.closest('[contenteditable=""], [contenteditable="true"], [role="textbox"], .cm-editor, .gjs-editor'))
}

/** True while a modal dialog is open (its own keys win). */
function inModal(target: EventTarget | null): boolean {
  if (typeof document === 'undefined') return false
  if (target instanceof Element && target.closest('[role="dialog"], [role="alertdialog"]')) return true
  return Boolean(document.querySelector('[role="dialog"][aria-modal="true"], [role="alertdialog"][aria-modal="true"]'))
}

/**
 * Maps a keydown to an action. `pendingG` is true right after "G"; the
 * result says whether the next key continues a sequence.
 */
export function shortcutFor(e: KeyLike, pendingG: boolean): { action?: ShortcutAction; pendingG: boolean } {
  const mod = e.metaKey || e.ctrlKey
  const key = e.key.length === 1 ? e.key.toLowerCase() : e.key
  if (mod && !e.altKey) {
    if (key === 'k' && !e.shiftKey) return { action: { type: 'search' }, pendingG: false }
    if (key === 's' && !e.shiftKey) return { action: { type: 'save' }, pendingG: false }
    if (key === 'Enter') return { action: { type: 'readiness' }, pendingG: false }
    return { pendingG: false }
  }
  if (mod || e.altKey || isTypingTarget(e.target) || inModal(e.target)) return { pendingG: false }
  if (pendingG) {
    const go = GO_KEYS[key]
    return go ? { action: { type: 'go', to: go.to }, pendingG: false } : { pendingG: false }
  }
  if (e.key === '?') return { action: { type: 'help' }, pendingG: false }
  if (key === 'g' && !e.shiftKey) return { pendingG: true }
  if (key === 'n' && !e.shiftKey) return { action: { type: 'new-campaign' }, pendingG: false }
  return { pendingG: false }
}

export type HandlerKind = 'search' | 'save' | 'readiness'
type Handler = () => boolean | void
const handlers: Record<HandlerKind, Handler[]> = { search: [], save: [], readiness: [] }

/**
 * Lets the screen that owns an action handle its shortcut, e.g. the
 * composer: useShortcutHandler('save', saveVersion). The most recently
 * mounted handler runs; returning false passes to the next one.
 */
export function useShortcutHandler(kind: HandlerKind, handler: Handler | undefined) {
  const ref = useRef(handler)
  useEffect(() => {
    ref.current = handler
  })
  useEffect(() => {
    if (!handler) return
    const h: Handler = () => ref.current?.()
    handlers[kind].push(h)
    return () => {
      handlers[kind] = handlers[kind].filter((x) => x !== h)
    }
    // registration follows whether a handler exists, not its identity
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, Boolean(handler)])
}

/** Runs the newest handler for kind; true when one handled it. */
export function runShortcutHandler(kind: HandlerKind): boolean {
  for (let i = handlers[kind].length - 1; i >= 0; i--) {
    if (handlers[kind][i]!() !== false) return true
  }
  return false
}

/** The page's own search field, the default target of ⌘K. */
export function focusPageSearch(): boolean {
  const el = document.querySelector<HTMLInputElement>('[data-shortcut="search"], main input[type="search"]')
  if (!el) return false
  el.focus()
  el.select()
  return true
}

/** ⌘ on Apple devices, Ctrl elsewhere. */
export function modKeyLabel(): string {
  if (typeof navigator === 'undefined') return 'Ctrl'
  const p = (navigator as Navigator & { userAgentData?: { platform?: string } }).userAgentData?.platform ?? navigator.platform ?? ''
  return /mac|iphone|ipad|ipod/i.test(p) ? '⌘' : 'Ctrl'
}
