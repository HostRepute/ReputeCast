import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useMemo, useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { problemReference } from '@/components/shell/SystemScreens'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { cn } from '@/lib/cn'
import { andList, clock, count, orList, pct } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { day, money } from '@/features/earn/format'
import { copyText, FormsTabs, hostPath } from '@/features/forms/common'
import { BuiltinPagesCard, useBuiltinPages } from '@/features/website/BuiltinPages'
import { m as wm } from '@/features/website/messages'
import {
  builderLink,
  filterPages,
  hasViews,
  isFiltered,
  noFilter,
  pagesPath,
  PERIOD_DAYS,
  rowActions,
  slugify,
  sortPages,
  statusCounts,
  STATUSES,
  templateChips,
  usePagesOverview,
  type Overview,
  type PageFilter,
  type PageRow,
  type PageStatus,
  type TemplateKey,
} from './format'
import { m } from './messages'
import { NewPageDialog, type StartProblem } from './NewPageDialog'
import { templateInfo } from './templates'
import './pages.css'

export interface PagesSearch {
  /** The template picker is open on this template (New page). */
  new?: TemplateKey
}

/**
 * Forms & public pages › Pages (PagesList, PagesStates): the 30-day
 * numbers, your pages with their template, status, visibility, views and
 * result, the built-in pages, and the theme they all use. "New page"
 * opens the template picker; each page opens in the page builder.
 */
