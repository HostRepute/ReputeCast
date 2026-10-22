import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, type Schemas } from '@/api/client'
import { money as minorMoney } from '@/features/earn/format'
import { pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'

export type StorePage = Schemas['CommerceStorePage']
export type StoreSummary = Schemas['CommerceStoreSummary']
export type SyncState = Schemas['CommerceSyncState']
export type ConnectState = Schemas['CommerceConnectState']
export type ActivityItem = Schemas['CommerceActivityItem']
export type Webhook = Schemas['CommerceWebhook']
export type Platform = StoreSummary['platform']

export const storesBase = '/api/v1/workspaces/{workspace}/stores' as const
export const commerceBase = '/api/v1/workspaces/{workspace}/commerce' as const

/** A store's page. */
export function useStore(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${storesBase}/{id}`, { params: { path: { workspace: slug, id } } })
}

/** Where connecting a store stands. */
export function useConnectState(id: string | undefined) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${storesBase}/{id}/connect`, { params: { path: { workspace: slug, id: id ?? '' } } }, { enabled: !!id })
}

/** Refreshes every store screen and the catalog. */
export function useRefreshStores() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey[1]
          return typeof k === 'string' && (k.startsWith(storesBase) || k.startsWith(commerceBase) || k.startsWith('/api/v1/workspaces/{workspace}/integrations/'))
        },
      }),
    [qc],
  )
}

/**
 * Money in minor units as the boards write it: "$64.00" for an order
 * ("always"), "$21,836" for totals ("never"), "$58" for prices ("auto").
 */
export function money(minor: number | null | undefined, currency = 'USD', cents: 'auto' | 'always' | 'never' = 'never'): string {
  return minorMoney(minor, (currency || 'USD').toLowerCase(), cents)
}

/** "23.7%", in the region format. */
export function share(v: number): string {
  return pct(Math.round(v * 10) / 10, 1)
}

/** The storefront's host: the public domain, else the store's own. */
export function storefront(p: Pick<StorePage, 'publicDomain' | 'store'>): string {
  return p.publicDomain || p.store.domain || p.store.address
}
