import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ApiError } from '@/api/client'
import { Page, type Crumb } from '@/components/shell/Page'
import { m as shell } from '@/components/shell/messages'
import { ForbiddenState, NotFoundState, problemReference, ServerErrorState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clockSeconds } from '@/lib/format'
import { useMediaQuery, useNow } from '@/lib/hooks'
import { cn } from '@/lib/cn'
import { useWorkspace } from '@/lib/session'
import { CancelSendDialog, useLiveActions, useLivePermissions, type LiveActions, type LivePermissions } from './actions'
import { canaryActive } from './canary'
import { mc } from './canary.messages'
import { CanaryView } from './Canary'
import { DecisionView } from './Decision'
import { LiveBody } from './LiveParts'
import { m } from './messages'
import { MobileLive } from './Mobile'
import { ThresholdsDialog } from './ResumeDialog'
import { freshness, headerBadges, isStarting, lostText, openIncident, openNotices, startedLine, whoLine, type Incident, type Snapshot } from './model'
import {
  CancelledView,
  ConnectingCard,
  FailedView,
  FinishedView,
  IdleView,
  IncidentBanner,
  LiveSkeleton,
  PausedBanner,
  NoticeBanners,
  ScheduledView,
  StaleBanner,
  staleText,
  StreamLostBanner,
} from './States'
import { MAX_ATTEMPTS, useLiveSend, type StreamState } from './useLiveSend'
import './live.css'

export interface LiveSearch {
  /** "live" keeps the live view while an incident waits for a decision. */
  view?: 'live'
}

/**
 * 7.6 Live sending monitor (LiveMonitor boards): the snapshot from the API,
 * kept live by the event stream.
 */
