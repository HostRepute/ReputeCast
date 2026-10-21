import type { LinkProps } from '@tanstack/react-router'
import { $api, type Schemas } from '@/api/client'
import { useWorkspace } from '@/lib/session'

export type Overview = Schemas['PagesOverview']
export type PageRow = Schemas['PageListItem']
export type Shop = Schemas['PagesShop']
export type TemplateKey = PageRow['template']
export type PageStatus = PageRow['status']

/** The six templates of "New page", in the picker's order. */
export const TEMPLATE_KEYS = ['about', 'lead_magnet', 'launch', 'webinar', 'sales', 'blank'] as const satisfies readonly TemplateKey[]

/** The statuses, in the status filter's order. */
export const STATUSES = ['published', 'scheduled', 'draft', 'unpublished'] as const satisfies readonly PageStatus[]

/** The numbers cover the last 30 days, as the board shows them. */
export const PERIOD_DAYS = 30

export const pagesPath = '/api/v1/workspaces/{workspace}/pages'

function overviewInit(workspace: string) {
  return { params: { path: { workspace }, query: { period: '30d' as const } } }
}

/** The query key of Pages' overview (the shop's switch updates it in place). */
export function overviewKey(workspace: string) {
  return $api.queryOptions('get', pagesPath, overviewInit(workspace)).queryKey
}

/** Your pages, the numbers, the shop's switch and the theme (list-pages, 30 days). */
export function usePagesOverview() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', pagesPath, overviewInit(slug))
}

/**
 * The page builder (5.10), a route of its own: /w/$workspace/forms/pages/$pageId,
 * with ?view=preview for its preview (the composer's convention). The cast
 * keeps this compiling while the builder's route is being added.
 */
export function builderLink(workspace: string, pageId: string, preview = false): LinkProps {
  return {
    to: '/w/$workspace/forms/pages/$pageId',
    params: { workspace, pageId },
    search: preview ? { view: 'preview' } : undefined,
  } satisfies LinkProps
}

/** Your pages as the board orders them: the most viewed first, then the latest edited. */
export function sortPages(pages: readonly PageRow[]): PageRow[] {
  return [...pages].sort((a, b) => b.views - a.views || Date.parse(b.editedAt) - Date.parse(a.editedAt))
}

export interface PageFilter {
  /** Matches the name or the address. */
  q: string
  status: PageStatus | ''
  template: TemplateKey | ''
}

export const noFilter: PageFilter = { q: '', status: '', template: '' }

export function filterPages(pages: readonly PageRow[], f: PageFilter): PageRow[] {
  const q = f.q.trim().toLocaleLowerCase()
  return pages.filter(
    (p) =>
      (!f.status || p.status === f.status) &&
      (!f.template || p.template === f.template) &&
      (!q || p.name.toLocaleLowerCase().includes(q) || p.path.toLocaleLowerCase().includes(q)),
  )
}

export function isFiltered(f: PageFilter): boolean {
  return Boolean(f.q.trim() || f.status || f.template)
}

/** How many pages have each status (the status filter's counts). */
export function statusCounts(pages: readonly PageRow[]): Record<PageStatus, number> {
  const out: Record<PageStatus, number> = { published: 0, scheduled: 0, draft: 0, unpublished: 0 }
  for (const p of pages) out[p.status]++
  return out
}

/**
 * The template chips: each template in use with its count, in the order
 * the (sorted) list shows them first, as on the board.
 */
export function templateChips(sorted: readonly PageRow[], counts: Overview['templates'] = []): { template: TemplateKey; count: number }[] {
  const order: TemplateKey[] = []
  for (const p of sorted) if (!order.includes(p.template)) order.push(p.template)
  for (const c of counts) {
    const key = c.template as TemplateKey
    if (c.count > 0 && TEMPLATE_KEYS.includes(key) && !order.includes(key)) order.push(key)
  }
  return order.map((template) => ({
    template,
    count: counts.find((c) => c.template === template)?.count ?? sorted.filter((p) => p.template === template).length,
  }))
}

/** A page's views mean something once it has been published (a draft shows "–"). */
export function hasViews(p: PageRow): boolean {
  return p.views > 0 || Boolean(p.firstPublishedAt)
}

/** What a row's "…" menu offers, for someone who can edit pages. */
export type RowAction = 'viewLive' | 'preview' | 'copyLink' | 'unpublish' | 'unschedule' | 'delete'

export function rowActions(p: PageRow, can: { edit: boolean; publish: boolean }): RowAction[] {
  const out: RowAction[] = []
  if (p.status === 'published') out.push('viewLive', 'copyLink')
  else out.push('preview')
  if (can.publish && p.status === 'published') out.push('unpublish')
  if (can.publish && p.status === 'scheduled') out.push('unschedule')
  // Deleting a live page takes it off the site, which needs website:publish too.
  if (can.edit && (p.status !== 'published' || can.publish)) out.push('delete')
  return out
}

/** The address rule of the server: lowercase letters, numbers and single hyphens. */
export const SLUG_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/

export const SLUG_MAX = 80
export const NAME_MAX = 120

/**
 * The address a name suggests: "Warm-up planner" → "warm-up-planner",
 * "Giới thiệu" → "gioi-thieu" (accents dropped, as an address needs).
 */
export function slugify(name: string): string {
  const s = name
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[đĐ]/g, 'd')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return s.slice(0, SLUG_MAX).replace(/-+$/, '')
}

/** An address as it's typed: lowercase, spaces become hyphens. */
export function typedSlug(input: string): string {
  return input.toLowerCase().replace(/\s+/g, '-').slice(0, SLUG_MAX)
}

/** "news.acme.io/warm-up-planner": the host and an address, without a doubled slash. */
export function siteAddress(host: string, slug: string): string {
  return `${host.replace(/\/+$/, '')}/${slug.replace(/^\/+/, '')}`
}
