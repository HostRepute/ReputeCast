import { useRouterState } from '@tanstack/react-router'
import { Component, createContext, lazy, Suspense, useContext, useEffect, useSyncExternalStore, type ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useMediaQuery } from '@/lib/hooks'
import { useOptionalWorkspace } from '@/lib/session'
import { modKeyLabel, useShortcutHandler } from '@/lib/shortcuts'
import { ml } from './launcher.messages'
import type { AssistantPanel } from './Panel'
import { loadPanel, mountSlot, PANEL_MIN, toggleAssistant, useAssistant } from './store'
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

/** Where the panel goes at this width: docked, over the content, a sheet or the whole screen. */
export function useAssistantLayout(): PanelLayout {
  const { width } = useAssistant()
  const phone = useMediaQuery('(max-width: 767px)')
  const tablet = useMediaQuery('(max-width: 1100px)')
  const vw = useSyncExternalStore(subscribeResize, () => document.documentElement.clientWidth, () => 1440)
  const review = useRouterState({ select: (s) => REVIEW_SCREEN.test(s.location.pathname) })
  if (phone) return { mode: 'full', width }
  if (tablet) return { mode: 'sheet', width }
  const room = vw - SIDEBAR - MIN_CONTENT
  if (width <= room) return { mode: 'docked', width }
  if (review) return { mode: 'docked', width: Math.max(PANEL_MIN, Math.min(width, room)) }
  return { mode: 'overlay', width }
}

/** Inside AppShell, which has the panel's place: the launcher shows only there. */
export const AssistantPlace = createContext(false)

/** The top bar's Assistant button, next to the bell (AssistantEntry). */
export function AssistantLauncher() {
  const ws = useOptionalWorkspace()
  const inShell = useContext(AssistantPlace)
  const ui = useAssistant()
  const t = useT()
  if (!ws || !inShell) return null
  const dot = ui.waiting && !ui.open
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
      onClick={toggleAssistant}
    >
      <Icon name="chat" size={18} />
      <span className="asst-label" aria-hidden="true">
        {t(ml.launcher)}
      </span>
      {dot ? <span className="dot" /> : null}
    </button>
  )
}

const Panel = lazy(() => loadPanel().then((mod) => ({ default: (mod as { AssistantPanel: typeof AssistantPanel }).AssistantPanel })))

/** The panel's chunk failed to load: say so where the panel goes, with Retry. */
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
    <aside id="assistant" className={`asst asst-${layout.mode}`} aria-busy="true">
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
  useEffect(mountSlot, [])
  useShortcutHandler('assistant', () => {
    toggleAssistant()
  })
  if (!ui.open) return null
  return (
    <PanelBoundary
      fallback={() => (
        <PanelFrame layout={layout}>
          <b role="alert">{t(ml.loadFailed)}</b>
          <span>
            {/* A chunk that didn't load (a new release meanwhile) comes with the page again. */}
            <Button size="sm" onClick={() => window.location.reload()}>
              {t(common.retry)}
            </Button>
          </span>
        </PanelFrame>
      )}
    >
      <Suspense fallback={<PanelFrame layout={layout}>{t(ml.loading)}</PanelFrame>}>
        <Panel layout={layout} />
      </Suspense>
    </PanelBoundary>
  )
}
