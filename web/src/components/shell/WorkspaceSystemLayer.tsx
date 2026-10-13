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
    const update = () => {
      const th = toasts?.getBoundingClientRect().height ?? 0
      root.style.setProperty('--rc-toasts-h', th > 0 ? `${th + 8}px` : '0px')
      const rh = tray.current?.getBoundingClientRect().height ?? 0
      root.style.setProperty('--rc-tray-h', rh > 0 ? `${rh + 8}px` : '0px')
    }
    update()
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(update)
    if (toasts) ro.observe(toasts)
    if (tray.current) ro.observe(tray.current)
    return () => ro.disconnect()
  }, [tray])
}

/**
 * Everything cross-cutting for signed-in workspace screens, full-screen
 * editors included: the session guard, the rate-limit notice, the jobs tray
 * and the keyboard shortcuts. Notices come first in the DOM so a keyboard
 * user reaches "Extend" quickly; the tray comes after the page.
 */
export function WorkspaceSystemLayer({ children }: { children: ReactNode }) {
  const status = useSystemStatus()
  const tray = useRef<HTMLDivElement>(null)
  useCornerOffsets(tray)
  return (
    <>
      <div className="corner c-notices">
        <SessionGuard />
        {status.rateLimit ? <RateLimitNotice limit={status.rateLimit} /> : null}
      </div>
      {children}
      <div className="corner c-tray" ref={tray}>
        <JobsTray />
      </div>
      <GlobalShortcuts />
    </>
  )
}
