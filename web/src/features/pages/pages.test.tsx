import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { IntlMessageFormat } from 'intl-messageformat'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { t, type Catalog, type MessageDescriptor } from '@/i18n'
import { setDisplayTimeZone } from '@/lib/format'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import {
  filterPages,
  hasViews,
  noFilter,
  rowActions,
  siteAddress,
  SLUG_RE,
  slugify,
  sortPages,
  statusCounts,
  TEMPLATE_KEYS,
  templateChips,
  typedSlug,
  type Overview,
  type PageRow,
} from './format'
import { m } from './messages'
import { NewPageDialog } from './NewPageDialog'
import { YourPages } from './PagesTab'
import { TEMPLATES } from './templates'

const row = (patch: Partial<PageRow> & Pick<PageRow, 'id' | 'name' | 'slug' | 'template'>): PageRow => ({
  path: `/${patch.slug}`,
  url: `https://news.acme.io/${patch.slug}`,
  status: 'published',
  changes: false,
  firstPublishedAt: '2026-10-01T14:00:00Z',
  visibility: 'public',
  views: 0,
  result: { kind: 'none', count: 0, amountCents: 0, currency: 'usd', conversion: 0 },
  editedAt: '2026-10-01T14:00:00Z',
  ...patch,
})

/** The PagesList board's six pages, in the order the server sends them (first published first). */
const boardPages = (): PageRow[] => [
  row({ id: 'p5', name: 'About', slug: 'about', template: 'about', firstPublishedAt: '2026-09-29T14:00:00Z', views: 2104,
    result: { kind: 'signups', count: 38, amountCents: 0, currency: 'usd', conversion: 1.8 }, editedBy: { id: 'u1', name: 'Linh Nguyen', initials: 'LN', avatarColor: 'info' }, editedAt: '2026-09-29T14:00:00Z' }),
  row({ id: 'p2', name: 'DNS records checklist', slug: 'dns-checklist', template: 'lead_magnet', firstPublishedAt: '2026-09-22T14:00:00Z', views: 6412,
    result: { kind: 'signups', count: 1148, amountCents: 0, currency: 'usd', conversion: 17.9 }, editedBy: { id: 'u3', name: 'Sofia Rossi', initials: 'SR', avatarColor: 'ok' }, editedAt: '2026-09-30T14:00:00Z' }),
  row({ id: 'p3', name: 'Newsletter Launch Kit', slug: 'launch-kit', template: 'launch', firstPublishedAt: '2026-09-29T14:00:00Z', countdownEnded: '2026-10-06T13:00:00Z', views: 3618,
    result: { kind: 'sales', count: 81, amountCents: 107100, currency: 'usd', conversion: 2.2 }, editedBy: { id: 'u3', name: 'Sofia Rossi', initials: 'SR', avatarColor: 'ok' }, editedAt: '2026-10-06T14:00:00Z' }),
  row({ id: 'p1', name: 'The Deliverability Field Guide', slug: 'field-guide', template: 'sales', views: 9840,
    result: { kind: 'sales', count: 189, amountCents: 548100, currency: 'usd', conversion: 1.9 }, editedBy: { id: 'u3', name: 'Sofia Rossi', initials: 'SR', avatarColor: 'ok' }, editedAt: '2026-10-14T14:00:00Z' }),
  row({ id: 'p4', name: 'Live: Fix your DMARC setup in 30 minutes', slug: 'dmarc-live', template: 'webinar', firstPublishedAt: '2026-10-08T14:00:00Z', views: 2930,
    result: { kind: 'registrations', count: 486, amountCents: 0, currency: 'usd', conversion: 16.6 }, editedBy: { id: 'u2', name: 'Marcus Chen', initials: 'MC', avatarColor: 'pink' }, editedAt: '2026-10-09T14:00:00Z' }),
  row({ id: 'p6', name: 'Sponsor the Weekly Digest', slug: 'sponsor', template: 'blank', status: 'draft', firstPublishedAt: undefined, views: 0,
    editedBy: { id: 'u4', name: 'Daniel Park', initials: 'DP', avatarColor: 'warn' }, editedAt: '2026-10-15T14:41:00Z' }),
]

