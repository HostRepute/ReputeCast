import type { QueryClient } from '@tanstack/react-query'
import { useQueryClient } from '@tanstack/react-query'
import { useSyncExternalStore } from 'react'
import { errorMessage, http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { t } from '@/i18n'
import { getThemeSnapshot, isThemeChoice, setThemeChoice, subscribeTheme, type ThemeChoice } from '@/lib/theme'
import { meQueryKey, type Me } from '@/lib/session'
import { m } from './messages'

const server = { choice: 'system', theme: 'light' } as const

/** The member's theme choice and the theme it comes to on this device. */
export function useTheme() {
  return useSyncExternalStore(subscribeTheme, getThemeSnapshot, () => server)
}

// One save at a time, in the order the member picked: the user menu stays
// open while they try each theme, and saves sent side by side could land in
// any order. A pick made while a save is on its way waits for it; of several
// such picks only the last is sent.
type Pick = { choice: ThemeChoice; announce: boolean }
let queued: Pick | null = null
let sending = false
// The account's theme as the server last said: where a refused save goes back to.
let confirmed: ThemeChoice = 'system'

/** user.theme of a /me answer in the query cache. */
function accountTheme(data: unknown): unknown {
  if (!data || typeof data !== 'object' || !('user' in data) || !data.user || typeof data.user !== 'object') return undefined
  return 'theme' in data.user ? data.user.theme : undefined
}

/**
 * Follows /me in the query cache: a member's theme is their account's, so a
 * choice made in another browser applies here as soon as /me says so.
 */
export function connectTheme(qc: QueryClient): void {
  qc.getQueryCache().subscribe((event) => {
    // While saves are on their way, /me (refetched for any reason) still holds
    // the theme from before: it mustn't switch the app back.
    if (event.type !== 'updated' || event.action.type !== 'success' || event.query.queryKey[0] !== 'me' || sending) return
    const theme = accountTheme(event.query.state.data)
    if (isThemeChoice(theme) && theme !== getThemeSnapshot().choice) setThemeChoice(theme)
  })
}

async function send(qc: QueryClient): Promise<void> {
  sending = true
  try {
    while (queued) {
      const job = queued
      queued = null
      try {
        const { data, error } = await http.PUT('/api/v1/me/appearance', { body: { theme: job.choice } })
        if (error || !data) throw error ?? new Error('no response')
        confirmed = data.theme
        qc.setQueryData<Me | null>(meQueryKey, (me) => (me ? { ...me, user: { ...me.user, theme: data.theme } } : me))
        if (job.announce && !queued) toast({ kind: 'success', title: t(job.choice === 'system' ? m.switchedSystem : job.choice === 'dark' ? m.switchedDark : m.switchedLight) })
      } catch (e) {
        // A newer pick is still to be sent: it decides, so nothing goes back.
        if (queued) continue
        setThemeChoice(confirmed)
        toast({ kind: 'error', title: t(m.notSaved), detail: errorMessage(e), retry: () => pick(qc, job.choice, job.announce) })
      }
    }
  } finally {
    sending = false
  }
}

function pick(qc: QueryClient, next: ThemeChoice, announce: boolean): void {
  const now = getThemeSnapshot().choice
  if (next === now && !queued) return
  // The theme shown when no save is on its way is the account's.
  if (!sending) confirmed = now
  setThemeChoice(next)
  queued = { choice: next, announce }
  if (!sending) void send(qc)
}

/**
 * Switches the theme at once and saves it to the account. When the server
 * refuses the last pick, the theme goes back to the account's with a message
 * and Retry, as switches do (design system, "Switches"). `announce` adds a
 * toast for a change made where nothing on the page shows it (the command
 * palette).
 */
export function useSaveTheme() {
  const qc = useQueryClient()
  return (next: ThemeChoice, opts: { announce?: boolean } = {}): void => pick(qc, next, opts.announce ?? false)
}
