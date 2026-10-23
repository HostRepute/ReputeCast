import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Fragment, useCallback, type ReactNode } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { NavTabs } from '@/components/ui/Tabs'
import { T, t as tr, useT } from '@/i18n'
import { clock, orList, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import './placement.css'

export type TestList = Schemas['PlacementTestList']
export type TestRow = Schemas['PlacementTestRow']
export type TestDetail = Schemas['PlacementTestDetail']
export type ProviderResult = Schemas['PlacementProviderResult']
export type Counts = Schemas['PlacementCounts']
export type CopyView = Schemas['PlacementCopyView']
export type CopyDetail = Schemas['PlacementCopyDetail']
export type Guidance = Schemas['PlacementGuidance']
export type SeedList = Schemas['PlacementSeedList']
export type SeedView = Schemas['PlacementSeedView']
export type SeedProvider = Schemas['PlacementSeedProvider']
export type Plan = Schemas['PlacementPlan']
export type RunInput = Schemas['PlacementRunInput']
export type PlanProvider = Schemas['PlacementProviderOption']

export const base = '/api/v1/workspaces/{workspace}/placement' as const

/** Where copies land, as the boards colour them. */
export const FOLDER = { inbox: '#15803D', tab: '#93C5FD', spam: '#7F1D1D', notArrived: '#8B94A3' } as const

/** A sending provider's chart colour (tokens --chart-ses, …). */
export function providerColor(adapter: string | undefined): string {
  switch (adapter) {
    case 'ses':
      return '#4A3AA7'
    case 'postal':
      return '#E87BA4'
    case 'postmark':
      return '#008300'
    case 'brevo':
      return '#0B996E'
    case 'mailgun':
      return '#F06B66'
    case 'sendgrid':
      return '#1A82E2'
    case 'resend':
      return '#0B1220'
    case 'smtp':
      return '#5B6472'
  }
  return '#8B94A3'
}

/** Refreshes every placement screen. */
export function useRefreshPlacement() {
  const qc = useQueryClient()
  return useCallback(
    () =>
      qc.invalidateQueries({
        predicate: (q) => typeof q.queryKey[1] === 'string' && (q.queryKey[1] as string).startsWith(`${base}/`),
      }),
    [qc],
  )
}

const open = (s: string) => s === 'sending' || s === 'waiting'

/** The tests list; it refreshes while a test runs. */
export function useTests() {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    `${base}/tests`,
    { params: { path: { workspace: slug } } },
    { refetchInterval: (q) => (q.state.data?.tests.some((x) => open(x.status)) ? 4000 : false) },
  )
}

/** One test; it refreshes while copies are being looked for. */
export function useTest(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    `${base}/tests/{test}`,
    { params: { path: { workspace: slug, test: id } } },
    { refetchInterval: (q) => (q.state.data && (open(q.state.data.status) || lateWindow(q.state.data)) ? 3000 : false) },
  )
}

/** A finished test still looks for late copies for an hour. */
function lateWindow(d: TestDetail): boolean {
  return d.status === 'finished' && Boolean(d.lateUntil) && new Date(d.lateUntil!).getTime() > Date.now() && d.counts.notArrived > 0
}

export function useCopy(test: string, copy: string | undefined) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/tests/{test}/copies/{copy}`, { params: { path: { workspace: slug, test, copy: copy ?? '' } } }, { enabled: Boolean(copy) })
}

export function useSeeds() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/seeds`, { params: { path: { workspace: slug } } })
}

// ── Formatting ─────────────────────────────────────────────────────

/** "Wed 15 Oct, 10:05" */
export function dayTime(d: string): string {
  return `${shortDate(d)}, ${clock(d)}`
}

/**
 * A mailbox provider's password, named in the app's language: "App
 * password", "App-specific password". The API names it in English.
 */
export function credentialName(label: string): string {
  switch (label) {
    case 'App password':
      return tr(m.credentialAppPassword)
    case 'App-specific password':
      return tr(m.credentialAppSpecific)
    case 'Password':
      return tr(m.credentialPassword)
  }
  return label
}