const overview = (patch: Partial<Overview> = {}): Overview => ({
  period: '30d',
  from: '2026-09-16T12:00:00Z',
  until: '2026-10-15T12:00:00Z',
  stats: { views: 28009, viewedPages: 6, shopViewed: true, signups: 1186, signupPages: ['About', 'DNS records checklist'], registrations: 486,
    registrationPages: ['Live: Fix your DMARC setup in 30 minutes'], sales: 311, salesCents: 749400, currency: 'usd', salesPages: 2, shopSales: true },
  pages: boardPages(),
  templates: TEMPLATE_KEYS.map((template) => ({ template, count: 1 })),
  shop: { on: true, products: 2, path: '/products', url: 'https://news.acme.io/products', canToggle: true },
  host: 'news.acme.io',
  theme: { publisher: 'Acme', logo: true, accent: '#4338CA', ground: '#F5F3FF', font: 'IBM Plex Sans' },
  websiteOn: true,
  editors: ['Marcus Chen', 'Linh Nguyen'],
  publishers: ['Sofia Rossi', 'Marcus Chen', 'Linh Nguyen'],
  canEdit: true,
  canPublish: true,
  ...patch,
})

// The board's day: dates in 2026 show without their year.
beforeAll(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2026-10-19T12:00:00Z'))
})
afterAll(() => vi.useRealTimers())

const owner = workspaceMe({
  permissions: ['website:view', 'website:edit', 'website:publish', 'website:manage', 'products:publish'],
  workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' },
})

describe('Your pages: order, filters and chips (PagesList)', () => {
  it('lists the most viewed first and drafts last, as the board does', () => {
    expect(sortPages(boardPages()).map((p) => p.slug)).toEqual(['field-guide', 'dns-checklist', 'launch-kit', 'dmarc-live', 'about', 'sponsor'])
  })

  it('orders the template chips as the list shows them, with their counts', () => {
    const chips = templateChips(sortPages(boardPages()), overview().templates)
    expect(chips.map((c) => c.template)).toEqual(['sales', 'lead_magnet', 'launch', 'webinar', 'about', 'blank'])
    expect(chips.every((c) => c.count === 1)).toBe(true)
    expect(templateChips([], [{ template: 'about', count: 2 }])).toEqual([{ template: 'about', count: 2 }])
  })

  it('filters by name or address, status and template', () => {
    const pages = boardPages()
    expect(filterPages(pages, { ...noFilter, q: 'DNS' }).map((p) => p.slug)).toEqual(['dns-checklist'])
    expect(filterPages(pages, { ...noFilter, q: '/launch' }).map((p) => p.slug)).toEqual(['launch-kit'])
    expect(filterPages(pages, { ...noFilter, status: 'draft' }).map((p) => p.slug)).toEqual(['sponsor'])
    expect(filterPages(pages, { ...noFilter, template: 'webinar' }).map((p) => p.slug)).toEqual(['dmarc-live'])
    expect(filterPages(pages, { q: 'guide', status: 'draft', template: '' })).toEqual([])
    expect(statusCounts(pages)).toEqual({ published: 5, scheduled: 0, draft: 1, unpublished: 0 })
  })

  it('shows views once a page has been published', () => {
    const [about, , , , , sponsor] = boardPages()
    expect(hasViews(about!)).toBe(true)
    expect(hasViews(sponsor!)).toBe(false)
    expect(hasViews({ ...sponsor!, views: 3 })).toBe(true)
  })
})