export function PagesTab({ search }: { search: PagesSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const q = usePagesOverview()
  const builtin = useBuiltinPages()
  const [start, setStart] = useState<{ name: string; problem?: StartProblem }>()
  const [starting, setStarting] = useState(false)
  const d = q.data
  if (!d && q.error instanceof ApiError && q.error.status === 403) throw q.error

  const pick = (template: TemplateKey | undefined) =>
    void navigate({ to: '/w/$workspace/forms/pages', params: { workspace: slug }, search: template ? { new: template } : {}, replace: true })

  // "Start with About": the About page at once; if its address is taken
  // or refused, the picker opens on About to change it.
  const startAbout = async () => {
    const name = t(m.tplAbout)
    setStarting(true)
    try {
      const { data } = await http.POST(pagesPath, { params: { path: { workspace: slug } }, body: { template: 'about', name, slug: slugify(name) || 'about' } })
      void qc.invalidateQueries({ queryKey: ['get', pagesPath] })
      if (data) void navigate(builderLink(slug, data.id) as never)
    } catch (e) {
      setStarting(false)
      if (e instanceof ApiError && e.status < 500 && e.status !== 403) {
        const free = e.problem.meta?.suggestion
        setStart({ name, problem: { slug: e.message, suggestion: typeof free === 'string' ? free : undefined } })
        pick('about')
        return
      }
      toast({ kind: 'error', title: t(m.startFailed), detail: errorMessage(e) })
    }
  }

  // Sign in and Account exist while the website is on.
  const pages = d && !d.websiteOn ? builtin.filter((p) => p.key !== 'signin' && p.key !== 'account') : builtin
  const actions = d ? (
    <>
      {d.pages.length > 0 ? <span className="k max-md:hidden">{t(m.period, { days: PERIOD_DAYS, from: day(d.from), until: day(d.until) })}</span> : null}
      {d.canEdit ? (
        <ButtonLink to="/w/$workspace/forms/pages" params={{ workspace: slug }} search={{ new: 'about' }} variant="primary">
          <Icon name="plus" size={14} strokeWidth={2} />
          {t(m.newPage)}
        </ButtonLink>
      ) : null}
    </>
  ) : null

  return (
    <Page title={t(m.screenTitle)} documentTitle={t(m.documentTitle)} filters={<FormsTabs />} actions={actions}>
      {!d ? (
        q.isError ? (
          <LoadError error={q.error} onRetry={() => void q.refetch()} />
        ) : (
          <PagesLoading />
        )
      ) : (
        <>
          {d.canEdit ? null : <ReadOnlyNote d={d} />}
          {d.pages.length === 0 ? (
            <EmptyPages d={d} builtinCount={pages.length} starting={starting} onStartAbout={() => void startAbout()} />
          ) : (
            <>
              <Stats d={d} />
              <YourPages d={d} />
            </>
          )}
          <BuiltinPagesCard pages={pages} />
          <ThemeNote d={d} />
          {d.canEdit && search.new ? (
            <NewPageDialog
              d={d}
              template={search.new}
              initialName={start?.name}
              initialProblem={start?.problem}
              onTemplate={pick}
              onClose={() => {
                setStart(undefined)
                pick(undefined)
              }}
            />
          ) : null}
        </>
      )}
    </Page>
  )
}

/** Views, signups, registrations and sales over the period. */
function Stats({ d }: { d: Overview }) {
  const t = useT()
  const s = d.stats
  return (
    <div className="pg-stats">
      <Stat label={t(m.statViews, { days: PERIOD_DAYS })} value={count(s.views)} sub={t(m.statViewsSub, { shop: String(s.shopViewed), pages: s.viewedPages })} />
      <Stat label={t(m.statSignups)} value={count(s.signups)} sub={s.signupPages.length ? andList(s.signupPages) : t(m.statSignupsNone)} />
      <Stat label={t(m.statRegistrations)} value={count(s.registrations)} sub={s.registrationPages.length ? andList(s.registrationPages) : t(m.statRegistrationsNone)} />
      <Stat
        label={t(m.statSales)}
        value={t(m.statSalesValue, { count: count(s.sales), amount: money(s.salesCents, s.currency, 'never') })}
        sub={t(m.statSalesSub, { shop: String(s.shopSales), pages: s.salesPages })}
      />
    </div>
  )
}

function Stat({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="card pg-stat">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      <span className="s">{sub}</span>
    </div>
  )
}

type Confirm = { kind: 'unpublish' | 'delete'; page: PageRow }

/** Your pages: search, status and template filters, the table (the list on tablets and phones). */
export function YourPages({ d }: { d: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const titleId = useId()
  const [filter, setFilter] = useState<PageFilter>(noFilter)
  const [confirm, setConfirm] = useState<Confirm>()
  const sorted = useMemo(() => sortPages(d.pages), [d.pages])
  const rows = filterPages(sorted, filter)
  const counts = statusCounts(d.pages)
  const chips = templateChips(sorted, d.templates)
  const set = (patch: Partial<PageFilter>) => setFilter((f) => ({ ...f, ...patch }))
  const refresh = () => qc.invalidateQueries({ queryKey: ['get', pagesPath] })
  const path = (p: PageRow) => ({ params: { path: { workspace: slug, page: p.id } } })

  const unschedule = async (p: PageRow) => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/unpublish', { ...path(p), body: {} })
      toast({ title: t(m.unscheduled, { name: p.name }) })
      void refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(m.unscheduleFailed, { name: p.name }), detail: errorMessage(e) })
    }
  }

  return (
    <section className="card" aria-labelledby={titleId}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id={titleId}>
            {t(m.yourPages, { count: d.pages.length })}
          </h2>
          <span className="k">{d.canEdit ? t(m.yourPagesIntro) : t(m.yourPagesReadOnly)}</span>
        </div>
      </div>
      <div className="pg-filters">
        <label className="pg-search">
          <Icon name="search" size={15} strokeWidth={1.7} />
          <input type="search" aria-label={t(m.searchLabel)} placeholder={t(m.searchPlaceholder)} value={filter.q} onChange={(e) => set({ q: e.target.value })} />
        </label>
        <select className="in pg-status" aria-label={t(m.statusLabel)} value={filter.status} onChange={(e) => set({ status: e.target.value as PageStatus | '' })}>
          <option value="">{t(m.statusAll)}</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {t(m.statusOption, { status: s, count: count(counts[s]) })}
            </option>
          ))}
        </select>
      </div>
      <div className="pg-chips" role="group" aria-label={t(m.templateLabel)}>
        <button type="button" className={cn('pg-chip', !filter.template && 'on')} aria-pressed={!filter.template} onClick={() => set({ template: '' })}>
          {t(m.templateAll)} <b>{count(d.pages.length)}</b>
        </button>
        {chips.map((c) => (
          <button
            key={c.template}
            type="button"
            className={cn('pg-chip', filter.template === c.template && 'on')}
            aria-pressed={filter.template === c.template}
            onClick={() => set({ template: c.template })}
          >
            {t(templateInfo(c.template).name)} <b>{count(c.count)}</b>
          </button>
        ))}
      </div>
      {rows.length === 0 && isFiltered(filter) ? (
        <div className="pg-none" role="status">
          <span className="font-semibold">{t(m.noMatchTitle)}</span>
          <span className="k">{t(m.noMatchBody)}</span>
          <Button size="sm" onClick={() => setFilter(noFilter)}>
            {t(m.clearFilters)}
          </Button>
        </div>
      ) : (
        <>
          <div className="pg-wide">
            <Table className="pg-tbl" caption={t(m.yourPages, { count: d.pages.length })}>
              <thead>
                <tr>
                  <Th>{t(m.colPage)}</Th>
                  <Th>{t(m.colTemplate)}</Th>
                  <Th>{t(m.colStatus)}</Th>
                  <Th>{t(m.colVisibility)}</Th>
                  <Th align="right">{t(m.colViews, { days: PERIOD_DAYS })}</Th>
                  <Th>{t(m.colResult)}</Th>
                  <Th>{t(m.colEdited)}</Th>
                  <Th>
                    <span className="vh">{t(m.colActions)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {rows.map((p) => (
                  <tr key={p.id}>
                    <Td>
                      <span className="pg-name">
                        <Link {...builderLink(slug, p.id)}>{p.name}</Link>
                        <span className="ws-addr">{p.path}</span>
                      </span>
                    </Td>
                    <Td>{t(templateInfo(p.template).name)}</Td>
                    <Td>
                      <StatusCell p={p} />
                    </Td>
                    <Td>
                      <Visibility p={p} />
                    </Td>
                    <Td align="right" className="tabular">
                      {hasViews(p) ? count(p.views) : <span className="k">–</span>}
                    </Td>
                    <Td>
                      <ResultCell p={p} />
                    </Td>
                    <Td>
                      <EditedCell p={p} />
                    </Td>
                    <Td align="right">
                      <RowActions p={p} d={d} onConfirm={setConfirm} onUnschedule={(x) => void unschedule(x)} />
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
          <div className="pg-narrow">
            {rows.map((p) => (
              <Link key={p.id} {...builderLink(slug, p.id)} className="pg-nrow">
                <span className="hd">
                  <b>{p.name}</b>
                  <StatusBadge status={p.status} />
                </span>
                <span className="ad">
                  <span className="ws-addr">{p.path}</span>
                  <span className="k">{t(m.withTemplate, { template: t(templateInfo(p.template).name) })}</span>
                </span>
                <span className="k ln">{narrowLine(t, p)}</span>
              </Link>
            ))}
          </div>
        </>
      )}
      <div className="ws-foot">
        <span className="k flex-[1_1_420px]">{t(m.foot)}</span>
        <Link className="pg-lnk" to="/w/$workspace/audience/growth" params={{ workspace: slug }}>
          {t(m.growthSources)}
          <Icon name="arrowRight" size={14} />
        </Link>
      </div>
      {confirm?.kind === 'unpublish' ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setConfirm(undefined)}
          tone="warn"
          badge={t(m.unpublishBadge)}
          title={t(m.unpublishTitle, { name: confirm.page.name })}
          description={t(m.unpublishBody, { address: hostPath(confirm.page.url) })}
          confirmLabel={t(m.unpublishConfirm)}
          busyLabel={t(m.unpublishing)}
          onConfirm={async () => {
            await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/unpublish', { ...path(confirm.page), body: {} })
            toast({ title: t(m.unpublished, { name: confirm.page.name }) })
            void refresh()
          }}
        />
      ) : null}
      {confirm?.kind === 'delete' ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setConfirm(undefined)}
          tone="bad"
          badge={t(m.deleteBadge)}
          title={t(m.deleteTitle, { name: confirm.page.name })}
          description={confirm.page.status === 'published' ? t(m.deleteBodyLive, { address: hostPath(confirm.page.url) }) : t(m.deleteBodyDraft)}
          confirmLabel={t(m.deleteConfirm)}
          busyLabel={t(m.deleting)}
          onConfirm={async () => {
            await http.DELETE('/api/v1/workspaces/{workspace}/pages/{page}', path(confirm.page))
            toast({ kind: 'deleted', title: t(m.deleted, { name: confirm.page.name }) })
            void refresh()
          }}
        />
      ) : null}
    </section>
  )
}