/** A copy's time to arrive: "9 s", "2 min 10 s", "34 min 5 s". */
export function arrive(ms: number | null | undefined): string {
  if (ms == null) return '–'
  const s = Math.round(ms / 1000)
  if (s < 60) return tr(m.timeSeconds, { s })
  const min = Math.floor(s / 60)
  const rest = s % 60
  return rest ? tr(m.timeMinutes, { m: min, s: rest }) : tr(m.timeMinutesOnly, { m: min })
}

/** "2 min 51 s" since a moment. */
export function elapsed(from: string, now = Date.now()): string {
  return arrive(Math.max(0, now - new Date(from).getTime()))
}

/** A folder and its count in one piece ("Thư rác 2"): a wrapping line breaks between folders, never inside one. */
function glue(text: string): string {
  return text.replace(/ /g, ' ')
}

/**
 * A provider's results in a line: "Inbox 8 · Tab 2", "Spam 9 · Promotions
 * tab 1", "Inbox 9 · iCloud not checked (sign-in failed)". The largest
 * counts come first; a tab is named when nothing reached the inbox.
 */
export function resultLine(c: Counts): string {
  const parts: { n: number; text: string; order: number }[] = []
  const add = (n: number, text: string, order: number) => n > 0 && parts.push({ n, text: glue(text), order })
  add(c.inbox, tr(m.resInbox, { n: c.inbox }), 0)
  add(c.tab, c.inbox === 0 && c.tabs.length === 1 ? tr(m.resNamedTab, { tab: c.tabs[0]!, n: c.tab }) : tr(m.resTab, { n: c.tab }), 1)
  add(c.spam, tr(m.resSpam, { n: c.spam }), 2)
  add(c.notArrived, tr(m.resNotArrived, { n: c.notArrived }), 3)
  add(c.failed, tr(m.resFailed, { n: c.failed }), 4)
  // Inbox first when copies reached it; otherwise the biggest folder leads.
  if (c.inbox === 0) parts.sort((a, b) => b.n - a.n || a.order - b.order)
  const out = parts.map((p) => p.text)
  if (c.notCheckedSeeds.length) {
    out.push(tr(m.resNotChecked, { names: [...new Set(c.notCheckedSeeds.map((s) => s.name))].join(', '), reason: c.notCheckedSeeds[0]!.reason }))
  }
  return out.length ? out.join(' · ') : tr(m.resNothing)
}

/** Every folder with copies, in a fixed order: "Inbox 8 · Tab 2". */
export function plainLine(c: Counts): string {
  const out: string[] = []
  if (c.inbox) out.push(glue(tr(m.resInbox, { n: c.inbox })))
  if (c.tab) out.push(glue(tr(m.resTab, { n: c.tab })))
  if (c.spam) out.push(glue(tr(m.resSpam, { n: c.spam })))
  if (c.notArrived) out.push(glue(tr(m.resNotArrived, { n: c.notArrived })))
  return out.length ? out.join(' · ') : tr(m.resNothing)
}

/** The bar's accessible label: every folder, zeros included. */
export function barLabel(c: Counts, waiting?: boolean): string {
  const v = { inbox: c.inbox, tab: c.tab, spam: c.spam, notArrived: c.notArrived }
  return waiting && c.waiting > 0 ? tr(m.barLabelWaiting, { ...v, waiting: c.waiting }) : tr(m.barLabel, v)
}

/** Inbox, tab, spam and not-arrived segments (and the copies still on their way). */
export function PBar({ counts: c, lg, waiting }: { counts: Counts; lg?: boolean; waiting?: boolean }) {
  const segs: [number, string, string][] = [
    [c.inbox, FOLDER.inbox, '#FFFFFF'],
    [c.tab, FOLDER.tab, '#0B1220'],
    [c.spam, FOLDER.spam, '#FFFFFF'],
    [c.notArrived, FOLDER.notArrived, '#0B1220'],
  ]
  return (
    <div className={lg ? 'pl-pbar lg' : 'pl-pbar'} role="img" aria-label={barLabel(c, waiting)}>
      {segs.map(([n, bg, fg], i) =>
        n > 0 ? (
          <i key={i} style={{ flex: `${n} 1 0%`, background: bg, color: fg }}>
            {lg ? n : null}
          </i>
        ) : null,
      )}
      {waiting && c.waiting > 0 ? <i className="w" style={{ flex: `${c.waiting} 1 0%` }} /> : null}
    </div>
  )
}

