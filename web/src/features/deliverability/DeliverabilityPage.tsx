import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, clock, count, monthDay } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { DeliverabilityTabs } from '@/features/placement/shared'
import { AssetGrid, SkeletonGrid, StaleGrid } from './AssetCards'
import { BlocklistCard } from './BlocklistCard'
import { DmarcCard, DmarcEmptyCard } from './DmarcCard'
import { IncidentTimeline } from './IncidentTimeline'
import { m } from './messages'
import { ProviderHealthCard } from './ProviderHealth'
import { PostmasterCard, SndsCard } from './SignalCards'
import { eta, firstRun, type Overview } from './words'

const overviewPath = '/api/v1/workspaces/{workspace}/deliverability' as const

/** 7.8 Deliverability: HostRepute connected or not, with loading and error states. */
export function DeliverabilityPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    overviewPath,
    { params: { path: { workspace: slug } } },
    { refetchInterval: (query) => (query.state.data?.checks.running ? 1000 : false) },
  )
  const o = q.data
  useFinishedToast(o)
  return (
    <Page
      title={t(m.title)}
      badges={
        <>
          {/* Health · Placement tests (X14). */}
          <DeliverabilityTabs />
          {o ? <HeaderStatus o={o} /> : null}
        </>
      }
      actions={<Actions o={o} />}
    >
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.loadFailedBody)}
        </ErrorPanel>
      ) : !o ? (
        <section className="flex flex-col gap-3.5" aria-label={t(m.loadingLabel)}>
          <span className="vh" role="status">
            {t(m.loadingStatus)}
          </span>
          <div className="flex items-center gap-3 rounded-[10px] border border-line bg-white px-4 py-3">
            <Skeleton w={16} h={16} r={999} />
            <Skeleton w={260} h={14} />
            <Skeleton w={200} h={12} />
          </div>
          <SkeletonGrid />
        </section>
      ) : noSetup(o) ? (
        <NoSetup />
      ) : (
        <Body o={o} />
      )}
    </Page>
  )
}

/** Toast when a check run that was running finishes. */
function useFinishedToast(o?: Overview): void {
  const last = useRef<string | null>(null)
  useEffect(() => {
    const running = o?.checks.running?.id ?? null
    if (last.current && !running && o) {
      const listed = o.blocklists.listed
      const sources = count(o.blocklists.sources)
      toast({
        kind: 'success',
        title: tt(m.checksFinished),
        detail: listed ? tt(m.checksFinishedListed, { listed, sources }) : tt(m.checksFinishedClean, { sources }),
      })
    }
    last.current = running
  }, [o])
}

function HeaderStatus({ o }: { o: Overview }) {
  const t = useT()
  const hr = o.hostRepute
  const c = o.checks
  let badge
  if (hr.status === 'connected' && hr.unreachable) {
    badge = (
      <Badge tone="bad" dot>
        {t(m.hrUnreachable)}
      </Badge>
    )
  } else if (hr.status === 'connected') {
    badge = (
      <Badge tone="ok" dot>
        {t(m.hrConnected, { interval: c.interval })}
      </Badge>
    )
  } else if (hr.status === 'not_connected') {
    badge = (
      <Badge tone="neutral" dot>
        {t(m.hrNotConnected)}
      </Badge>
    )
  } else {
    badge = (
      <Badge tone="warn" dot>
        {t(m.hrNeedsReconnect)}
      </Badge>
    )
  }
  const when =
    c.via === 'hostrepute'
      ? c.lastAt
        ? c.nextAt
          ? t(m.lastFullCheckNext, { time: clock(c.lastAt), next: clock(c.nextAt) })
          : t(m.lastFullCheck, { time: clock(c.lastAt) })
        : t(m.firstCheckRunning)
      : c.lastAt
        ? t(m.lastCheckDaily, { time: clock(c.lastAt) })
        : t(m.dailyChecks)
  return (
    <>
      {badge}
      <span className="k">{when}</span>
    </>
  )
}