describe('Row actions: only what the person may do', () => {
  const [, , , guide, , sponsor] = boardPages()
  it('offers the live page, the link, unpublishing and deleting to a publisher', () => {
    expect(rowActions(guide!, { edit: true, publish: true })).toEqual(['viewLive', 'copyLink', 'unpublish', 'delete'])
    expect(rowActions(sponsor!, { edit: true, publish: true })).toEqual(['preview', 'delete'])
    expect(rowActions({ ...sponsor!, status: 'scheduled' }, { edit: true, publish: true })).toEqual(['preview', 'unschedule', 'delete'])
  })

  it('keeps a live page on the site for someone who can edit but not publish (Daniel Park)', () => {
    expect(rowActions(guide!, { edit: true, publish: false })).toEqual(['viewLive', 'copyLink'])
    expect(rowActions(sponsor!, { edit: true, publish: false })).toEqual(['preview', 'delete'])
    expect(rowActions({ ...sponsor!, status: 'scheduled' }, { edit: true, publish: false })).toEqual(['preview', 'delete'])
  })
})

describe('Addresses', () => {
  it('suggests an address from the name', () => {
    expect(slugify('Warm-up planner')).toBe('warm-up-planner')
    expect(slugify('  Live: Fix your DMARC setup in 30 minutes! ')).toBe('live-fix-your-dmarc-setup-in-30-minutes')
    expect(slugify('Giới thiệu')).toBe('gioi-thieu')
    expect(slugify('Đà Nẵng')).toBe('da-nang')
    expect(slugify('!!!')).toBe('')
    const long = slugify('word '.repeat(40))
    expect(long.length).toBeLessThanOrEqual(80)
    expect(SLUG_RE.test(long)).toBe(true)
  })

  it('keeps typed addresses lowercase with hyphens, and checks them as the server does', () => {
    expect(typedSlug('Warm Up')).toBe('warm-up')
    expect(SLUG_RE.test('warm-up-planner')).toBe(true)
    expect(SLUG_RE.test('warm--up')).toBe(false)
    expect(SLUG_RE.test('-warm')).toBe(false)
    expect(SLUG_RE.test('Warm')).toBe(false)
    expect(siteAddress('news.acme.io', 'about')).toBe('news.acme.io/about')
    expect(siteAddress('news.acme.io/', '/about')).toBe('news.acme.io/about')
  })
})

describe('Pages messages', () => {
  it('are all valid ICU', () => {
    for (const d of Object.values(m)) expect(() => new IntlMessageFormat(d.message, 'en').getAst(), d.id).not.toThrow()
  })

  it('word the cards as the board does', () => {
    expect(t(m.statViewsSub, { shop: 'true', pages: 6 })).toBe('6 pages and the shop, no cookies')
    expect(t(m.statViewsSub, { shop: 'false', pages: 1 })).toBe('1 page, no cookies')
    expect(t(m.statSalesSub, { shop: 'true', pages: 2 })).toBe('2 sales pages and the shop, through Stripe')
    expect(t(m.statSalesSub, { shop: 'false', pages: 0 })).toBe('Through Stripe')
    expect(t(m.statusOption, { status: 'published', count: '5' })).toBe('Published · 5')
    expect(t(m.narrowNumbers, { kind: 'sales', views: 9840, count: 189, amount: '$5,481' })).toBe('9,840 views · 189 sales · $5,481')
    expect(t(m.narrowNumbers, { kind: 'signups', views: 2104, count: 38, amount: '' })).toBe('2,104 views · 38 signups')
    expect(t(m.narrowEdited, { status: 'draft', name: 'Daniel Park', date: '15 Oct' })).toBe('Draft · edited by Daniel Park, 15 Oct')
    expect(t(m.emptyBuiltin, { count: 14 })).toBe('Your 14 built-in pages, like Subscribe and the archive, keep working as they are.')
    expect(t(m.themeNote, { logo: 'true', publisher: 'Acme', accent: '#4338CA', ground: '#F5F3FF', font: 'IBM Plex Sans' })).toBe(
      'Pages use the theme from Branding & domain: the Acme logo, #4338CA on #F5F3FF, IBM Plex Sans. Blocks only offer the theme’s colors, fonts and spacing, so every page stays on brand and passes contrast.',
    )
  })

  it('has every template of "New page", each with a thumbnail but Blank', () => {
    expect(TEMPLATES.map((x) => x.key)).toEqual([...TEMPLATE_KEYS])
    expect(TEMPLATES.filter((x) => x.wires.length === 0).map((x) => x.key)).toEqual(['blank'])
  })
})