/** A provider's swatch and name. */
export function Pv({ adapter, children, className }: { adapter: string | undefined; children: ReactNode; className?: string }) {
  return (
    <span className={className ? `pl-pv ${className}` : 'pl-pv'}>
      <span className="pl-sw" style={{ background: providerColor(adapter) }} />
      {children}
    </span>
  )
}

/** A swatch of a legend. */
export function Sw({ color }: { color: string }) {
  return <span className="pl-sw" style={{ background: color }} />
}

/** "SES vs Postal" with swatches, or one provider's full name. */
export function Through({ row, short }: { row: TestRow; short?: boolean }) {
  const t = useT()
  if (row.providers.length === 1 && !short) {
    const p = row.providers[0]!
    return <Pv adapter={p.adapter}>{p.name}</Pv>
  }
  return (
    <span className="inline-flex items-center gap-2 whitespace-nowrap">
      {row.providers.map((p, i) => (
        <span key={p.id} className="inline-flex items-center gap-2">
          {i > 0 ? <span className="k">{t(m.vs)}</span> : null}
          <Pv adapter={p.adapter}>{p.short}</Pv>
        </span>
      ))}
    </span>
  )
}

/** "Amazon SES vs Postal", in plain text. */
export function throughText(row: TestRow): string {
  if (row.providers.length === 1) return row.providers[0]!.name
  return row.providers.map((p) => p.name).join(` ${tr(m.vs)} `)
}

/** The content facts of the list: "Campaign · sent Tue 7 Oct · warm-up day 5". */
export function ContentFacts({ row }: { row: TestRow }) {
  const facts = contentFactList(row)
  return (
    <>
      {/* Lines break between facts, after the separator, never inside one. */}
      {facts.map((f, i) => (
        <Fragment key={i}>
          <span className="whitespace-nowrap">{f}</span>
          {i < facts.length - 1 ? ' · ' : ''}
        </Fragment>
      ))}
    </>
  )
}

export function contentFacts(row: TestRow): string {
  return contentFactList(row).join(' · ')
}

function contentFactList(row: TestRow): string[] {
  const out: string[] = []
  if (row.contentKind === 'transactional') {
    out.push(tr(m.factTransactional))
    if (row.contentKey) out.push(tr(m.factKeyVersion, { key: row.contentKey, version: row.contentVersion ?? 0 }))
    if (row.personalization) out.push(tr(m.factTestData, { name: row.personalization }))
  } else if (row.contentKind === 'template') {
    out.push(tr(m.factTemplate))
  } else if (row.contentKind === 'digest') {
    out.push(tr(m.factDigest), tr(m.factNextIssue))
  } else {
    out.push(tr(m.factCampaign))
    switch (row.contentNote) {
      case 'scheduled':
        out.push(row.origin === 'review' || !row.contentAt ? tr(m.factAsScheduled) : tr(m.factScheduled, { date: dayTime(row.contentAt) }))
        break
      case 'sent':
        out.push(row.contentAt ? tr(m.factSent, { date: shortDate(row.contentAt) }) : tr(m.factSent, { date: '' }).trim())
        break
      default:
        out.push(tr(m.factDraft))
    }
  }
  if (row.origin === 'review') out.push(tr(m.factFromReview))
  const warm = row.providers.find((p) => p.warmDay)
  if (warm?.warmDay && row.through !== 'compare') out.push(tr(m.factWarmupDay, { day: warm.warmDay }))
  return out
}

// ── Header pieces ──────────────────────────────────────────────────

/** Deliverability's section tabs: Health · Placement tests. */
export function DeliverabilityTabs() {
  const { slug } = useWorkspace()
  const t = useT()
  return (
    <NavTabs
      label={t(m.sectionsLabel)}
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: t(m.health), to: '/w/$workspace/deliverability', params: { workspace: slug }, exact: true },
        { label: t(m.placementTests), to: '/w/$workspace/deliverability/placement', params: { workspace: slug } },
      ]}
    />
  )
}

