import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'
import { Component, createContext, useContext, useEffect, useSyncExternalStore, type CSSProperties, type ReactNode } from 'react'
import { $api, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useMediaQuery } from '@/lib/hooks'
import { fetchWorkspaceMe, useOptionalWorkspace, workspaceMeKey } from '@/lib/session'
import { modKeyLabel, useShortcutHandler } from '@/lib/shortcuts'
import { ml } from './launcher.messages'
import type { AssistantPanel } from './Panel'
import { getAssistant, loadedPanel, loadPanel, mountSlot, PANEL_MIN, setAssistantThread, toggleAssistant, useAssistant, waitingThread } from './store'
import './shell.css'

/**
 * The assistant's part of the first load (assistant brief §3.1): the top
 * bar's button, ⌘J and the panel's place in the shell. The panel is a lazy
 * chunk, fetched when the button is hovered or focused, or on ⌘J.
 */

/** The sidebar's width above 1100 px. */
const SIDEBAR = 232
/** The content column never gets narrower than this beside a docked panel. */
const MIN_CONTENT = 720
/**
 * Screens with review decisions (Review & send, an approval, a proposal,
 * an agent run's review): the panel never covers them on a desktop. When
 * the content would get too narrow it docks at its narrowest instead of
 * overlaying.
 */
const REVIEW_SCREEN = /\/(approvals|proposals|agent)\/[^/]+$|\/campaigns\/[^/]+\/(review|sms-review|push-review|whatsapp-review)$/

export type PanelMode = 'docked' | 'overlay' | 'sheet' | 'full'
export interface PanelLayout {
  mode: PanelMode
  width: number
}

function subscribeResize(cb: () => void) {
  window.addEventListener('resize', cb)
  return () => window.removeEventListener('resize', cb)
}

/**
 * Where the panel goes at this width: docked, over the content, a sheet or
 * the whole screen. Without docking (a full-screen editor) it goes over
 * the content on a desktop.
 */
export function useAssistantLayout(dock = true): PanelLayout {
  const { width } = useAssistant()
  const phone = useMediaQuery('(max-width: 767px)')
  const tablet = useMediaQuery('(max-width: 1100px)')
  const vw = useSyncExternalStore(subscribeResize, () => document.documentElement.clientWidth, () => 1440)
  const review = useRouterState({ select: (s) => REVIEW_SCREEN.test(s.location.pathname) })
  if (phone) return { mode: 'full', width }
  if (tablet) return { mode: 'sheet', width }
  if (!dock) return { mode: 'overlay', width }
  const room = vw - SIDEBAR - MIN_CONTENT
  if (width <= room) return { mode: 'docked', width }
  if (review) return { mode: 'docked', width: Math.max(PANEL_MIN, Math.min(width, room)) }
  return { mode: 'overlay', width }
}

/** Inside AppShell, which has the panel's place: the launcher shows only there. */
export const AssistantPlace = createContext(false)

/**
 * The panel's place on a screen outside AppShell (the composer): over the
 * content, never docked, for the contextual buttons and ⌘J there.
 */
export function AssistantOver({ children }: { children: ReactNode }) {
  const layout = useAssistantLayout(false)
  return (
    <AssistantPlace.Provider value>
      {children}
      <AssistantSlot layout={layout} />
    </AssistantPlace.Provider>
  )
}

/**
 * A call that waits for the person, from their unread notifications: the
 * launcher's dot survives a reload, and shows when the call came to wait
 * after the panel closed. Asked again as the unread count changes, and
 * the count every 15 s while a closed panel's turn may come to wait.
 */
function useWaitingThread(slug: string, can: boolean, unread: number, watching: boolean): string | undefined {
  const path = '/api/v1/workspaces/{workspace}/notifications'
  // Under the notifications' key: reading one, or all, refreshes it too.
  const note = useQuery({
    queryKey: ['get', path, 'waiting', slug, unread],
    queryFn: async () => (await http.GET(path, { params: { path: { workspace: slug }, query: { filter: 'unread', limit: 20 } } })).data!,
    enabled: can && unread > 0,
    select: (d) => d.items.map((n) => (n.read ? undefined : waitingThread(n))).find(Boolean),
  })
  useQuery({
    queryKey: workspaceMeKey(slug),
    queryFn: () => fetchWorkspaceMe(slug),
    enabled: can && watching && !note.data,
    refetchInterval: () => (getAssistant().watch > Date.now() ? 15_000 : false),
  })
  return can && unread > 0 ? note.data : undefined
}

