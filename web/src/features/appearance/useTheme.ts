import type { QueryClient } from '@tanstack/react-query'
import { useQueryClient } from '@tanstack/react-query'
import { useSyncExternalStore } from 'react'
import { errorMessage, http } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { t } from '@/i18n'
import { getThemeSnapshot, isThemeChoice, setThemeChoice, subscribeTheme, type ThemeChoice } from '@/lib/theme'
import { meQueryKey } from '@/lib/session'
import { m } from './messages'

const server = { choice: 'system', theme: 'light' } as const

/** The member's theme choice and the theme it comes to on this device. */
export function useTheme() {
  return useSyncExternalStore(subscribeTheme, getThemeSnapshot, () => server)
}

// While a save is on its way, /me (refetched for any reason) still holds
// the theme from before: it mustn't switch the app back.
let saving = 0

/**
 * Follows /me in the query cache: a member's theme is their account's, so a
 * choice made in another browser applies here as soon as /me says so.
 */
export function connectTheme(qc: QueryClient): void {
  qc.getQueryCache().subscribe((event) => {
    if (event.type !== 'updated' || event.action.type !== 'success' || event.query.queryKey[0] !== 'me' || saving) return
    const theme = (event.query.state.data as { user?: { theme?: unknown } } | null | undefined)?.user?.theme
    if (isThemeChoice(theme) && theme !== getThemeSnapshot().choice) setThemeChoice(theme)
  })
}

/**
 * Switches the theme at once and saves it to the account. When the server
 * refuses, the theme goes back with a message and Retry, as switches do
 * (design system, "Switches"). `announce` adds a toast for a change made
 * where nothing on the page shows it (the command palette).
 */
export function useSaveTheme() {
  const qc = useQueryClient()
  const save = async (next: ThemeChoice, opts: { announce?: boolean } = {}): Promise<void> => {
    const before = getThemeSnapshot().choice
    if (next === before) return
    setThemeChoice(next)
    saving++
    try {
      const { data, error } = await http.PUT('/api/v1/me/appearance', { body: { theme: next } })
      if (error || !data) throw error ?? new Error('no response')
      qc.setQueryData(meQueryKey, (me: unknown) => (me && typeof me === 'object' && 'user' in me ? { ...me, user: { ...(me as { user: object }).user, theme: data.theme } } : me))
      if (opts.announce) toast({ kind: 'success', title: t(next === 'system' ? m.switchedSystem : next === 'dark' ? m.switchedDark : m.switchedLight) })
    } catch (e) {
      setThemeChoice(before)
      toast({ kind: 'error', title: t(m.notSaved), detail: errorMessage(e), retry: () => void save(next, opts) })
    } finally {
      saving--
    }
  }
  return save
}