/** Tests · Seed mailboxes, under the title. */
export function PlacementPills({ current, tests, seeds }: { current: 'tests' | 'seeds'; tests?: number; seeds?: number }) {
  const { slug } = useWorkspace()
  const t = useT()
  return (
    <nav className="pl-pills" aria-label={t(m.pillsLabel)}>
      <Link to="/w/$workspace/deliverability/placement" params={{ workspace: slug }} className={current === 'tests' ? 'on' : undefined} aria-current={current === 'tests' ? 'page' : undefined} activeOptions={{ exact: true, includeSearch: false }}>
        {t(m.pillTests)}
        {tests !== undefined ? <b>{tests}</b> : null}
      </Link>
      <Link to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} className={current === 'seeds' ? 'on' : undefined} aria-current={current === 'seeds' ? 'page' : undefined}>
        {t(m.pillSeeds)}
        {seeds !== undefined ? <b>{seeds}</b> : null}
      </Link>
    </nav>
  )
}

/**
 * What the member can do here. Without placement:run, Run placement test
 * stays visible and says why; a member who only holds view permissions
 * (the Viewer role) gets the read-only header instead.
 */
export function usePlacementAccess(d: { canRun: boolean; canManage: boolean; runHelpers: string[]; manageHelpers?: string[] } | undefined) {
  const { me } = useWorkspace()
  const canRun = Boolean(d?.canRun)
  const canManage = Boolean(d?.canManage)
  const readOnly = Boolean(d) && !canRun && !canManage && me.permissions.every((p) => p.endsWith(':view'))
  const helpers = d?.runHelpers ?? []
  return {
    canRun,
    canManage,
    readOnly,
    runReason: canRun ? undefined : helpers.length ? tr(m.cantRun, { helpers: orList(helpers) }) : tr(m.cantRunNobody),
  }
}

/** The amber note of seeds whose sign-in failed or expired. */
export function SeedAttention({ seeds, onReconnect }: { seeds: SeedView[]; onReconnect?: (s: SeedView) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const first = seeds[0]
  if (!first) return null
  const at = first.statusAt ?? first.checkedAt ?? first.addedAt
  const expired = first.status === 'expired'
  return (
    <div className="note w" style={{ flexWrap: 'nowrap' }} role="note">
      <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(expired ? m.attentionExpired : m.attentionFailed, { count: seeds.length, provider: first.providerName })}</span>
        <div>
          <T
            msg={expired ? m.attentionExpiredBody : m.attentionFailedBody}
            values={{
              company: first.company,
              password: credentialName(first.passwordLabel).toLowerCase(),
              email: first.email,
              date: shortDate(at),
              time: clock(at),
              mono: (c) => <span className="mono">{c}</span>,
            }}
          />
          <div className="mt-2 flex flex-wrap gap-2">
            {onReconnect ? (
              <Button size="sm" variant="primary" onClick={() => onReconnect(first)}>
                {t(m.reconnect)}
              </Button>
            ) : (
              <ButtonLink size="sm" variant="primary" to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} search={{ reconnect: first.id }}>
                {t(m.reconnect)}
              </ButtonLink>
            )}
            <ButtonLink size="sm" to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }}>
              {t(m.seedMailboxes)}
            </ButtonLink>
          </div>
        </div>
      </div>
    </div>
  )
}

/** The board's load error: what failed, that nothing stopped, Retry. */
export function LoadError({ title, error, body, onRetry }: { title: string; error: unknown; body: (status: number) => string; onRetry: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const status = error instanceof ApiError ? error.status : 0
  const ref = error instanceof ApiError ? (error.problem.instance?.split('/').pop() ?? '') : ''
  return (
    <section className="card flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
      <Icon name="alertCircle" size={28} className="text-bad-strong" />
      <span className="text-[16px] font-bold">{title}</span>
      <span className="k max-w-[440px] text-ink-2!">{status ? body(status) : t(m.loadOfflineBody)}</span>
      <div className="flex gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace/deliverability" params={{ workspace: slug }}>
          {t(m.healthLink)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(m.request, { id: ref })}</span> : null}
    </section>
  )
}