function Actions({ o }: { o?: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/deliverability/checks', {
    onSuccess: () => qc.invalidateQueries({ queryKey: ['get', overviewPath] }),
    onError: (e) => toast({ kind: 'error', title: tt(m.startFailed), detail: errorMessage(e) }),
  })
  const [exporting, setExporting] = useState(false)
  async function exportReport() {
    setExporting(true)
    try {
      const { data, response } = await http.GET('/api/v1/workspaces/{workspace}/deliverability/export', { params: { path: { workspace: slug } }, parseAs: 'blob' })
      const name = /filename="([^"]+)"/.exec(response.headers.get('Content-Disposition') ?? '')?.[1] ?? 'deliverability.csv'
      const url = URL.createObjectURL(data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = name
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
    } finally {
      setExporting(false)
    }
  }
  const running = Boolean(o?.checks.running)
  const label = o && o.hostRepute.status === 'connected' ? t(m.runAll) : t(m.runChecks)
  const empty = o && noSetup(o) ? t(m.nothingToCheck) : undefined
  return (
    <>
      <Button onClick={exportReport} loading={exporting} disabled={!o} disabledReason={empty}>
        {t(m.exportReport)}
      </Button>
      <Button
        variant="primary"
        onClick={() => run.mutate({ params: { path: { workspace: slug } } })}
        loading={run.isPending || running}
        disabled={!o}
        disabledReason={empty}
      >
        {label}
      </Button>
    </>
  )
}

/** A workspace without a sending domain or provider has nothing to check yet. */
function noSetup(o: Overview): boolean {
  return o.assets.length === 0 && o.providers.length === 0
}

/** First run of a workspace without a sending domain or provider. */
function NoSetup() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <EmptyState
      title={t(m.emptyTitle)}
      action={
        <ButtonLink variant="primary" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
          {t(m.connectProvider)}
        </ButtonLink>
      }
    >
      {t(m.emptyBody)}
    </EmptyState>
  )
}

/** "Checking news.acme.io and 203.0.113.25 · 61 of 84 sources answered · about 6 s left". */
function Progress({ o }: { o: Overview }) {
  const t = useT()
  const now = useNow(1000)
  const r = o.checks.running!
  const names = r.assets.length ? r.assets : o.assets.map((a) => a.asset)
  return (
    <div className="flex items-center gap-3 rounded-[10px] border border-line bg-white px-4 py-3 max-md:flex-wrap" role="status" aria-live="polite">
      <span className="spin" aria-hidden="true" />
      <span className="font-semibold">{t(m.checking, { assets: andList(names) })}</span>
      <span className="k">{t(m.checkingProgress, { answered: count(r.answered), total: count(r.total), eta: eta(r, now.getTime()) })}</span>
    </div>
  )
}

