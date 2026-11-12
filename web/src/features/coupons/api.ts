import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { $api, type Schemas } from '@/api/client'
import { money as minorMoney } from '@/features/earn/format'
import { useWorkspace } from '@/lib/session'

export type Pool = Schemas['CouponPool']
export type PoolItem = Schemas['CouponPoolListItem']
export type PoolSettings = Schemas['CouponPoolSettings']
export type Batch = Schemas['CouponBatch']
export type PoolUse = Schemas['CouponPoolUse']
export type Source = Schemas['CouponSource']
export type CodesResult = Schemas['CouponCodesResult']

export const couponsBase = '/api/v1/workspaces/{workspace}/coupons' as const

/** The workspace's pools with their numbers. */
export function usePools() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${couponsBase}/pools`, { params: { path: { workspace: slug } } })
}

/** A pool's page; it refreshes every few seconds while codes are being created. */
export function usePool(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    `${couponsBase}/pools/{pool}`,
    { params: { path: { workspace: slug, pool: id } } },
    { refetchInterval: (q) => (q.state.data?.running ? 5_000 : false) },
  )
}

/** The connected stores a pool can create codes in, and why a store can't. */
export function useSources(enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${couponsBase}/sources`, { params: { path: { workspace: slug } } }, { enabled })
}

/** Refreshes the list, every pool page and the composer's pool picker. */
export function useRefreshCoupons() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey[1]
          return typeof k === 'string' && (k.startsWith(couponsBase) || k.startsWith('/api/v1/workspaces/{workspace}/campaigns'))
        },
      }),
    [qc],
  )
}

/** Money in minor units: "$18,240.00". */
export function money(minor: number, currency: string): string {
  return minorMoney(minor, (currency || 'USD').toLowerCase(), 'always')
}

/** The share of sent codes that were redeemed, as a percentage. */
export function redeemedShare(p: Pick<PoolItem, 'redeemed' | 'sent'>): number {
  return p.sent > 0 ? Math.round((p.redeemed / p.sent) * 1000) / 10 : 0
}

/** Codes a store takes per call (the progress note). */
export function perCall(platform: string | undefined): number {
  return platform === 'woocommerce' ? 100 : 250
}

/** The platform's name, never translated. */
export function platformName(platform: string | undefined): string {
  return platform === 'woocommerce' ? 'WooCommerce' : 'Shopify'
}

/** An example of the codes a store pool creates: the prefix and 8 characters. */
export function exampleCode(prefix: string): string {
  return `${prefix}7KQ4M2XA`
}

/** The code previews and test sends show. */
export function sampleCode(prefix: string): string {
  return `${prefix}SAMPLE1`
}

/** "shop.acme.io/discount/{code}": an apply address without its scheme. */
export function bareAddress(url: string): string {
  return url.replace(/^https?:\/\//, '')
}
