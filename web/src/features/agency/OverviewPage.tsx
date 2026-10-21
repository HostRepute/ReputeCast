import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import { Popover as RPopover } from 'radix-ui'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { formatList, useT } from '@/i18n'
import { ago } from '@/lib/format'
import { useMediaQuery } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AgencyPage } from './AgencyShell'
import type { AgencyApproval, AgencyClientRow, AgencyIncident, AgencyOverview, AgencyPulse, AgencySend } from './api'
import { m } from './messages'
import { NewClientDialog } from './NewClientDialog'
import { ClientTile, dayMonth, daysUntil, HealthCell, HealthScore, hhmm, hoursSince, isToday, PersonChip, sendDay, sendWhen, span } from './parts'
import { SetupCard } from './SetupCard'

type Filter = 'all' | 'attention' | 'mine'

/** Agency › Overview (AgencyOverview, + Tablet, + Phone): every client at a glance. */
export function OverviewPage() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/agency/overview')
  const search = useSearch({ strict: false }) as { new?: number }
  const navigate = useNavigate()
  const { slug } = useWorkspace()
  const closeNew = () => void navigate({ to: '/w/$workspace/agency', params: { workspace: slug }, search: {} })
  const d = q.data
  return (
    <AgencyPage
      title={t(m.overviewTitle)}
      subtitle={d ? t(m.overviewSubtitle, { clients: d.counts.all, attention: d.counts.attention }) : undefined}
    >
      {search.new && d?.canCreate ? <NewClientDialog open onOpenChange={(o) => (o ? null : closeNew())} /> : null}
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <OverviewSkeleton />
      ) : d.clients.length === 0 ? (
        <EmptyAgency canCreate={d.canCreate} />
      ) : (
        <>
          <div className="grid items-stretch gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(420px,100%),1fr))' }}>
            <ApprovalsCard items={d.approvals} />
            <IncidentsCard items={d.incidents} />
          </div>
          <SendsCard overview={d} />
          <ClientsCard overview={d} />
        </>
      )}
    </AgencyPage>
  )
}

function OverviewSkeleton() {
  return (
    <div className="flex flex-col gap-5" aria-busy="true">
      <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(420px,100%),1fr))' }}>
        <Skeleton w="100%" h={240} r={10} />
        <Skeleton w="100%" h={240} r={10} />
      </div>
      <Skeleton w="100%" h={220} r={10} />
      <Skeleton w="100%" h={420} r={10} />
    </div>
  )
}