const statusBadge: Record<PageStatus, [Tone, MessageDescriptor]> = {
  published: ['ok', m.statusPublished],
  scheduled: ['info', m.statusScheduled],
  draft: ['neutral', m.statusDraft],
  unpublished: ['neutral', m.statusUnpublished],
}

function StatusBadge({ status }: { status: PageStatus }) {
  const t = useT()
  const [tone, label] = statusBadge[status]
  return <Badge tone={tone}>{t(label)}</Badge>
}

/** The badge and its date: "1 Oct", "not published yet", "Countdown ended 6 Oct". */
function StatusCell({ p }: { p: PageRow }) {
  const t = useT()
  // Times show in the workspace's time zone, named as WebsitePosts does ("16 Oct, 17:00 America/New_York").
  const tz = useWorkspace().me.timezone
  let lines: ReactNode = null
  switch (p.status) {
    case 'published': {
      const at = p.firstPublishedAt ?? p.publishedAt
      const ended = p.countdownEnded ? t(m.countdownEnded, { date: day(p.countdownEnded) }) : null
      lines =
        at && ended ? (
          <>
            {day(at)}
            <br />
            {ended}
          </>
        ) : at ? (
          day(at)
        ) : (
          ended
        )
      break
    }
    case 'scheduled':
      lines = p.publishAt ? t(m.scheduledFor, { date: day(p.publishAt), time: clock(p.publishAt), tz }) : null
      break
    case 'draft':
      lines = t(m.notPublishedYet)
      break
    case 'unpublished':
      lines = p.unpublishedAt ? day(p.unpublishedAt) : null
      break
  }
  return (
    <span className="pg-stk">
      <StatusBadge status={p.status} />
      {lines ? <span className="k">{lines}</span> : null}
    </span>
  )
}