export function LiveMonitorPage({ campaignId, search }: { campaignId: string; search: LiveSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const live = useLiveSend(slug, campaignId)
  if (!live.snapshot) {
    if (live.query.isError) return <LiveError error={live.query.error} retry={() => void live.query.refetch()} campaignId={campaignId} />
    return (
      <Page title={t(m.title)} breadcrumb={crumbs(slug, t(common.loading))}>
        <LiveSkeleton />
      </Page>
    )
  }
  return <LiveScreen s={live.snapshot} campaignId={campaignId} search={search} helpers={live.helpers} stream={live.stream} apply={live.apply} />
}

function crumbs(slug: string, name: string): Crumb[] {
  return [{ label: tt(m.campaigns), to: '/w/$workspace/campaigns', params: { workspace: slug } }, { label: name }]
}

/** Which body the screen shows. */
type Mode = 'decision' | 'connecting' | 'live' | 'finished' | 'cancelled' | 'failed' | 'scheduled' | 'idle'

function modeOf(s: Snapshot, incident: Incident | undefined, view: LiveSearch['view'], now: Date): Mode {
  switch (s.state) {
    case 'finished':
      return 'finished'
    case 'cancelled':
      return 'cancelled'
    case 'failed':
      return 'failed'
    case 'scheduled':
      return 'scheduled'
    case 'idle':
      return 'idle'
  }
  if (incident && view !== 'live') return 'decision'
  if (isStarting(s, now)) return 'connecting'
  return 'live'
}

function LiveScreen({
  s,
  campaignId,
  search,
  helpers,
  stream,
  apply,
}: {
  s: Snapshot
  campaignId: string
  search: LiveSearch
  helpers: Snapshot['helpers']
  stream: StreamState & { reconnect: () => void }
  apply: (s: Snapshot) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const now = useNow(1000)
  const phone = useMediaQuery('(max-width: 767px)')
  const actions = useLiveActions(slug, campaignId, apply)
  const can = useLivePermissions(helpers)
  const [cancelling, setCancelling] = useState(false)
  const [editing, setEditing] = useState(false)
  const incident = openIncident(s)
  const mode = modeOf(s, incident, search.view, now)
  const active = ['preparing', 'sending', 'partial', 'paused'].includes(s.state)
  const lost = active && (stream.status === 'retrying' || stream.status === 'stopped') && stream.lostAt !== undefined
  // A stream that says nothing for 90 s is stale: the monitor checks every
  // 10 s and says so (LiveMonitorStates; review GS-14).
  const stale = active && !lost && stream.status === 'stale'
  const fresh = freshness(s, now, stale ? 'polling' : lost ? 'reconnecting' : 'live')
  const asOf = clockSeconds(s.at)
  const setView = (view: LiveSearch['view']) =>
    void navigate({ to: '/w/$workspace/campaigns/$campaignId/live', params: { workspace: slug, campaignId }, search: view ? { view } : {}, replace: true })
  const pause = () => void actions.pause().catch(() => undefined)
  const resume = () => void actions.resume().catch(() => undefined)
  const cancelOpen = () => setCancelling(true)

  const lostBanner =
    lost && stream.lostAt ? (
      <StreamLostBanner
        {...lostText(stream.lostAt, s.at, stream.attempt, MAX_ATTEMPTS, stream.status === 'stopped', now)}
        onReconnect={stream.reconnect}
      />
    ) : null
  const staleBanner = stale ? <StaleBanner {...staleText(s, stream.heardAt, now)} onReconnect={stream.reconnect} /> : null
  const pausedBanner =
    s.state === 'paused' && !incident ? <PausedBanner s={s} can={can} onResume={resume} onCancel={cancelOpen} busy={actions.pending} /> : null
  const incidentBanner = incident && mode !== 'decision' ? <IncidentBanner s={s} incident={incident} onDecide={() => setView(undefined)} /> : null
  const noticeBanners = active && openNotices(s).length ? <NoticeBanners s={s} actions={actions} can={can} /> : null

  const dialog = (
    <>
      <CancelSendDialog open={cancelling} onOpenChange={setCancelling} s={s} incident={incident} actions={actions} canPause={!can.pause} />
      <ThresholdsDialog open={editing} onOpenChange={setEditing} s={s} actions={actions} />
    </>
  )
  // "Edit for this send" (review GS-2): who can resume can change them.
  const editThresholds = active ? { onEdit: () => setEditing(true), disabledReason: can.resume } : undefined

  // The connecting card comes first, a canary's too (LiveMonitorStates › loading).
  const canary = canaryActive(s.canary) && (mode === 'live' || (mode === 'decision' && incident?.kind === 'canary')) ? s.canary : undefined
  if (phone && !canary && (mode === 'live' || mode === 'connecting')) {
    return (
      <>
        <MobileLive
          s={s}
          edit={editThresholds}
          streaming={!lost && !stale}
          freshness={fresh}
          actions={actions}
          incident={incident}
          connecting={mode === 'connecting'}
          banner={
            lostBanner || staleBanner || pausedBanner || incidentBanner || noticeBanners ? (
              <>
                {lostBanner}
                {staleBanner}
                {pausedBanner}
                {incidentBanner}
                {noticeBanners}
              </>
            ) : null
          }
          can={can}
          onPause={pause}
          onResume={resume}
          onCancel={cancelOpen}
          busy={actions.pending}
        />
        {dialog}
      </>
    )
  }

  let body: ReactNode
  switch (canary ? 'canary' : mode) {
    case 'canary':
      body = (
        <>
          {lostBanner}
          {staleBanner}
          <CanaryView s={s} c={canary!} actions={actions} can={can} now={now} />
        </>
      )
      break
    case 'decision':
      body = <DecisionView s={s} incident={incident!} actions={actions} can={can} onCancelAll={cancelOpen} />
      break
    case 'connecting':
      body = (
        <>
          {lostBanner}
          <ConnectingCard s={s} />
        </>
      )
      break
    case 'finished':
      body = <FinishedView s={s} campaignId={campaignId} actions={actions} can={can} />
      break
    case 'cancelled':
      body = <CancelledView s={s} campaignId={campaignId} />
      break
    case 'failed':
      body = <FailedView s={s} campaignId={campaignId} />
      break
    case 'scheduled':
      body = <ScheduledView s={s} campaignId={campaignId} />
      break
    case 'idle':
      body = <IdleView s={s} campaignId={campaignId} />
      break
    default:
      body = (
        <>
          {lostBanner}
          {staleBanner}
          {pausedBanner}
          {incidentBanner}
          {noticeBanners}
          <LiveBody
            s={s}
            campaignId={campaignId}
            streaming={!lost && !stale}
            stale={lost}
            polling={stale ? fresh.text : undefined}
            asOf={asOf}
            incident={incident}
            edit={editThresholds}
            actions={actions}
            can={can}
          />
        </>
      )
  }

  return (
    <Page
      title={t(m.title)}
      documentTitle={t(m.documentTitle, { name: s.name })}
      breadcrumb={crumbs(slug, s.name)}
      subtitle={startedLine(s) || undefined}
      badges={
        <>
          {canary ? (
            <Badge tone={canary.phase === 'held' ? 'bad' : 'info'} dot>
              {t(mc.badge, { phase: canary.phase })}
            </Badge>
          ) : (
            <HeaderBadges s={s} incident={incident} mode={mode} />
          )}
          {active ? (
            <span className={cn('lv-fresh', fresh.stale && 'stale')}>
              <span className="dot" aria-hidden="true" />
              {fresh.text}
            </span>
          ) : null}
        </>
      }
      actions={
        canary ? (
          <span className="k lv-caption">{whoLine(s, 'canary')}</span>
        ) : (
          <HeaderActions s={s} mode={mode} incident={incident} can={can} actions={actions} onPause={pause} onCancel={cancelOpen} onBack={() => setView('live')} />
        )
      }
    >
      {body}
      {dialog}
    </Page>
  )
}

/** The state words after the breadcrumb (none once a send has ended, as on LiveMonitorStates). */
function HeaderBadges({ s, incident, mode }: { s: Snapshot; incident?: Incident; mode: Mode }) {
  if (mode === 'finished' || mode === 'cancelled' || mode === 'failed' || mode === 'connecting' || mode === 'idle') return null
  const badges = headerBadges(s, incident)
  return (
    <>
      {badges.map((b) =>
        b.live ? (
          <span key={b.text} className="badge ok">
            <span className="dot pulse-live" aria-hidden="true" />
            {b.text}
          </span>
        ) : (
          <Badge key={b.text} tone={b.tone} dot={!b.pulse} pulse={b.pulse}>
            {b.text}
          </Badge>
        ),
      )}
    </>
  )
}

/** The top bar's events line and actions for each view. */
function HeaderActions({
  s,
  mode,
  incident,
  can,
  actions,
  onPause,
  onCancel,
  onBack,
}: {
  s: Snapshot
  mode: Mode
  incident?: Incident
  can: LivePermissions
  actions: LiveActions
  onPause: () => void
  onCancel: () => void
  onBack: () => void
}) {
  const t = useT()
  if (mode === 'decision') {
    return (
      <>
        <span className="k lv-caption">{whoLine(s)}</span>
        <Button onClick={onBack}>{t(m.backToLive)}</Button>
      </>
    )
  }
  if (mode !== 'live' && mode !== 'connecting') return null
  const paused = s.state === 'paused' && !incident
  return (
    <>
      <span className="k lv-caption">{whoLine(s)}</span>
      {s.options.canPause ? (
        <Button onClick={onPause} disabledReason={can.pause} loading={actions.pending}>
          {t(m.pause)}
        </Button>
      ) : null}
      {s.options.canCancel && !paused ? (
        <Button variant="danger" onClick={onCancel} disabledReason={can.cancel}>
          {t(m.cancelSend)}
        </Button>
      ) : null}
    </>
  )
}

/** 403, 404 or 500 while loading the monitor. */
function LiveError({ error, retry, campaignId }: { error: unknown; retry: () => void; campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = error instanceof ApiError ? error.status : 0
  const back = (
    <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
      {t(m.backToCampaigns)}
    </ButtonLink>
  )
  let body: ReactNode
  if (error instanceof ApiError && status === 403 && error.problem.permission) {
    body = <ForbiddenState permission={error.problem.permission} title={t(m.forbiddenTitle)} readOnly={back} />
  } else if (status === 404) {
    body = (
      <NotFoundState path={`/campaigns/${campaignId}/live`} reference={problemReference(error)} primary={back}>
        {t(m.notFoundBody)}
      </NotFoundState>
    )
  } else {
    body = <ServerErrorState reference={problemReference(error)} savedNote={t(m.serverErrorNote)} onRetry={retry} back={back} />
  }
  return (
    <Page title={status === 404 ? t(shell.stateNotFoundTitle) : status === 403 ? t(shell.stateNoAccessTitle) : t(m.title)} breadcrumb={crumbs(slug, t(m.title))}>
      {body}
    </Page>
  )
}