function Body({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const hr = o.hostRepute
  const connected = hr.status === 'connected'
  const stale = (connected && Boolean(hr.unreachable)) || hr.status === 'revoked' || hr.status === 'expired'
  const canConnect = useCan('integrations:hostrepute')
  const delistReason = usePermissionReason('integrations:hostrepute')
  const [delisting, setDelisting] = useState<string>()
  async function delist(asset: string) {
    setDelisting(asset)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/deliverability/delistings', { params: { path: { workspace: slug } }, body: { asset } })
      await qc.invalidateQueries({ queryKey: ['get', overviewPath] })
      const r = data?.requests[0]
      if (r?.already) {
        toast({ kind: 'info', title: t(m.delistAlready, { date: monthDay(r.requestedAt) }), detail: t(m.delistAlreadyDetail, { source: r.source }) })
      } else if (r) {
        toast({ kind: 'success', title: t(m.delistRequested, { asset }), detail: t(m.delistRequestedDetail, { sources: andList(data!.requests.map((x) => x.source)) }) })
      }
    } catch (e) {
      toast({ kind: 'error', title: t(m.delistFailed), detail: e instanceof ApiError ? e.message : errorMessage(e) })
    } finally {
      setDelisting(undefined)
    }
  }

  if (firstRun(o)) {
    return (
      <section className="flex flex-col gap-3.5" aria-label={t(m.loadingLabel)}>
        <Progress o={o} />
        <SkeletonGrid />
      </section>
    )
  }
  return (
    <>
      {o.checks.running ? <Progress o={o} /> : null}
      {hr.status === 'revoked' || hr.status === 'expired' ? <ReconnectBanner rejectedAt={hr.tokenRejectedAt} /> : null}
      {connected && hr.unreachable ? <UnreachableAlert o={o} /> : null}
      {hr.status === 'not_connected' ? <ConnectInvite canConnect={canConnect} /> : null}
      {stale ? (
        <section className="flex flex-col gap-3.5" aria-label={t(m.cachedLabel)}>
          <StaleGrid o={o} />
          <span className="k">{t(m.staleNote)}</span>
        </section>
      ) : (
        <AssetGrid o={o} onDelist={delist} delistReason={delistReason} delisting={delisting} />
      )}
      <ProviderHealthCard providers={o.providers} connected={connected} />
      <div className="flex flex-wrap items-start gap-5">
        <BlocklistCard b={o.blocklists} connected={connected} canConnect={canConnect} />
        <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-5">
          {o.postmaster ? <PostmasterCard p={o.postmaster} /> : <PanelPlaceholder title={'Google Postmaster Tools'} text={t(m.postmasterPlaceholder)} />}
          {o.snds ? <SndsCard s={o.snds} /> : o.assets.some((a) => a.type === 'ip') ? <PanelPlaceholder title={'Microsoft SNDS'} text={t(m.sndsPlaceholder)} /> : null}
          {connected ? null : o.dmarc.sources.length ? <DmarcCard d={o.dmarc} className="card min-w-0" /> : <DmarcEmptyCard connected={false} />}
        </div>
      </div>
      {connected ? (
        <div className="flex flex-wrap items-start gap-5">
          {o.dmarc.sources.length ? <DmarcCard d={o.dmarc} /> : <div className="min-w-0 flex-[1_1_440px]"><DmarcEmptyCard connected rua={hr.dmarcRua} /></div>}
          <IncidentTimeline t={o.incidents} connected className="flex-[1_1_440px]" />
        </div>
      ) : (
        <IncidentTimeline t={o.incidents} connected={false} />
      )}
    </>
  )
}

/** A Postmaster or SNDS panel without data: what it shows, and how to bring it in. */
function PanelPlaceholder({ title, text }: { title: string; text: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-4.5 py-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2">{title}</h2>
        <Badge tone="neutral">{t(m.noDataYet)}</Badge>
      </div>
      <span className="k">{text}</span>
      <div className="flex">
        <ButtonLink size="sm" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
          {t(m.connectHostRepute)}
        </ButtonLink>
      </div>
    </section>
  )
}