describe('Your pages (PagesList, PagesStates · read-only)', () => {
  it('lists the board’s pages with Edit and a row menu', async () => {
    await renderInWorkspace(<YourPages d={overview()} />, { me: owner, path: '/w/acme-newsletter/forms/pages' })
    const table = screen.getByRole('table', { name: 'Your pages · 6' })
    const rows = within(table).getAllByRole('row').slice(1)
    expect(rows.map((r) => within(r).getAllByRole('cell')[0]!.textContent)).toEqual([
      'The Deliverability Field Guide/field-guide',
      'DNS records checklist/dns-checklist',
      'Newsletter Launch Kit/launch-kit',
      'Live: Fix your DMARC setup in 30 minutes/dmarc-live',
      'About/about',
      'Sponsor the Weekly Digest/sponsor',
    ])
    expect(rows[0]).toHaveTextContent('Sales page')
    expect(rows[0]).toHaveTextContent('Published1 Oct')
    expect(rows[0]).toHaveTextContent('189 sales · $5,481')
    expect(rows[0]).toHaveTextContent('1.9% conversion')
    expect(rows[2]).toHaveTextContent('Countdown ended 6 Oct')
    expect(rows[5]).toHaveTextContent('Draftnot published yet')
    expect(within(rows[0]!).getByRole('link', { name: 'Edit The Deliverability Field Guide' })).toHaveAttribute('href', '/w/acme-newsletter/forms/pages/p1')
    expect(within(rows[0]!).getByRole('button', { name: 'More actions for The Deliverability Field Guide' })).toBeInTheDocument()
    expect(screen.getByText('Custom pages built with blocks on your theme. Pages work with or without the website.')).toBeInTheDocument()
  })

  it('dates a scheduled page in the workspace’s time zone, and an unpublished one', async () => {
    setDisplayTimeZone('UTC')
    const pages = boardPages().map((p) =>
      p.slug === 'sponsor' ? { ...p, status: 'scheduled' as const, publishAt: '2026-10-22T13:00:00Z' } : p.slug === 'about' ? { ...p, status: 'unpublished' as const, unpublishedAt: '2026-10-12T14:00:00Z' } : p,
    )
    await renderInWorkspace(<YourPages d={overview({ pages })} />, { me: owner, path: '/w/acme-newsletter/forms/pages' })
    const rows = within(screen.getByRole('table')).getAllByRole('row')
    expect(rows[5]).toHaveTextContent('Unpublished12 Oct')
    expect(rows[6]).toHaveTextContent('Scheduled22 Oct, 13:00 UTC')
    expect(screen.getByText('Scheduled · edited by Daniel Park, 15 Oct')).toBeInTheDocument()
    setDisplayTimeZone(undefined)
  })

  it('gives a Viewer View instead of Edit and the menu', async () => {
    await renderInWorkspace(<YourPages d={overview({ canEdit: false, canPublish: false })} />, { path: '/w/acme-newsletter/forms/pages' })
    expect(screen.getByText('“New page” and row actions are hidden')).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: /^View / })).toHaveLength(6)
    expect(screen.queryByRole('link', { name: /^Edit / })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^More actions/ })).not.toBeInTheDocument()
  })

  it('filters by template and says when nothing matches', async () => {
    await renderInWorkspace(<YourPages d={overview()} />, { me: owner, path: '/w/acme-newsletter/forms/pages' })
    fireEvent.click(screen.getByRole('button', { name: 'Webinar registration 1' }))
    expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2)
    fireEvent.change(screen.getByRole('searchbox', { name: 'Search pages' }), { target: { value: 'guide' } })
    expect(screen.getByRole('status')).toHaveTextContent('No pages match')
    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }))
    expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(7)
  })
})