function EmptyAgency({ canCreate }: { canCreate: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [open, setOpen] = useState(false)
  return (
    <section className="card flex flex-col items-center gap-3 px-7 py-11 text-center" style={{ borderStyle: 'dashed' }}>
      <span className="inline-flex h-[52px] w-[52px] items-center justify-center rounded-xl text-accent" style={{ background: '#E3F1EE' }} aria-hidden="true">
        <Icon name="overview" size={24} />
      </span>
      <h2 className="m-0 text-[18px] font-bold tracking-[-0.01em]">{t(m.emptyTitle)}</h2>
      <p className="m-0 max-w-[560px] text-ink-2">{t(m.emptyBody)}</p>
      <div className="mt-1 flex flex-wrap justify-center gap-2.5">
        {canCreate ? (
          <Button variant="primary" icon="plus" onClick={() => setOpen(true)}>
            {t(m.newClient)}
          </Button>
        ) : null}
        <Link to="/w/$workspace/agency/library" params={{ workspace: slug }} className="btn">
          {t(m.emptyTemplates)}
        </Link>
      </div>
      {open ? <NewClientDialog open onOpenChange={setOpen} /> : null}
    </section>
  )
}

/** A card of the overview's first row: a number, a caption, rows, a footnote. */
function CountCard({ label, count, caption, foot, children }: { label: string; count: number; caption: string; foot: string; children: ReactNode }) {
  return (
    <section className="card flex min-w-0 flex-col" aria-label={label}>
      <div className="flex flex-col gap-0.5 px-4 pt-3.5 pb-3">
        <span className="k">{label}</span>
        <span className="num text-[22px] leading-[1.25] font-bold tracking-[-0.01em]">{count}</span>
        <span className="k text-ink-2!">{caption}</span>
      </div>
      {children}
      <div className="mt-auto flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 px-[18px] py-3">
        <span className="k">{foot}</span>
      </div>
    </section>
  )
}

function CardRow({ to, params, tile, title, lines, badge }: { to: string; params: Record<string, string>; tile: AgencyClientRow['workspace']; title: ReactNode; lines: ReactNode; badge?: ReactNode }) {
  return (
    <Link to={to} params={params as never} className="flex items-start gap-2.5 border-t border-line-2 px-4 py-[11px] text-ink! no-underline">
      <ClientTile ws={tile} />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="font-semibold">{title}</span>
        <span className="k text-ink-2!">{lines}</span>
      </span>
      {badge ? <span className="flex flex-none items-center pt-px">{badge}</span> : null}
    </Link>
  )
}

function ApprovalsCard({ items }: { items: AgencyApproval[] }) {
  const t = useT()
  const now = new Date()
  const signoff = items.some((a) => a.clientSignoff)
  return (
    <CountCard
      label={t(m.approvalsWaiting)}
      count={items.length}
      caption={items.length === 0 ? t(m.approvalsNone) : signoff ? t(m.approvalsCaption) : t(m.approvalsCaptionStaff)}
      foot={t(m.approvalsFoot)}
    >
      {items.map((a) => {
        const days = a.sendAt ? daysUntil(a.sendAt, now) : undefined
        const waiting = a.waitingFor.length ? t(m.approvalWaitingFor, { client: a.workspace.name, names: formatList(a.waitingFor) }) : t(m.approvalWaitingNobody, { client: a.workspace.name })
        const requested = a.sendAt
          ? t(m.approvalSendsBy, { when: sendWhen(a.sendAt, a.timezone, now, true), name: a.requestedBy, ago: ago(a.requestedAt, now) })
          : t(m.approvalRequestedBy, { name: a.requestedBy, ago: ago(a.requestedAt, now) })
        return (
          <CardRow
            key={a.id}
            to="/w/$workspace/approvals/$approvalId"
            params={{ workspace: a.workspace.slug, approvalId: a.id }}
            tile={a.workspace}
            title={a.campaign}
            lines={
              <>
                {waiting}
                <br />
                {requested}
              </>
            }
            badge={days !== undefined ? <Badge tone={days <= 7 ? 'warn' : 'neutral'}>{days <= 7 ? t(m.sendsInDays, { n: days }) : t(m.inDays, { n: days })}</Badge> : null}
          />
        )
      })}
    </CountCard>
  )
}

const mailboxNames: Record<string, typeof m.mbYahoo> = { yahoo: m.mbYahoo, gmail: m.mbGmail, outlook: m.mbOutlook, icloud: m.mbIcloud }
const metricNames: Record<string, typeof m.metricDeferral> = { deferral: m.metricDeferral, bounce: m.metricBounce, complaint: m.metricComplaint }
const streamNames: Record<string, typeof m.streamAutomations> = {
  automation: m.streamAutomations,
  system: m.streamSystem,
  transactional: m.streamTransactional,
  broadcast: m.streamBroadcasts,
}

function pct(v: number) {
  return Number.isInteger(v) ? String(v) : v.toFixed(1)
}

/** What an incident was, in a sentence: the card's title and detail. */
export function incidentText(t: ReturnType<typeof useT>, i: AgencyIncident) {
  const when = `${dayMonth(i.occurredAt)}, ${hhmm(i.occurredAt)}`
  const streams = formatList([...new Set(i.failovers.map((f) => t(streamNames[f.stream] ?? m.streamAutomations)))])
  const to = i.failovers[0]?.to ?? ''
  switch (i.kind) {
    case 'auto_pause': {
      const mailbox = mailboxNames[i.mailboxProvider ?? ''] ? t(mailboxNames[i.mailboxProvider!]!) : (i.mailboxProvider ?? '')
      const metric = metricNames[i.metric ?? ''] ? t(metricNames[i.metric!]!) : (i.metric ?? '')
      let detail = t(m.incPauseDetail, { client: i.workspace.name, when, mailbox, metric, value: pct(i.value ?? 0), limit: pct(i.threshold ?? 0) })
      if (i.state === 'resumed' && i.resolvedBy && i.resolvedAt) {
        detail += ' ' + (i.withChanges ? t(m.incResumedBy, { name: i.resolvedBy, time: hhmm(i.resolvedAt) }) : t(m.incResumedPlain, { name: i.resolvedBy, time: hhmm(i.resolvedAt) }))
      } else if (i.state === 'cancelled' && i.resolvedBy && i.resolvedAt) {
        detail += ' ' + t(m.incCancelledBy, { name: i.resolvedBy, time: hhmm(i.resolvedAt) })
      }
      return {
        title: t(m.incAutoPaused, { campaign: i.campaign ?? '' }),
        detail,
        short: t(m.pauseShort, { date: dayMonth(i.occurredAt), mailbox, metric, value: pct(i.value ?? 0), limit: pct(i.threshold ?? 0) }),
        badge: t(m.badgeAutoPaused),
        tone: 'warn' as Tone,
      }
    }
    case 'provider_down': {
      const minutes = i.resolvedAt ? Math.round((new Date(i.resolvedAt).getTime() - new Date(i.occurredAt).getTime()) / 60000) : 0
      const duration = minutes >= 120 ? t(m.durationHours, { n: Math.round(minutes / 60) }) : t(m.durationMinutes, { n: minutes })
      const sp = span(i.occurredAt, i.resolvedAt)
      const detail = i.failovers.length
        ? t(m.incOutageDetail, { client: i.workspace.name, span: sp, streams, to })
        : i.broadcastHeld
          ? t(m.incOutageHeld, { client: i.workspace.name, span: sp })
          : t(m.incOutagePlain, { client: i.workspace.name, span: sp })
      return {
        title: i.resolvedAt ? t(m.incProviderDown, { provider: i.provider ?? '', duration }) : t(m.incProviderDownNow, { provider: i.provider ?? '' }),
        detail,
        short: i.failovers.length ? t(m.outageShort, { provider: i.provider ?? '', span: sp, streams, to }) : t(m.outageShortPlain, { provider: i.provider ?? '', span: sp }),
        badge: t(m.badgeProviderOutage),
        tone: 'bad' as Tone,
      }
    }
    default: {
      const title = i.kind === 'webhooks_silent' ? t(m.incWebhooksSilent, { provider: i.provider ?? '' }) : i.kind === 'sms_budget_reached' ? t(m.incSmsBudget) : t(m.incStoreDisconnected)
      const badge = i.kind === 'webhooks_silent' ? t(m.badgeWebhooks) : i.kind === 'sms_budget_reached' ? t(m.badgeSmsBudget) : t(m.badgeStore)
      return { title, detail: t(m.incGeneric, { client: i.workspace.name, when, detail: i.detail || i.title }), short: `${when} · ${i.detail || i.title}`, badge, tone: 'warn' as Tone }
    }
  }
}

function stateBadge(t: ReturnType<typeof useT>, i: AgencyIncident) {
  switch (i.state) {
    case 'resumed':
      return <Badge tone="warn">{t(m.stateResumed)}</Badge>
    case 'recovered':
      return <Badge tone="ok">{t(m.stateRecovered)}</Badge>
    case 'cancelled':
      return <Badge tone="neutral">{t(m.stateCancelled)}</Badge>
    default:
      return <Badge tone="bad">{t(m.stateOngoing)}</Badge>
  }
}

function IncidentsCard({ items }: { items: AgencyIncident[] }) {
  const t = useT()
  const ongoing = items.filter((i) => i.state === 'ongoing').length
  const caption = items.length === 0 ? t(m.incidentsNone) : ongoing ? t(m.incidentsCaptionOngoing, { n: ongoing }) : t(m.incidentsCaptionOver, { n: items.length })
  return (
    <CountCard label={t(m.incidentsTitle)} count={items.length} caption={caption} foot={t(m.incidentsFoot)}>
      {items.map((i) => {
        const x = incidentText(t, i)
        return (
          <CardRow
            key={i.id}
            to={i.kind === 'auto_pause' ? '/w/$workspace/campaigns' : '/w/$workspace/providers'}
            params={{ workspace: i.workspace.slug }}
            tile={i.workspace}
            title={x.title}
            lines={x.detail}
            badge={stateBadge(t, i)}
          />
        )
      })}
    </CountCard>
  )
}

function sendBadge(t: ReturnType<typeof useT>, s: AgencySend) {
  if (s.status === 'approval') return <Badge tone="warn">{t(m.waitingForApproval)}</Badge>
  if (s.health !== undefined && s.health !== null && s.health < 80) return <Badge tone="warn">{t(m.senderHealth, { score: s.health })}</Badge>
  return <Badge tone="neutral">{t(m.scheduled)}</Badge>
}

function SendsCard({ overview }: { overview: AgencyOverview }) {
  const t = useT()
  const next = overview.nextAfter
  const caption = next
    ? t(m.sendsCaptionThen, { client: next.workspace.name, campaign: next.name, date: dayMonth(next.sendAt, next.timezone) })
    : t(m.sendsCaption)
  return (
    <section className="card" aria-label={t(m.sendsTitle, { n: overview.sends.length })}>
      <div className="ag-c8h">
        <div>
          <h2 className="h2">{t(m.sendsTitle, { n: overview.sends.length })}</h2>
          <span className="k">{caption}</span>
        </div>
      </div>
      {overview.sends.length === 0 ? (
        <p className="k m-0 px-[18px] py-4">{t(m.sendsNone)}</p>
      ) : (
        <div className="ag-sendstrip">
          {overview.sends.map((s) => (
            <Link
              key={s.campaignId}
              to="/w/$workspace/campaigns/$campaignId/review"
              params={{ workspace: s.workspace.slug, campaignId: s.campaignId }}
              className="items-stretch! gap-2! no-underline"
            >
              <span className="flex items-baseline justify-between gap-2">
                <span className="font-bold">{sendDay(s.sendAt)}</span>
                <span className="k num">{hhmm(s.sendAt, s.timezone)}</span>
              </span>
              <span className="flex min-w-0 items-center gap-2">
                <ClientTile ws={s.workspace} size={22} />
                <span className="k font-semibold text-[#374151]!">{s.workspace.name}</span>
              </span>
              <span className="font-semibold">{s.name}</span>
              <span className="k">{t(m.people, { n: s.recipients })}</span>
              <span>{sendBadge(t, s)}</span>
            </Link>
          ))}
        </div>
      )}
    </section>
  )
}

/** Each client's row details load on their own (AgencyStates: one slow client never hides the others). */
function usePulse(id: string) {
  return $api.useQuery('get', '/api/v1/agency/clients/{workspaceId}', { params: { path: { workspaceId: id } } }, { staleTime: 30_000, retry: 1 })
}

function ClientsCard({ overview }: { overview: AgencyOverview }) {
  const t = useT()
  const [filter, setFilter] = useState<Filter>('all')
  const phone = useMediaQuery('(max-width: 767px)')
  const rows = useMemo(
    () => overview.clients.filter((c) => (filter === 'attention' ? c.attention : filter === 'mine' ? c.mine : true)),
    [overview.clients, filter],
  )
  const counts = overview.counts
  const own = overview.agencyWorkspace && filter !== 'attention' ? overview.agencyWorkspace : undefined
  return (
    <section className="card" aria-label={t(m.clientsTitle, { n: counts.all })}>
      <div className="ag-c8h">
        <div>
          <h2 className="h2">{t(m.clientsTitle, { n: counts.all })}</h2>
          <span className="k">{t(m.clientsCaption)}</span>
        </div>
      </div>
      <div className="ag-fchips" role="group" aria-label={t(m.filterLabel)}>
        {(
          [
            ['all', t(m.filterAll), counts.all],
            ['attention', t(m.filterAttention), counts.attention],
            ['mine', t(m.filterMine), counts.mine],
          ] as const
        ).map(([key, label, n]) => (
          <button key={key} type="button" className="ag-fchip" aria-pressed={filter === key} onClick={() => setFilter(key)}>
            {label} <b>{n}</b>
          </button>
        ))}
        <span className="k ml-auto">{t(m.sortedBy)}</span>
      </div>
      {phone ? (
        <div className="flex flex-col">
          {rows.map((r) => (
            <PhoneClient key={r.workspace.id} row={r} />
          ))}
          {own ? <PhoneClient row={own} agency /> : null}
          {rows.length === 0 && !own ? <p className="k m-0 px-[18px] py-4">{t(m.noClients)}</p> : null}
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="tbl cl8" aria-label={t(m.colClient)}>
            <thead>
              <tr>
                <th scope="col">{t(m.colClient)}</th>
                <th scope="col" className="r ag-hmd">
                  {t(m.colContacts)}
                </th>
                <th scope="col">{t(m.colHealth)}</th>
                <th scope="col">{t(m.colNextSend)}</th>
                <th scope="col">{t(m.colApprovals)}</th>
                <th scope="col">{t(m.colIncidents)}</th>
                <th scope="col" className="ag-hmd">
                  {t(m.colManager)}
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <ClientRow key={r.workspace.id} row={r} />
              ))}
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={7} className="k">
                    {t(m.noClients)}
                  </td>
                </tr>
              ) : null}
              {own ? (
                <>
                  <tr className="ag-group">
                    <td colSpan={7}>{t(m.agencyWorkspaceGroup)}</td>
                  </tr>
                  <ClientRow row={own} />
                </>
              ) : null}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

