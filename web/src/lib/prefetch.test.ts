import { QueryClient } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import { contactsFirst } from '@/features/audience/queries'
import { campaignsFirst } from '@/features/campaigns/queries'
import { overviewFirst } from '@/features/overview/queries'
import { prefetchFirst } from './prefetch'
import { workspaceMeKey } from './session'

const settle = () => new Promise((r) => setTimeout(r, 0))

describe('prefetchFirst', () => {
  it('asks only for what the workspace lets the person see, without waiting', async () => {
    const client = new QueryClient()
    client.setQueryData(workspaceMeKey('acme'), { permissions: ['campaigns:view'] })
    const seen = vi.fn(async () => 'data')
    const shown = vi.fn(async () => 'data')
    const plain = vi.fn(async () => 'data')
    const out = prefetchFirst(client, 'acme', [
      { options: { queryKey: ['a'], queryFn: seen }, perm: 'campaigns:view' },
      { options: { queryKey: ['b'], queryFn: shown }, perm: 'subscribers:view' },
      { options: { queryKey: ['c'], queryFn: plain } },
    ])
    expect(out).toBeUndefined()
    await settle()
    expect(seen).toHaveBeenCalledTimes(1)
    expect(shown).not.toHaveBeenCalled()
    expect(plain).toHaveBeenCalledTimes(1)
    expect(client.getQueryData(['a'])).toBe('data')
  })

  it('leaves fresh data alone', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000 } } })
    client.setQueryData(workspaceMeKey('acme'), { permissions: [] })
    client.setQueryData(['a'], 'cached')
    const fn = vi.fn(async () => 'new')
    prefetchFirst(client, 'acme', [{ options: { queryKey: ['a'], queryFn: fn } }])
    await settle()
    expect(fn).not.toHaveBeenCalled()
  })
})

describe('first queries', () => {
  const keyOf = (q: { options: { queryKey: readonly unknown[] } }) => JSON.stringify(q.options.queryKey)

  it('prefetch a list only as the menu opens it', () => {
    expect(contactsFirst('acme', {}).map(keyOf).some((k) => k.includes('/contacts"'))).toBe(true)
    expect(contactsFirst('acme', { tier: 'active' }).map(keyOf).some((k) => k.includes('/contacts"'))).toBe(false)
    expect(campaignsFirst('acme', {}).length).toBe(2)
    expect(campaignsFirst('acme', { page: 2 }).length).toBe(1)
  })

  it('name the permission of every Overview card', () => {
    const qs = overviewFirst('acme')
    expect(qs).toHaveLength(5)
    expect(qs.every((q) => q.perm)).toBe(true)
  })
})
