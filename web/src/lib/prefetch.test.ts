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

  it("asks for a query's follow-ups once its data says so, with the same permission check", async () => {
    const client = new QueryClient()
    client.setQueryData(workspaceMeKey('acme'), { permissions: ['providers:view'] })
    const after = vi.fn(async () => 'overview')
    const hidden = vi.fn(async () => 'x')
    prefetchFirst(client, 'acme', [
      {
        options: { queryKey: ['channels'], queryFn: async () => ({ sms: true }) },
        next: (d: { sms: boolean }) => [
          ...(d.sms ? [{ options: { queryKey: ['overview'], queryFn: after }, perm: 'providers:view' }] : []),
          { options: { queryKey: ['hidden'], queryFn: hidden }, perm: 'website:view' },
        ],
      },
    ])
    await settle()
    await settle()
    expect(after).toHaveBeenCalledTimes(1)
    expect(hidden).not.toHaveBeenCalled()
  })

  it("skips a hidden section's queries", async () => {
    const client = new QueryClient()
    client.setQueryData(workspaceMeKey('acme'), { permissions: ['earn:view'], hiddenSections: ['earn'] })
    const fn = vi.fn(async () => 'card')
    prefetchFirst(client, 'acme', [{ options: { queryKey: ['earn'], queryFn: fn }, perm: 'earn:view', section: 'earn' }])
    await settle()
    expect(fn).not.toHaveBeenCalled()
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

  it("prefetch Contacts' filters as their features ask for them", () => {
    const qs = contactsFirst('acme', { tier: 'active' })
    const byPath = new Map(qs.map((q) => [String(q.options.queryKey[1]), q]))
    // The key openapi-react-query gives the features' own $api.useQuery calls.
    expect(byPath.get('/api/v1/workspaces/{workspace}/website')?.options.queryKey).toEqual(['get', '/api/v1/workspaces/{workspace}/website', { params: { path: { workspace: 'acme' } } }])
    expect(byPath.get('/api/v1/workspaces/{workspace}/website')?.perm).toBe('website:view')
    expect(byPath.get('/api/v1/workspaces/{workspace}/products')?.perm).toBe('products:view')
    expect(byPath.get('/api/v1/workspaces/{workspace}/commerce/catalog')?.perm).toBe('campaigns:view')
    expect(byPath.get('/api/v1/workspaces/{workspace}/languages')?.perm).toBeUndefined()
    const channels = byPath.get('/api/v1/workspaces/{workspace}/channels')!
    expect(channels.perm).toBe('providers:view')
    expect(channels.next!({ channels: [{ channel: 'sms', setUp: false }] })).toEqual([])
    expect(channels.next!({ channels: [{ channel: 'sms', setUp: true }] }).map((q) => q.options.queryKey[1])).toEqual(['/api/v1/workspaces/{workspace}/sms/overview'])
  })

  it('name the permission of every Overview card', () => {
    const qs = overviewFirst('acme')
    expect(qs).toHaveLength(9)
    expect(qs.every((q) => q.perm)).toBe(true)
    expect(qs.filter((q) => q.section).map((q) => q.section)).toEqual(['transactional', 'transactional', 'earn'])
  })
})