describe('New page (PagesStates · the template picker)', () => {
  it('suggests the address from the name and offers a free one when it is taken', async () => {
    let body: unknown
    mockApi({
      'POST /api/v1/workspaces/acme-newsletter/pages': async (r) => {
        body = await r.json()
        return json({ status: 409, title: 'Conflict', code: 'slug_taken', detail: 'Another page already uses /warm-up-planner.', meta: { suggestion: 'warm-up-planner-2' } }, 409)
      },
    })
    await renderInWorkspace(<NewPageDialog d={overview()} template="sales" onTemplate={() => {}} onClose={() => {}} />, { me: owner, path: '/w/acme-newsletter/forms/pages' })
    const dialog = screen.getByRole('dialog', { name: 'New page' })
    expect(within(dialog).getByRole('button', { name: /^Sales page/ })).toHaveAttribute('aria-pressed', 'true')
    fireEvent.change(within(dialog).getByLabelText('Page name'), { target: { value: 'Warm-up planner' } })
    expect(within(dialog).getByLabelText('Address')).toHaveValue('warm-up-planner')
    expect(screen.getByText('as a draft at news.acme.io/warm-up-planner, visible only to your team')).toBeInTheDocument()
    expect(screen.getByText('What “Sales page” adds')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create page' }))
    const use = await within(dialog).findByRole('button', { name: 'Use /warm-up-planner-2' })
    expect(body).toEqual({ template: 'sales', name: 'Warm-up planner', slug: 'warm-up-planner' })
    expect(within(dialog).getByText(/Another page already uses \/warm-up-planner\./)).toBeInTheDocument()
    fireEvent.click(use)
    await waitFor(() => expect(within(dialog).getByLabelText('Address')).toHaveValue('warm-up-planner-2'))
    expect(within(dialog).queryByText(/Another page already uses/)).not.toBeInTheDocument()
    mockApi({})
  })

  it('asks for a name before creating anything', async () => {
    mockApi({})
    await renderInWorkspace(<NewPageDialog d={overview()} template="about" onTemplate={() => {}} onClose={() => {}} />, { me: owner, path: '/w/acme-newsletter/forms/pages' })
    fireEvent.click(screen.getByRole('button', { name: 'Create page' }))
    expect(await screen.findByText('Give the page a name.')).toBeInTheDocument()
    expect(screen.getByLabelText('Page name')).toHaveAttribute('aria-invalid', 'true')
  })
})

// X-1: pages say what the reader's role can't do, or who can, in words
// (website:edit and website:publish never show).
const permissionKey = /\b[a-z]+:(?:view|edit|send|pause|approve|publish|content|create|import|export|delete|verify|manage|sql|share|hostrepute|invite|use|run|review|connect|transfer|usage|library)\b/
const english = import.meta.glob<Record<string, Record<string, MessageDescriptor>>>('./**/*messages.ts', { eager: true })
const vietnamese = import.meta.glob<Catalog>('./**/*messages.vi.ts', { eager: true, import: 'default' })

describe("pages copy", () => {
  it('names no permission key, in English or Vietnamese', () => {
    const en = Object.values(english).flatMap((file) => Object.values(file).flatMap((group) => Object.values(group)))
    const vi = Object.values(vietnamese).flatMap((catalog) => Object.entries(catalog))
    expect(en.length).toBeGreaterThan(0)
    expect(vi.length).toBeGreaterThan(0)
    for (const d of en) {
      expect(d.message, d.id).not.toMatch(permissionKey)
      // Nor does a call site pass one in: no {perm}, {perms} or {permission} to fill.
      expect(d.message, d.id).not.toMatch(/\{perm(?:s|ission)?\}/)
    }
    for (const [id, text] of vi) expect(text, id).not.toMatch(permissionKey)
  })
})