function nameCell(t: ReturnType<typeof useT>, row: AgencyClientRow, setup?: AgencyPulse['setup']) {
  const manager = row.accountManager?.name ?? ''
  return (
    <div className="flex min-w-0 items-start gap-2.5">
      <ClientTile ws={row.workspace} size={28} />
      <div className="flex min-w-0 flex-col items-start gap-[3px]">
        <Link to="/w/$workspace" params={{ workspace: row.workspace.slug }} className="font-semibold" style={{ color: "#0B1220" }} aria-label={t(m.openWorkspace, { name: row.workspace.name })}>
          {row.workspace.name}
        </Link>
        {row.attention ? (
          <Badge tone="warn" className="h-5! text-[11px]!">
            {t(m.needsAttention)}
          </Badge>
        ) : row.settingUp || setup ? (
          <Badge tone="neutral" className="h-5! text-[11px]!">
            {t(m.settingUp)}
          </Badge>
        ) : null}
        <span className="ag-smd k mt-0.5">{row.contacts ? t(m.contactsBy, { contacts: row.contacts, manager }) : t(m.noContactsBy, { manager })}</span>
      </div>
    </div>
  )
}

function ClientRow({ row }: { row: AgencyClientRow }) {
  const t = useT()
  const pulse = usePulse(row.workspace.id)
  const p = pulse.data
  const first = <td className={row.attention ? 'ag-attn' : undefined}>{nameCell(t, row, p?.setup)}</td>
  if (pulse.isError) {
    return (
      <tr>
        {first}
        <td colSpan={6}>
          <div className="flex flex-wrap items-center gap-2.5 rounded-lg border border-[#FCA5A5] bg-[#FFF5F5] px-3 py-2 text-[13px] text-[#991B1B]" role="alert">
            <Icon name="alertCircle" size={16} />
            <span className="min-w-0 flex-1">{t(m.rowFailed, { name: row.workspace.name, error: errorMessage(pulse.error) })}</span>
            <Button size="sm" onClick={() => pulse.refetch()}>
              {t(m.retry)}
            </Button>
          </div>
        </td>
      </tr>
    )
  }
  if (!p) {
    return (
      <tr aria-busy="true">
        {first}
        <td className="r ag-hmd">
          <span className="vh">{t(m.rowLoading, { name: row.workspace.name })}</span>
          <Skeleton w={48} h={12} />
        </td>
        <td>
          <Skeleton w={90} h={12} />
        </td>
        <td>
          <Skeleton w={150} h={12} />
        </td>
        <td>
          <Skeleton w={70} h={12} />
        </td>
        <td>
          <Skeleton w={80} h={12} />
        </td>
        <td className="ag-hmd">
          <Skeleton w={90} h={12} />
        </td>
      </tr>
    )
  }
  return (
    <tr>
      {first}
      <td className="r num ag-hmd">{p.contacts ? p.contacts.toLocaleString('en-US') : t(m.dash)}</td>
      <td>{p.setup ? <SetupHealth row={row} pulse={p} /> : <HealthCell health={p.health} />}</td>
      <td className="nx8">
        <NextSend row={row} pulse={p} />
      </td>
      <td>
        <ApprovalsCell pulse={p} />
      </td>
      <td>
        <IncidentsCell pulse={p} />
      </td>
      <td className="ag-hmd">{row.accountManager ? <PersonChip name={row.accountManager.name} color={row.accountManager.avatarColor} /> : t(m.dash)}</td>
    </tr>
  )
}

