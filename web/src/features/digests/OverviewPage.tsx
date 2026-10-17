import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues, StackBar } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { formatList, T, useT } from '@/i18n'
import { clock, count } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { DeleteDialog, PauseDialog, PreviewDialog, SkipDialog, TestDialog, TurnOnDialog } from './Dialogs'
import { HistoryCard, IssuesCard, OpensChart } from './IssuesCard'
import { m } from './messages'
import { m as cm } from '@/features/campaigns/messages'
import { NextIssueCard, PausedCard } from './NextIssueCard'
import { DigestStateBadge, scheduleText, useDigest, useRefreshDigest, type Digest } from './shared'

type Dialogs = { kind: 'turn-on' | 'pause' | 'delete' | 'preview' | 'test' } | { kind: 'skip'; issueId: string; number: number } | null

/** 5.4 Digest overview and issues (DigestOverview, DigestStates). */
export function DigestOverviewPage({ digestId }: { digestId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useDigest(digestId)
  const [dialog, setDialog] = useState<Dialogs>(null)
  const d = q.data
  const crumbs = [{ label: t(cm.title), to: '/w/$workspace/campaigns' as const, params: { workspace: slug } }]
  if (q.isError) {
    return (
      <Page title={t(m.loadFailed)} breadcrumb={[...crumbs, { label: t(m.loadFailed) }]}>
        <ErrorPanel
          title={t(m.loadFailed)}
          actions={
            <>
              <Button variant="primary" onClick={() => void q.refetch()}>
                {t(m.retry)}
              </Button>
              <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                {t(m.backToCampaigns)}
              </ButtonLink>
            </>
          }
        >
          {errorMessage(q.error)} {t(m.loadFailedBody)}
        </ErrorPanel>
      </Page>
    )
  }
  if (!d) {
    return (
      <Page title={t(m.loading)} breadcrumb={[...crumbs, { label: '…' }]}>
        <OverviewSkeleton />
      </Page>
    )
  }
  return (
    <Page
      title={d.name}
      breadcrumb={[...crumbs, { label: d.name }]}
      badges={
        <>
          <DigestStateBadge state={d.state} mode={d.mode} />
          <span className="k max-lg:basis-full">
            {d.owner
              ? t(m.overviewSub, { owner: d.owner.name, schedule: scheduleText(d.schedule, 'lower'), audience: d.audience.label || '–', count: count(d.audience.final) })
              : t(m.overviewSubNoOwner, { schedule: scheduleText(d.schedule, 'lower'), audience: d.audience.label || '–', count: count(d.audience.final) })}
          </span>
        </>
      }
      actions={<HeaderActions d={d} onDialog={setDialog} />}
    >
      <OverviewBody d={d} onDialog={setDialog} />
      {dialog?.kind === 'turn-on' ? <TurnOnDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog?.kind === 'pause' ? <PauseDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog?.kind === 'delete' ? <DeleteDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog?.kind === 'preview' ? <PreviewDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog?.kind === 'test' ? <TestDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog?.kind === 'skip' ? <SkipDialog digestId={d.id} issueId={dialog.issueId} number={dialog.number} onOpenChange={() => setDialog(null)} /> : null}
    </Page>
  )
}

function HeaderActions({ d, onDialog }: { d: Digest; onDialog: (x: Dialogs) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const canEdit = useCan('campaigns:edit')
  const pauseReason = usePermissionReason('campaigns:pause', d.pauseHelpers)
  const editReason = usePermissionReason('campaigns:edit', d.editHelpers)
  const refresh = useRefreshDigest()
  const [checking, setChecking] = useState(false)
  const check = async () => {
    setChecking(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/check', { params: { path: { workspace: slug, digestId: d.id } } })
      refresh()
      toast({
        kind: data?.verdict === 'blocked' ? 'error' : 'success',
        title: t(m.checked, { verdict: data?.verdict ?? 'ready' }),
        action: data?.campaignId
          ? { label: t(m.openReview), onClick: () => void navigate({ to: '/w/$workspace/campaigns/$campaignId/review', params: { workspace: slug, campaignId: data.campaignId } }) }
          : undefined,
      })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setChecking(false)
    }
  }
  const params = { workspace: slug, digestId: d.id }
  // Narrower than the desktop boards, Pause, Settings, Design and Send test move into More.
  return (
    <>
      <Menu trigger={<Button icon="more" className="xl:hidden">{t(m.more)}</Button>}>
        {d.state === 'on' ? (
          <MenuItem disabled={Boolean(pauseReason)} onSelect={() => onDialog({ kind: 'pause' })}>
            {t(m.pause)}
          </MenuItem>
        ) : (
          <MenuItem disabled={Boolean(editReason)} onSelect={() => onDialog({ kind: 'turn-on' })}>
            {t(m.turnOn)}
          </MenuItem>
        )}
        <MenuItem asChild>
          <Link to="/w/$workspace/campaigns/digests/$digestId/settings" params={params}>
            {t(m.settings)}
          </Link>
        </MenuItem>
        {d.templateId ? (
          <MenuItem asChild>
            <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: d.templateId }}>
              {t(m.design)}
            </Link>
          </MenuItem>
        ) : null}
        {canEdit ? <MenuItem onSelect={() => onDialog({ kind: 'test' })}>{t(m.sendTest)}</MenuItem> : null}
        {canEdit && d.state !== 'on' ? <MenuItem onSelect={() => onDialog({ kind: 'delete' })}>{t(m.deleteDigest)}</MenuItem> : null}
      </Menu>
      {d.state === 'on' ? (
        <Button icon="pause" className="max-xl:hidden" disabledReason={pauseReason} onClick={() => onDialog({ kind: 'pause' })}>
          {t(m.pause)}
        </Button>
      ) : (
        <Button icon="play" className="max-xl:hidden" disabledReason={editReason} onClick={() => onDialog({ kind: 'turn-on' })}>
          {t(m.turnOn)}
        </Button>
      )}
      <ButtonLink to="/w/$workspace/campaigns/digests/$digestId/settings" params={params} className="max-xl:hidden">
        {t(m.settings)}
      </ButtonLink>
      {d.templateId ? (
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: d.templateId }} className="max-xl:hidden">
          {t(m.design)}
        </ButtonLink>
      ) : null}
      {canEdit ? (
        <Button className="max-xl:hidden" onClick={() => onDialog({ kind: 'test' })}>
          {t(m.sendTest)}
        </Button>
      ) : null}
      <Button onClick={() => onDialog({ kind: 'preview' })}>{t(m.previewNext)}</Button>
      {canEdit ? (
        <Button variant="primary" loading={checking} busyLabel={t(m.checking)} onClick={() => void check()}>
          {t(m.checkNext)}
        </Button>
      ) : null}
    </>
  )
}