/** Not connected: everything works, HostRepute adds more (DeliverabilityNoHostRepute). */
function ConnectInvite({ canConnect }: { canConnect: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('integrations:hostrepute')
  return (
    <section className="card flex flex-wrap items-center gap-5 border-accent-line bg-accent-tint px-5.5 py-5" aria-labelledby="h-invite">
      <svg width="44" height="44" viewBox="0 0 44 44" aria-hidden="true">
        <rect width="44" height="44" rx="11" fill="#0F766E" />
        <path d="M22 9 33 13.5v8c0 6-4.6 10.6-11 13-6.4-2.4-11-7-11-13v-8z" fill="none" stroke="#fff" strokeWidth="2" strokeLinejoin="round" />
        <path d="m16.5 22 3.6 3.6 7.4-7.6" fill="none" stroke="#fff" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
      <div className="flex flex-[1_1_360px] flex-col gap-1">
        <h2 className="m-0 text-[16px] font-bold" id="h-invite">
          {t(m.inviteTitle)}
        </h2>
        <span className="text-[13px] text-ink-nav">{t(m.inviteBody)}</span>
      </div>
      <div className="flex flex-wrap gap-2.5">
        <ButtonAnchor href="https://hostrepute.com">{t(m.whatHostReputeChecks)}</ButtonAnchor>
        {canConnect ? (
          <ButtonLink variant="primary" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
            {t(m.connectHostRepute)}
          </ButtonLink>
        ) : (
          <Button variant="primary" disabledReason={reason}>
            {t(m.connectHostRepute)}
          </Button>
        )}
      </div>
    </section>
  )
}

/** Token revoked or expired: the Deliverability banner (HostReputeStates). */
function ReconnectBanner({ rejectedAt }: { rejectedAt?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('integrations:hostrepute')
  return (
    <section className="note w items-center" role="status" aria-label={t(m.reconnectLabel)}>
      <Badge tone="warn">{t(m.reconnectBadge)}</Badge>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <span className="min-w-0 flex-[1_1_420px]">
          {rejectedAt ? t(m.reconnectBodyAt, { date: monthDay(rejectedAt), time: clock(rejectedAt) }) : t(m.reconnectBody)}
        </span>
        {reason ? null : (
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
            {t(m.reconnectWith)}
          </ButtonLink>
        )}
      </div>
    </section>
  )
}

/** HostRepute unreachable: cached results are shown (DeliverabilityStates › error). */
function UnreachableAlert({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/deliverability/checks', {
    onSuccess: () => qc.invalidateQueries({ queryKey: ['get', overviewPath] }),
    onError: (e) => toast({ kind: 'error', title: tt(m.startFailed), detail: errorMessage(e) }),
  })
  const hr = o.hostRepute
  const last = o.checks.lastAt ?? hr.lastSyncAt
  // Google Postmaster and Microsoft SNDS are cached when they come through
  // HostRepute, current when they're fetched directly.
  const yes = (v: boolean) => (v ? 'yes' : 'no')
  const stale = t(m.unreachableStale, {
    pm: yes(Boolean(o.postmaster) && o.postmaster?.via !== 'direct'),
    snds: yes(Boolean(o.snds) && o.snds?.via !== 'direct'),
    hasTime: yes(Boolean(last)),
    time: last ? clock(last) : '',
  })
  const current = t(m.unreachableCurrent, { pm: yes(o.postmaster?.via === 'direct'), snds: yes(o.snds?.via === 'direct') })
  const fallback = o.assets.find((a) => a.fallback)?.fallback
  const fallbackListed = o.assets.reduce((n, a) => n + (a.fallback?.listed ?? 0), 0)
  const fallbackText = fallback?.checkedAt
    ? fallbackListed
      ? t(m.unreachableFallbackFound, { sources: fallback.sources, listed: fallbackListed })
      : t(m.unreachableFallbackNothing, { sources: fallback.sources })
    : t(m.unreachableFallbackRuns)
  return (
    <div role="alert" className="flex items-start gap-3.5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-4 py-3.5 max-md:flex-wrap">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="#B91C1C" strokeWidth="1.8" aria-hidden="true">
        <circle cx="10" cy="10" r="8" />
        <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
      </svg>
      <div className="flex flex-[1_1_300px] flex-col gap-1">
        <span className="font-semibold text-bad-fg">{hr.unreachable?.message}</span>
        <span className="text-[13px] text-[#7F1D1D]">{`${stale} ${current} ${fallbackText}`}</span>
        <div className="mt-1.5 flex flex-wrap gap-2.5">
          <Button size="sm" variant="primary" onClick={() => run.mutate({ params: { path: { workspace: slug } } })} loading={run.isPending}>
            {t(common.retry)}
          </Button>
          <ButtonAnchor size="sm" href="https://status.hostrepute.com">
            {t(m.hrStatus)}
          </ButtonAnchor>
          <Button size="sm" onClick={() => run.mutate({ params: { path: { workspace: slug }, query: { builtin: true } } })}>
            {t(m.useBuiltin)}
          </Button>
        </div>
      </div>
    </div>
  )
}