const visibility: Record<PageRow['visibility'], [IconName, MessageDescriptor]> = {
  public: ['globe', m.visPublic],
  members: ['user', m.visMembers],
  paid: ['star', m.visPaid],
}

function Visibility({ p }: { p: PageRow }) {
  const t = useT()
  const [icon, label] = visibility[p.visibility]
  return (
    <span className="pg-vis">
      <Icon name={icon} size={14} strokeWidth={1.7} />
      {t(label)}
    </span>
  )
}

/** "189 sales · $5,481" over "1.9% conversion"; "–" for a page that brings nothing in yet. */
function ResultCell({ p }: { p: PageRow }) {
  const t = useT()
  const r = p.result
  if (r.kind === 'none') return <span className="k">–</span>
  return (
    <span className="pg-stk">
      <span className="pg-res">{resultText(t, p)}</span>
      {p.views > 0 ? <span className="k">{t(m.conversion, { rate: pct(r.conversion) })}</span> : null}
    </span>
  )
}

type Tr = ReturnType<typeof useT>

function resultText(t: Tr, p: PageRow): string {
  const r = p.result
  switch (r.kind) {
    case 'sales':
      return t(m.resultSales, { count: r.count, amount: money(r.amountCents, r.currency, 'never') })
    case 'signups':
      return t(m.resultSignups, { count: r.count })
    case 'registrations':
      return t(m.resultRegistrations, { count: r.count })
  }
  return ''
}

