/**
 * Resolves the placeholders in a QA path ({campaign:Name}, {sso:Name},
 * {member:email}…) to the demo workspace's ids through the API, for the QA
 * scripts (a11y.ts, i18n-layout.ts).
 */
import type { BrowserContext } from '@playwright/test'

type Item = Record<string, unknown>
const lists: Record<string, (v: string) => [string, string, (i: Item) => boolean]> = {
  member: (email) => ['members', 'items', (m) => m.email === email],
  role: (name) => ['roles', 'roles', (r) => r.name === name],
  automation: (name) => ['automations', 'items', (a) => a.name === name],
  contact: (email) => [`contacts?q=${encodeURIComponent(email)}`, 'items', (c) => c.email === email],
  import: (which) => ['imports', 'items', (i) => (which === 'draft' ? i.status === 'draft' : i.source === which && i.status === 'done')],
  form: (name) => ['forms', 'forms', (f) => f.name === name],
  digest: (name) => ['digests', 'items', (d) => d.name === name],
  survey: (name) => ['surveys', 'items', (s) => s.name === name],
  placement: (name) => ['placement/tests', 'tests', (t) => t.name === name],
  verification: (label) => ['verification/services', 'uses', (u) => u.label === label && u.kind !== 'realtime'],
  offer: (name) => ['earn/offers', 'items', (o) => o.name === name],
  post: (title) => [`website/posts?pageSize=100&q=${encodeURIComponent(title)}`, 'posts', (p) => p.title === title],
  product: (name) => ['products', 'products', (p) => p.name === name],
  order: (n) => [`products/orders?period=all&q=${encodeURIComponent(n)}`, 'items', (o) => String(o.number) === n],
  page: (name) => ['pages', 'pages', (p) => p.name === name],
  pool: (name) => ['coupons/pools', 'items', (p) => p.name === name],
  // An import of posts by its status ({postImport:ready}, M61).
  postImport: (status) => ['website/imports', 'items', (i) => i.status === status],
  // A Meta Lead Ads connection by its name ({leadConnection:Acme Marketing}, M66).
  leadConnection: (name) => ['lead-ads/connections', 'items', (c) => c.name === name],
}

/** Looks up the ids behind {campaign:Name}, {sso:Name} and the kinds in `lists` in a path. */
export async function resolvePath(ctx: BrowserContext, app: string, path: string, workspace = 'acme-newsletter') {
  let outPath = path
  for (const m of path.matchAll(/\{(\w+):([^}]+)\}/g)) {
    const [kind, name] = [m[1]!, m[2]!]
    let id: string | undefined
    if (kind === 'campaign') {
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/campaigns`, { params: { q: name, pageSize: 100 } })
      id = ((await r.json()) as { items?: { id: string; name: string }[] }).items?.find((c) => c.name === name)?.id
    } else if (kind === 'sso') {
      const r = await ctx.request.get(`${app}/api/v1/admin/sso`)
      id = ((await r.json()) as { items?: { id: string; name: string }[] }).items?.find((p) => p.name === name)?.id
    } else if (kind === 'setup') {
      // The latest provider setup of an adapter ({setup:ses}).
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/connect/setups`, { params: { adapter: name } })
      id = ((await r.json()) as { items?: { id: string }[] }).items?.[0]?.id
    } else if (kind === 'digestTemplate') {
      // A recurring digest's template ({digestTemplate:Name}): it isn't in the campaigns list.
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/digests`)
      const digest = ((await r.json()) as { items?: { id: string; name: string }[] }).items?.find((d) => d.name === name)
      if (digest) {
        const detail = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/digests/${digest.id}`)
        id = ((await detail.json()) as { templateId?: string }).templateId
      }
    } else if (kind === 'booking') {
      // A sponsor booking by its issue ({booking:141}).
      type Row = { number: number; booking?: { id: string } }
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/sponsors/overview`, { params: { range: 'all' } })
      const body = (await r.json()) as { past?: Row[]; upcoming?: Row[] }
      id = [...(body.past ?? []), ...(body.upcoming ?? [])].find((i) => String(i.number) === name && i.booking)?.booking?.id
    } else if (kind in lists) {
      // Ids the demo generates at seed time (see scripts/qa.ts).
      const [list, key, match] = lists[kind]!(name)
      const r = await ctx.request.get(`${app}/api/v1/workspaces/${workspace}/${list}`)
      id = ((await r.json()) as Record<string, Record<string, unknown>[] | undefined>)[key]?.find(match)?.id as string | undefined
    }
    if (!id) return undefined
    outPath = outPath.replace(m[0], encodeURIComponent(id))
  }
  return outPath
}
