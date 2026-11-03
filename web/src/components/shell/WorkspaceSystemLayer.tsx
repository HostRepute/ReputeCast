import { useEffect, useRef, type ReactNode } from 'react'
import { http } from '@/api/client'
import { installSessionSharing } from '@/lib/sessionClock'
import { installSystemStatus, useSystemStatus } from '@/lib/systemStatus'
import { JobsTray } from './JobsTray'
import { SessionGuard } from './SessionGuard'
import { GlobalShortcuts } from './Shortcuts'
import { RateLimitNotice } from './SystemNotices'

// One middleware on the API client feeds the offline, maintenance, rate
// limit and session states for every screen.
if (typeof window !== 'undefined') {
  installSystemStatus(http)
  installSessionSharing()
}

/**
 * Keeps the bottom-left stack in order: toasts at the bottom, the jobs tray
 * above them, notices above the tray. Each layer publishes its height as a
 * CSS variable the next one uses.
 */
function useCornerOffsets(tray: React.RefObject<HTMLDivElement | null>) {
  useEffect(() => {
    const root = document.documentElement
    const toasts = document.querySelector<HTMLElement>('.toasts')
    // A property on the root element restyles the whole page, so it is
    // written only when it changes, and left out at zero (the CSS falls back
    // to 0px).
    const publish = (name: string, height: number) => {
      const value = height > 0 ? `${height + 8}px` : ''
      if (root.style.getPropertyValue(name) === value) return
      if (value) root.style.setProperty(name, value)
      else root.style.removeProperty(name)
    }
    const update = () => {
      publish('--rc-toasts-h', toasts?.getBoundingClientRect().height ?? 0)
      publish('--rc-tray-h', tray.current?.getBoundingClientRect().height ?? 0)
    }
    if (typeof ResizeObserver === 'undefined') {
      update()
      return
    }
    // The observer reports each box once it has been laid out (the CSS falls
    // back to 0px until then): measuring here, while the screen is still
    // being committed, made the browser lay out the whole page in the middle
    // of React's work.
    const ro = new ResizeObserver(update)
    if (toasts) ro.observe(toasts)
    if (tray.current) ro.observe(tray.current)
    return () => ro.disconnect()
  }, [tray])
}

/**
 * Cross-cutting layers for signed-in workspace screens, full-screen editors
 * included. Notices come first in the DOM so a keyboard user reaches
 * "Extend" quickly; the tray comes after the page.
 */
export function WorkspaceSystemLayer({ children }: { children: ReactNode }) {
  const rateLimit = useSystemStatus((s) => s.rateLimit)
  const tray = useRef<HTMLDivElement>(null)
  useCornerOffsets(tray)
  return (
    <>
      <div className="corner c-notices">
        <SessionGuard />
        {rateLimit ? <RateLimitNotice limit={rateLimit} /> : null}
      </div>
      {children}
      <div className="corner c-tray" ref={tray}>
        <JobsTray />
      </div>
      <GlobalShortcuts />
    </>
  )
}