/** The top bar's Assistant button, next to the bell (AssistantEntry). */
export function AssistantLauncher() {
  const ws = useOptionalWorkspace()
  const inShell = useContext(AssistantPlace)
  const ui = useAssistant()
  const t = useT()
  const waitsIn = useWaitingThread(ws?.slug ?? '', Boolean(ws?.me.permissions.includes('ai:use')), ws?.me.unread ?? 0, ui.watch > 0)
  if (!ws || !inShell) return null
  const dot = (ui.waiting || Boolean(waitsIn)) && !ui.open
  const prefetch = () => void loadPanel().catch(() => {})
  return (
    <button
      type="button"
      className="asst-launch"
      data-assistant-launcher=""
      aria-expanded={ui.open}
      aria-controls={ui.open ? 'assistant' : undefined}
      aria-label={dot ? t(ml.launcherWaiting) : t(ml.launcher)}
      title={t(ml.launcherTitle, { keys: modKeyLabel() === '⌘' ? '⌘J' : 'Ctrl J' })}
      onPointerEnter={prefetch}
      onFocus={prefetch}
      onClick={() => {
        // The dot's call is in its own thread: the panel opens on it.
        if (dot && waitsIn) setAssistantThread(ws.slug, waitsIn)
        toggleAssistant()
      }}
    >
      <Icon name="chat" size={18} />
      <span className="asst-label" aria-hidden="true">
        {t(ml.launcher)}
      </span>
      {dot ? <span className="dot" /> : null}
    </button>
  )
}

/**
 * Once the page is idle after loading, the panel's chunk comes too, so it
 * opens at once when the button is pressed (not only after a hover, which
 * touch screens don't have): only for someone who can use the assistant,
 * when it is on (its models answer, which the panel then has at hand),
 * and not on a connection that saves data or is 2G.
 */
function useIdlePrefetch(qc: ReturnType<typeof useQueryClient>, slug: string | undefined, can: boolean) {
  useEffect(() => {
    const net = (navigator as { connection?: { saveData?: boolean; effectiveType?: string } }).connection
    if (!slug || !can || net?.saveData || /2g/.test(net?.effectiveType ?? '')) return
    let idle = 0
    const go = () => {
      idle = (window.requestIdleCallback ?? ((f: () => void) => window.setTimeout(f, 1000)))(() => {
        void qc
          .fetchQuery({ ...$api.queryOptions('get', '/api/v1/workspaces/{workspace}/ai/assistant/models', { params: { path: { workspace: slug } } }), staleTime: 30_000 })
          .then(() => loadPanel())
          .catch(() => {})
      })
    }
    if (document.readyState === 'complete') go()
    else window.addEventListener('load', go, { once: true })
    return () => {
      window.removeEventListener('load', go)
      ;(window.cancelIdleCallback ?? window.clearTimeout)(idle)
    }
  }, [qc, slug, can])
}

/** The panel failed to render: say so where the panel goes, with Retry. */
class PanelBoundary extends Component<{ children: ReactNode; fallback: () => ReactNode }, { failed: boolean }> {
  state = { failed: false }
  static getDerivedStateFromError() {
    return { failed: true }
  }
  render() {
    return this.state.failed ? this.props.fallback() : this.props.children
  }
}

function PanelFrame({ layout, children }: { layout: PanelLayout; children: ReactNode }) {
  return (
    <aside id="assistant" className={`asst asst-${layout.mode}`} aria-busy="true" style={{ '--asst-w': `${layout.width}px` } as CSSProperties}>
      <div className="asst-wait">{children}</div>
    </aside>
  )
}

/**
 * The panel's place in AppShell: ⌘J toggles it, and it renders once open.
 * The docked panel is a column of the shell, so the content narrows.
 */
export function AssistantSlot({ layout }: { layout: PanelLayout }) {
  const ui = useAssistant()
  const t = useT()
  const ws = useOptionalWorkspace()
  const qc = useQueryClient()
  useEffect(mountSlot, [])
  useIdlePrefetch(qc, ws?.slug, Boolean(ws?.me.permissions.includes('ai:use')))
  useShortcutHandler('assistant', () => {
    toggleAssistant()
  })
  useEffect(() => {
    if (ui.open) void loadPanel().catch(() => {})
  }, [ui.open])
  if (!ui.open) return null
  const failed = (
    <PanelFrame layout={layout}>
      <b role="alert">{t(ml.loadFailed)}</b>
      <span>
        {/* A chunk that didn't load (a new release meanwhile) comes with the page again. */}
        <Button size="sm" onClick={() => window.location.reload()}>
          {t(common.retry)}
        </Button>
      </span>
    </PanelFrame>
  )
  const Panel = (loadedPanel() as { AssistantPanel: typeof AssistantPanel } | undefined)?.AssistantPanel
  if (!Panel) return ui.panel === 'failed' ? failed : <PanelFrame layout={layout}>{t(ml.loading)}</PanelFrame>
  return (
    <PanelBoundary fallback={() => failed}>
      <Panel layout={layout} />
    </PanelBoundary>
  )
}