function OverviewBody({ d, onDialog }: { d: Digest; onDialog: (x: Dialogs) => void }) {
  const t = useT()
  const canEdit = useCan('campaigns:edit')
  const canPause = useCan('campaigns:pause')
  const helpers = d.editHelpers.length ? d.editHelpers : d.sendHelpers
  const onSkip = (issueId: string, number: number) => onDialog({ kind: 'skip', issueId, number })
  return (
    <>
      {!canEdit && !canPause ? <Note tone="neutral">{t(m.readOnly, { helpers: formatList(helpers, 'disjunction') })}</Note> : null}
      <div className="grid grid-cols-[minmax(0,3fr)_minmax(0,2fr)] items-start gap-4 max-lg:grid-cols-1">
        <div className="flex min-w-0 flex-col gap-4">
          {d.state === 'paused' ? <PausedCard d={d} onTurnOn={() => onDialog({ kind: 'turn-on' })} /> : null}
          {d.state === 'off' ? <OffCard d={d} onTurnOn={() => onDialog({ kind: 'turn-on' })} /> : null}
          <NextIssueCard d={d} onSkip={onSkip} />
        </div>
        <div className="flex min-w-0 flex-col gap-4">
          <FeedCard d={d} />
          <SettingsCard d={d} />
        </div>
      </div>
      <IssuesCard d={d} onSkip={onSkip} onPreview={() => onDialog({ kind: 'preview' })} />
      <div className="grid grid-cols-2 items-start gap-4 max-lg:grid-cols-1">
        <OpensChart d={d} />
        <HistoryCard d={d} />
      </div>
    </>
  )
}