/** A row of the tablet and phone list: its numbers, or who's drafting it. */
function narrowLine(t: Tr, p: PageRow): string {
  if (hasViews(p)) {
    const r = p.result
    return t(m.narrowNumbers, { kind: r.kind, views: p.views, count: r.count, amount: money(r.amountCents, r.currency, 'never') })
  }
  return p.editedBy ? t(m.narrowEdited, { status: p.status, name: p.editedBy.name, date: day(p.editedAt) }) : t(m.narrowEditedNoName, { status: p.status, date: day(p.editedAt) })
}

function EditedCell({ p }: { p: PageRow }) {
  return (
    <span className="pg-stk">
      {p.editedBy ? (
        <span className="pg-who">
          <Avatar name={p.editedBy.name} text={p.editedBy.initials} color={p.editedBy.avatarColor || undefined} size={22} fontSize={11} />
          <span>{p.editedBy.name}</span>
        </span>
      ) : null}
      <span className="k">{day(p.editedAt)}</span>
    </span>
  )
}

/** Edit and the "…" menu; a Viewer gets View. Only what the person may do is offered. */
function RowActions({ p, d, onConfirm, onUnschedule }: { p: PageRow; d: Overview; onConfirm: (c: Confirm) => void; onUnschedule: (p: PageRow) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (!d.canEdit) {
    return (
      <ButtonLink size="sm" {...builderLink(slug, p.id)} aria-label={t(m.viewPage, { name: p.name })}>
        {t(m.view)}
      </ButtonLink>
    )
  }
  const acts = rowActions(p, { edit: d.canEdit, publish: d.canPublish })
  const copy = async () => {
    const ok = await copyText(p.url)
    toast(ok ? { title: t(m.linkCopied), detail: hostPath(p.url) } : { kind: 'error', title: t(m.copyFailed), detail: p.url })
  }
  const label = t(m.moreActions, { name: p.name })
  return (
    <span className="pg-acts">
      <ButtonLink size="sm" {...builderLink(slug, p.id)} aria-label={t(m.editPage, { name: p.name })}>
        {t(m.edit)}
      </ButtonLink>
      <Menu
        label={label}
        trigger={
          <Button size="sm" iconOnly aria-label={label}>
            {/* The board's dots, bolder than the kit's "more" icon. */}
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
              <circle cx="3.5" cy="8" r=".9" />
              <circle cx="8" cy="8" r=".9" />
              <circle cx="12.5" cy="8" r=".9" />
            </svg>
          </Button>
        }
      >
        {acts.includes('viewLive') ? (
          <MenuItem asChild>
            <a href={p.url} target="_blank" rel="noopener noreferrer">
              <Icon name="external" />
              {t(m.viewLive)}
            </a>
          </MenuItem>
        ) : null}
        {acts.includes('preview') ? (
          <MenuItem asChild>
            <Link {...builderLink(slug, p.id, true)}>
              <Icon name="eye" />
              {t(m.preview)}
            </Link>
          </MenuItem>
        ) : null}
        {acts.includes('copyLink') ? (
          <MenuItem onSelect={() => void copy()}>
            <Icon name="link" />
            {t(m.copyLink)}
          </MenuItem>
        ) : null}
        {acts.includes('unpublish') ? (
          <MenuItem onSelect={() => onConfirm({ kind: 'unpublish', page: p })}>
            <Icon name="undo" />
            {t(m.unpublish)}
          </MenuItem>
        ) : null}
        {acts.includes('unschedule') ? (
          <MenuItem onSelect={() => onUnschedule(p)}>
            <Icon name="calendar" />
            {t(m.unschedule)}
          </MenuItem>
        ) : null}
        {acts.includes('delete') ? (
          <>
            <MenuSeparator />
            <MenuItem className="text-bad-fg" onSelect={() => onConfirm({ kind: 'delete', page: p })}>
              <Icon name="trash" />
              {t(m.delete)}
            </MenuItem>
          </>
        ) : null}
      </Menu>
    </span>
  )
}

/** "No pages yet. Start with About or a template." */
function EmptyPages({ d, builtinCount, starting, onStartAbout }: { d: Overview; builtinCount: number; starting: boolean; onStartAbout: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card pg-empty">
      <svg width="64" height="56" viewBox="0 0 64 56" aria-hidden="true">
        <rect x="6" y="6" width="40" height="44" rx="4" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <path d="M14 16h24M14 24h18M14 32h22" stroke="#9AA3B2" strokeWidth="2" strokeLinecap="round" />
        <circle cx="48" cy="40" r="10" fill="#fff" stroke="#0F766E" strokeWidth="2.2" />
        <path d="M48 35v10M43 40h10" stroke="#0F766E" strokeWidth="2.2" strokeLinecap="round" />
      </svg>
      <h2 className="tt">{t(m.emptyTitle)}</h2>
      <span className="ds">{t(m.emptyBody)}</span>
      {d.canEdit ? (
        <div className="mt-1 flex flex-wrap justify-center gap-2.5">
          <Button variant="primary" loading={starting} onClick={onStartAbout}>
            {t(m.startAbout)}
          </Button>
          <ButtonLink to="/w/$workspace/forms/pages" params={{ workspace: slug }} search={{ new: 'about' }}>
            {t(m.chooseTemplate)}
          </ButtonLink>
        </div>
      ) : null}
      <span className="k">{t(m.emptyBuiltin, { count: builtinCount })}</span>
    </section>
  )
}

/** A Viewer's note: what they can do, and who can edit and publish. */
function ReadOnlyNote({ d }: { d: Overview }) {
  const t = useT()
  const { me: ws } = useWorkspace()
  const name = useMe().data?.user.name
  const role = ws.workspace.roleName
  return (
    <div className="note n pg-note" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-2" />
      <div className="flex min-w-0 flex-col gap-1">
        <span className="font-bold">{name ? t(m.readOnlyTitle, { name, role }) : t(m.readOnlyRole, { role })}</span>
        <div>
          <T msg={d.editors.length ? m.readOnly : m.readOnlyNobody} values={{ helpers: orList(d.editors) }} />
        </div>
      </div>
    </div>
  )
}

/** The theme every page uses, from Branding & domain. */
function ThemeNote({ d }: { d: Overview }) {
  const { slug } = useWorkspace()
  const th = d.theme
  const link = (c: ReactNode[]) => (
    <Link to="/w/$workspace/forms/branding" params={{ workspace: slug }}>
      {c}
    </Link>
  )
  return (
    <div className="note n pg-note" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-2" />
      <div>
        {th.accent && th.ground && th.font ? (
          <T msg={m.themeNote} values={{ link, logo: String(th.logo), publisher: th.publisher, accent: th.accent, ground: th.ground, font: th.font }} />
        ) : (
          <T msg={wm.themeNote} values={{ link }} />
        )}
      </div>
    </div>
  )
}

function PagesLoading() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="card flex flex-col gap-3.5 p-[18px]">
      <div className="pg-stats">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="card pg-stat">
            <Skeleton w={90} h={12} />
            <Skeleton w={70} h={22} style={{ marginTop: 6 }} />
            <Skeleton w={120} h={10} style={{ marginTop: 6 }} />
          </div>
        ))}
      </div>
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="pg-skrow">
          <Skeleton w={160} h={14} />
          <Skeleton w={80} h={20} r={999} />
          <Skeleton h={10} style={{ flex: 1 }} />
          <Skeleton w={50} h={28} />
        </div>
      ))}
    </Loading>
  )
}

/** "Couldn't load your pages": what failed, that published pages stay online, Retry. */
function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = error instanceof ApiError ? error.status : 0
  const ref = problemReference(error)
  return (
    <section className="card flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
      <Icon name="alertCircle" size={28} className="text-bad-strong" />
      <h2 className="m-0 text-[16px] font-bold">{t(m.loadError)}</h2>
      <span className="k max-w-[420px]">{status >= 500 ? t(m.loadErrorRetried, { status }) : status ? t(m.loadErrorStatus, { status }) : t(m.loadErrorOffline)}</span>
      <div className="flex flex-wrap justify-center gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
          {t(m.statusOnOverview)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(m.request, { id: ref })}</span> : null}
    </section>
  )
}
