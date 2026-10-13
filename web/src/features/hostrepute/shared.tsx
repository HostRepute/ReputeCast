import { useQueryClient } from '@tanstack/react-query'
import { useCallback, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { useWorkspace } from '@/lib/session'

export type Connection = Schemas['HostReputeConnection']
export type Integrations = Schemas['IntegrationsOverview']

export const integrationsPath = '/api/v1/workspaces/{workspace}/integrations' as const

/**
 * The HostRepute mark used in the board's cards and buttons: the shield in
 * accent teal. On the primary buttons it is drawn in the same teal as the
 * fill, as on the boards.
 */
export function ShieldMark({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="1.6" aria-hidden="true">
      <path d="M8 1.5 13.5 3.5v4.5c0 3-2.3 5.3-5.5 6.5C4.8 13.3 2.5 11 2.5 8V3.5z" strokeLinejoin="round" />
      <path d="m5.5 8 1.8 1.8L10.8 6" />
    </svg>
  )
}

/** Refreshes everything that shows the connection (screens and the sidebar status). */
export function useRefreshConnection() {
  const qc = useQueryClient()
  const { slug } = useWorkspace()
  return useCallback(
    () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: ['get', integrationsPath] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/deliverability'] }),
        qc.invalidateQueries({ queryKey: ['workspace-me', slug] }),
      ]),
    [qc, slug],
  )
}

/** Starts the OAuth handoff and sends the browser to HostRepute. */
export function useStartOAuth(onUrl?: (url: string) => void) {
  const { slug } = useWorkspace()
  const [pending, setPending] = useState(false)
  async function start() {
    setPending(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/oauth', { params: { path: { workspace: slug } } })
      const url = data!.authorizeUrl
      onUrl?.(url)
      window.setTimeout(() => window.location.assign(url), onUrl ? 700 : 0)
    } catch (e) {
      setPending(false)
      toast({ kind: 'error', title: "Couldn't start connecting HostRepute", detail: errorMessage(e) })
    }
  }
  return { start, pending }
}

/** The connection, for screens that need it outside Settings › Integrations. */
export function useIntegrations() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', integrationsPath, { params: { path: { workspace: slug } } })
}

/**
 * Neutral notes use the boards' `.note.n`, which in the app also matches the
 * global `.n` (bold tabular numbers); the add-on boards have no global `.n`,
 * so their neutral notes are regular weight.
 */
export const plainNote = 'font-normal [font-variant-numeric:normal]'

/** "three" for small counts, digits above ten. */
export function numberWord(n: number): string {
  return ['zero', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten'][n] ?? String(n)
}

/** Role of an asset in the assets table. */
export function roleLabel(a: Schemas['HostReputeAsset']): string {
  switch (a.role) {
    case 'sending_domain':
      return 'Sending domain'
    case 'organization_domain':
      return 'Organization domain'
    case 'dedicated_ip':
      return a.provider ? `Dedicated IP · ${a.provider}` : 'Dedicated IP'
    case 'server_hostname':
      return a.provider ? `${a.provider} server hostname` : 'Server hostname'
  }
  return a.type === 'ip' ? 'IP' : 'Domain'
}