/** A new client's health: its setup, which opens the steps. */
function SetupHealth({ row, pulse }: { row: AgencyClientRow; pulse: AgencyPulse }) {
  const t = useT()
  const s = pulse.setup!
  return (
    <RPopover.Root>
      <RPopover.Trigger asChild>
        <button type="button" className="flex cursor-pointer flex-col items-start gap-0.5 border-0 bg-transparent p-0 text-left">
          <span className="font-semibold">{t(m.settingUp)}</span>
          <span className="k">{t(m.setupSteps, { done: s.done, total: s.total })}</span>
        </button>
      </RPopover.Trigger>
      <RPopover.Portal>
        <RPopover.Content className="z-50 w-[460px] max-w-[calc(100vw-24px)]" align="start" sideOffset={8} collisionPadding={12}>
          <SetupCard row={row} setup={s} />
        </RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}

function NextSend({ row, pulse }: { row: AgencyClientRow; pulse: AgencyPulse }) {
  const t = useT()
  const s = pulse.nextSend
  if (s) {
    return (
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="font-semibold">{s.name}</span>
        <span className="k">{t(m.nextSendLine, { when: sendWhen(s.sendAt, s.timezone), count: s.recipients.toLocaleString('en-US') })}</span>
      </span>
    )
  }
  if (pulse.setup && row.template) {
    const today = isToday(row.createdAt)
    return <span className="k">{today ? t(m.createdFromToday, { template: row.template }) : t(m.createdFrom, { when: dayMonth(row.createdAt), template: row.template })}</span>
  }
  if (pulse.draft) return <span className="k">{t(m.isDraft, { name: pulse.draft })}</span>
  return <span className="k">{t(m.nothingScheduled)}</span>
}

function ApprovalsCell({ pulse }: { pulse: AgencyPulse }) {
  const t = useT()
  const a = pulse.approvals
  if (!a.length) return <span className="k text-[13px]">{t(m.dash)}</span>
  const names = formatList([...new Set(a.flatMap((x) => x.waitingFor))])
  const oldest = a.reduce((min, x) => (x.requestedAt < min ? x.requestedAt : min), a[0]!.requestedAt)
  const h = hoursSince(oldest)
  return (
    <span className="flex flex-col gap-0.5">
      <span className="font-semibold">{t(m.approvalsCell, { n: a.length, names })}</span>
      <span className="k">{h >= 48 ? t(m.waitingDays, { n: Math.floor(h / 24) }) : t(m.waitingHours, { n: h })}</span>
    </span>
  )
}

function IncidentsCell({ pulse }: { pulse: AgencyPulse }) {
  const t = useT()
  const i = pulse.incidents[0]
  if (!i) return <span className="k text-[13px]">{t(m.dash)}</span>
  const x = incidentText(t, i)
  return (
    <span className="flex flex-col items-start gap-[3px]">
      <Badge tone={x.tone}>{x.badge}</Badge>
      <span className="k">{x.short}</span>
    </span>
  )
}

/** A client on a phone: a card with the same facts (AgencyOverviewPhone). */
function PhoneClient({ row, agency }: { row: AgencyClientRow; agency?: boolean }) {
  const t = useT()
  const pulse = usePulse(row.workspace.id)
  const p = pulse.data
  const manager = row.accountManager?.name ?? ''
  const h = p?.health
  const next = p?.nextSend
  const incident = p?.incidents[0]
  return (
    <Link
      to="/w/$workspace"
      params={{ workspace: row.workspace.slug }}
      className="flex flex-col gap-1.5 border-t border-line-2 px-4 py-3 text-ink! no-underline first:border-t-0"
      style={row.attention ? { boxShadow: 'inset 3px 0 0 #B45309' } : undefined}
    >
      <span className="flex items-start gap-2.5">
        <ClientTile ws={row.workspace} size={28} />
        <span className="flex min-w-0 flex-1 flex-col">
          <b>{agency ? t(m.agencyWorkspaceName, { name: row.workspace.name }) : row.workspace.name}</b>
          <span className="k">{row.contacts ? t(m.contactsBy, { contacts: row.contacts, manager }) : t(m.noContactsBy, { manager })}</span>
        </span>
        {h && h.state === 'ready' ? <HealthScore health={h} /> : null}
      </span>
      {pulse.isError ? (
        <span className="err text-[12.5px]" role="alert">
          {t(m.rowFailed, { name: row.workspace.name, error: errorMessage(pulse.error) })}
        </span>
      ) : !p ? (
        <Skeleton w="70%" h={12} />
      ) : (
        <>
          <span className="k text-ink-2!">
            {next
              ? t(m.nextLine, { name: next.name, when: sendWhen(next.sendAt, next.timezone) })
              : p.setup && row.template
                ? `${t(m.phoneNoSend)} · ${t(m.createdFromToday, { template: row.template })}`
                : p.draft
                  ? `${t(m.phoneNoSend)} · ${t(m.isDraft, { name: p.draft })}`
                  : t(m.nothingScheduled)}
          </span>
          {incident ? (
            <span>
              <Badge tone={incidentText(t, incident).tone}>{`${incidentText(t, incident).badge} · ${incidentShortest(incident)}`}</Badge>
            </span>
          ) : p.approvals.length ? (
            <span>
              <Badge tone="info">{t(m.approvalBadge, { n: p.approvals.length, names: formatList([...new Set(p.approvals.flatMap((a) => a.waitingFor))]) })}</Badge>
            </span>
          ) : p.setup ? (
            <span>
              <Badge tone="neutral">{t(m.setupBadge, { done: p.setup.done, total: p.setup.total })}</Badge>
            </span>
          ) : null}
        </>
      )}
    </Link>
  )
}

function incidentShortest(i: AgencyIncident) {
  return i.kind === 'provider_down' ? `${i.provider ?? ''} down ${span(i.occurredAt, i.resolvedAt)}` : dayMonth(i.occurredAt)
}