/** A digest that was never turned on. */
function OffCard({ d, onTurnOn }: { d: Digest; onTurnOn: () => void }) {
  const t = useT()
  const reason = usePermissionReason('campaigns:edit', d.editHelpers)
  return (
    <section className="card flex flex-col" aria-label={t(m.nextIssue)}>
      <div className="sec">
        <h2 className="h2">{t(m.nextIssue)}</h2>
        <DigestStateBadge state="off" mode={d.mode} />
      </div>
      <div className="pad flex flex-col gap-3">
        <p className="m-0 text-[13px]">{t(m.offBody, { schedule: scheduleText(d.schedule, 'lower') })}</p>
        <div>
          <Button variant="primary" icon="play" disabledReason={reason} onClick={onTurnOn}>
            {t(m.turnOn)}
          </Button>
        </div>
      </div>
    </section>
  )
}

/** The feed as last read: source, address, categories, what's waiting. */
function FeedCard({ d }: { d: Digest }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const reason = usePermissionReason('campaigns:edit', d.editHelpers)
  const [reading, setReading] = useState(false)
  const f = d.feed
  const read = async () => {
    setReading(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/read', { params: { path: { workspace: slug, digestId: d.id } } })
      refresh()
      if (data?.ok) toast({ kind: 'success', title: data.notModified ? t(m.readNotModified) : t(m.readDone, { count: data.new }) })
      else toast({ kind: 'error', title: data?.error?.detail ?? t(m.feedBroken, { status: 'unreachable' }) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setReading(false)
    }
  }
  const leftOut = d.filters.categories.length ? d.categoryOptions.filter((c) => !d.filters.categories.some((p) => p.toLowerCase() === c.toLowerCase())) : []
  const nextNumber = d.next?.number ?? d.sentIssues + 1
  return (
    <section className="card" aria-labelledby="digest-feed">
      <div className="sec">
        <h2 className="h2" id="digest-feed">
          {t(m.feedCard)}
        </h2>
        {f.status === 'ok' && f.lastReadAt ? (
          <Badge tone="ok" dot>
            {t(m.readAt, { time: clock(f.lastReadAt) })}
          </Badge>
        ) : (
          <Badge tone={f.status === 'unread' ? 'neutral' : 'bad'} dot={f.status !== 'unread'}>
            {t(m.feedBroken, { status: f.status })}
          </Badge>
        )}
      </div>
      <div className="pad flex flex-col gap-3">
        {f.error ? <Note tone="bad">{f.error.detail}</Note> : null}
        <KeyValues
          items={[
            [
              t(m.source),
              d.source === 'wordpress' && d.site ? (
                <span>
                  <T
                    msg={m.sourceWordPress}
                    values={{
                      host: d.site.host,
                      site: (c) => (
                        <Link to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} className="lnk">
                          {c}
                        </Link>
                      ),
                    }}
                  />
                </span>
              ) : (
                t(m.sourceUrl, { host: f.url.replace(/^https?:\/\//, '').split('/')[0] ?? '' })
              ),
            ],
            [
              t(m.address),
              <span key="a" className="flex flex-col gap-0.5">
                <span className="mono break-all">{f.url}</span>
                {f.status === 'ok' ? <span className="k">{t(m.feedKind, { kind: f.kind || 'rss', count: f.items })}</span> : null}
              </span>,
            ],
            [t(m.categories), d.filters.categories.length ? d.filters.categories.join(', ') : t(m.allCategories)],
            [t(m.leftOutLabel), leftOut.length ? leftOut.join(', ') : t(m.nothingLeftOut)],
            [t(m.lastRead), f.lastReadAt ? t(m.lastReadValue, { time: clock(f.lastReadAt), via: f.lastReadVia || 'setup' }) : '–'],
            [
              t(m.waiting),
              <span key="w" className="flex flex-col gap-0.5">
                <span>{t(m.waitingPosts, { count: f.waiting, number: nextNumber })}</span>
                {f.leftOut ? <span className="k">{t(m.leftOutCount, { count: f.leftOut })}</span> : null}
              </span>,
            ],
          ]}
        />
        <div className="flex flex-wrap gap-2">
          <Button size="sm" icon="refresh" disabledReason={reason} loading={reading} busyLabel={t(m.reading)} onClick={() => void read()}>
            {t(m.readNow)}
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/campaigns/digests/$digestId/settings" params={{ workspace: slug, digestId: d.id }} hash="feed">
            {t(m.feedSettings)}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

const routeColors: Record<string, string> = { ses: 'var(--chart-ses)', postal: 'var(--chart-postal)', postmark: 'var(--chart-postmark)' }

/** The settings at a glance: schedule, audience, From, route, sending mode, archive. */
function SettingsCard({ d }: { d: Digest }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('campaigns:edit')
  const shares = d.route.shares
  const routeText = shares.map((s) => `${s.short} ${Math.round(s.percent)}%`).join(' · ')
  return (
    <section className="card" aria-labelledby="digest-settings">
      <div className="sec">
        <h2 className="h2" id="digest-settings">
          {t(m.settingsCard)}
        </h2>
        {canEdit ? (
          <Link to="/w/$workspace/campaigns/digests/$digestId/settings" params={{ workspace: slug, digestId: d.id }} className="lnk text-[13px] font-semibold">
            {t(m.editSettings)}
          </Link>
        ) : null}
      </div>
      <div className="pad">
        <KeyValues
          items={[
            [
              t(m.schedule),
              <span key="s" className="flex flex-col gap-0.5">
                <span>{scheduleText(d.schedule)}</span>
                <span className="k">{t(m.preparedLate, { time: d.next?.prepareAt ? clock(d.next.prepareAt) : prepareClock(d), hours: d.lateHours })}</span>
              </span>,
            ],
            [t(m.audience), d.audience.empty ? '–' : t(m.audienceValue, { label: d.audience.label, count: count(d.audience.final) })],
            [t(m.from), d.sender ? `${d.sender.name} <${d.sender.email}>` : '–'],
            [
              t(m.route),
              shares.length ? (
                <span key="r" className="flex flex-col gap-1">
                  <StackBar label={routeText} height={6} segments={shares.map((s) => ({ value: s.percent, color: routeColors[s.adapter] ?? 'var(--chart-other)' }))} />
                  <span className="k">{t(m.routeValue, { stream: 'Broadcasts', shares: routeText })}</span>
                </span>
              ) : (
                '–'
              ),
            ],
            [
              t(m.sending),
              <span key="m" className="flex flex-col gap-0.5">
                <span>{d.mode === 'draft' ? t(m.sendingDraft) : t(m.sendingAuto)}</span>
                {d.owner ? <span className="k">{d.mode === 'draft' ? t(m.sendingOwnerDraft, { name: d.owner.name }) : t(m.sendingOwner, { name: d.owner.name })}</span> : null}
              </span>,
            ],
            [t(m.archive), d.archive ? t(m.archiveOn) : t(m.archiveOff)],
          ]}
        />
      </div>
    </section>
  )
}

/** The preparation clock from the schedule: send time minus the lead. */
function prepareClock(d: Digest): string {
  const [h, min] = d.schedule.time.split(':').map(Number)
  const total = (((h ?? 8) * 60 + (min ?? 0) - d.prepareHours * 60) % 1440 + 1440) % 1440
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
}

function OverviewSkeleton() {
  const t = useT()
  return (
    <div className="grid grid-cols-[minmax(0,3fr)_minmax(0,2fr)] gap-4 max-lg:grid-cols-1" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
      </span>
      <section className="card pad flex flex-col gap-4">
        <Skeleton w={180} h={14} />
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} w="80%" h={28} />
          ))}
        </div>
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} w="100%" h={36} />
        ))}
      </section>
      <section className="card pad flex flex-col gap-3">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} w="90%" h={12} />
        ))}
      </section>
    </div>
  )
}
