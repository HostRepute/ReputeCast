import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, clock, shortDate, weekdayDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AuthMarks, CopyDetailPanel, Matrix } from './Matrix'
import { m } from './messages'
import {
  arrive,
  base,
  dayTime,
  elapsed,
  FOLDER,
  LoadError,
  PBar,
  Pv,
  Sw,
  useRefreshPlacement,
  useTest,
  type Guidance,
  type ProviderResult,
  type TestDetail,
} from './shared'

const HOSTREPUTE = 'https://app.hostrepute.com'

/** A placement test's result: the providers, the matrix, a copy's detail and the guidance. */
export function ResultPage({ testId, copy }: { testId: string; copy?: string }) {
  const t = useT()
  const q = useTest(testId)
  const { slug } = useWorkspace()
  const crumbs = (name?: string) => [
    { label: t(m.deliverability), to: '/w/$workspace/deliverability' as const, params: { workspace: slug } },
    { label: t(m.backToTests), to: '/w/$workspace/deliverability/placement' as const, params: { workspace: slug } },
    ...(name ? [{ label: name }] : []),
  ]
  if (q.isPending) {
    return (
      <Page title={t(m.placementTests)} breadcrumb={crumbs()}>
        <Loading label={t(m.loading)} className="card flex flex-col gap-3.5 p-[18px]">
          <span className="k" aria-hidden="true">
            {t(m.loading)}
          </span>
          <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(200px,100%),1fr))' }}>
            {[0, 1].map((i) => (
              <div key={i} className="card flex flex-col gap-2 p-3.5">
                <Skeleton w={110} h={12} />
                <Skeleton w="100%" h={20} r={6} />
                <Skeleton w={140} h={10} />
              </div>
            ))}
          </div>
        </Loading>
      </Page>
    )
  }
  if (q.isError) {
    const missing = q.error instanceof ApiError && q.error.status === 404
    return (
      <Page title={t(m.placementTests)} breadcrumb={crumbs()}>
        {missing ? (
          <EmptyState
            title={t(m.notFound)}
            action={
              <ButtonLink to="/w/$workspace/deliverability/placement" params={{ workspace: slug }}>
                {t(m.backToTests)}
              </ButtonLink>
            }
          />
        ) : (
          <LoadError title={t(m.loadTestError)} error={q.error} body={(status) => t(m.loadTestBody, { status })} onRetry={() => void q.refetch()} />
        )}
      </Page>
    )
  }
  return <Result d={q.data} copy={copy} crumbs={crumbs(q.data.name)} />
}

const statusBadge = (d: TestDetail, t: ReturnType<typeof useT>) =>
  d.status === 'finished' ? (
    <Badge tone="ok" dot>
      {t(m.statusFinished)}
    </Badge>
  ) : d.status === 'stopped' ? (
    <Badge tone="neutral">{t(m.statusStopped)}</Badge>
  ) : d.status === 'sending' ? (
    <Badge tone="info" dot>
      {t(m.statusSending)}
    </Badge>
  ) : (
    <Badge tone="neutral" dot>
      {t(m.statusWaiting)}
    </Badge>
  )

function Result({ d, copy, crumbs }: { d: TestDetail; copy?: string; crumbs: { label: string; to?: '/w/$workspace/deliverability' | '/w/$workspace/deliverability/placement'; params?: { workspace: string } }[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshPlacement()
  const live = d.status === 'sending' || d.status === 'waiting'
  const now = useNow(live ? 1000 : 60_000)
  const [confirm, setConfirm] = useState<'delete' | 'stop'>()
  const [busy, setBusy] = useState(false)
  // The copy shown in the detail: the one picked, else the first in spam.
  const firstSpam = d.copies.find((c) => c.status === 'found' && c.folder === 'spam')
  const [closed, setClosed] = useState(false)
  const selected = copy ?? (closed ? undefined : firstSpam?.id)
  const select = (id?: string) => {
    setClosed(!id)
    void navigate({ to: '/w/$workspace/deliverability/placement/$testId', params: { workspace: slug, testId: d.id }, search: id ? { copy: id } : {}, replace: true, resetScroll: false })
  }
  const runReason = d.canRun ? undefined : d.runHelpers.length ? t(m.cantRun, { helpers: andList(d.runHelpers) }) : t(m.cantRunNobody)
  const copiesIn = !d.copiesDeletedAt && d.counts.found > 0

  const act = async (what: 'stop' | 'delete-copies' | 'run-again') => {
    setBusy(true)
    const path = what === 'stop' ? `${base}/tests/{test}/stop` : what === 'delete-copies' ? `${base}/tests/{test}/delete-copies` : `${base}/tests/{test}/run-again`
    const res = await http.POST(path as `${typeof base}/tests/{test}/stop`, { params: { path: { workspace: slug, test: d.id } } })
    setBusy(false)
    if (!res.data) throw new Error(errorMessage(res.error))
    void refresh()
    if (what === 'run-again') {
      toast({ kind: 'success', title: t(m.testStarted), detail: t(m.testStartedDetail, { copies: res.data.counts.copies, seeds: res.data.seeds.length }) })
      void navigate({ to: '/w/$workspace/deliverability/placement/$testId', params: { workspace: slug, testId: res.data.id } })
    } else toast({ kind: what === 'delete-copies' ? 'deleted' : 'success', title: t(what === 'delete-copies' ? m.copiesDeleted : m.stopped) })
  }
  const runAgain = () => act('run-again').catch((e: unknown) => toast({ kind: 'error', title: errorMessage(e) }))

  const actions = d.canRun ? (
    <>
      {live ? (
        <Button onClick={() => setConfirm('stop')}>{t(m.stopWaiting)}</Button>
      ) : null}
      <Button icon="trash" onClick={() => setConfirm('delete')} disabledReason={copiesIn && !live ? undefined : live ? t(m.deleteWhileWaiting) : t(m.nothingToDelete)}>
        {t(m.deleteCopiesNow)}
      </Button>
      {live ? null : (
        <Button variant="primary" loading={busy} onClick={() => void runAgain()}>
          {t(m.runAgain)}
        </Button>
      )}
    </>
  ) : live ? null : (
    <Button variant="primary" disabledReason={runReason}>
      {t(m.runAgain)}
    </Button>
  )

  const name = `${d.contentName}, ${andList(d.providers.map((p) => p.name))}`
  const guidanceFor = new Set(d.guidance.map((g) => g.providerId))
  const missing = d.copies.filter((c) => c.status === 'sent' || c.status === 'queued').length
  return (
    <Page
      title={t(m.resultH1, { name })}
      documentTitle={t(m.resultDocument, { name: d.name })}
      breadcrumb={crumbs}
      badges={
        <>
          {statusBadge(d, t)}
          {live ? (
            <span className="k">
              {t(m.waitingLine, { found: d.counts.found, copies: d.counts.copies, elapsed: elapsed(d.startedAt, now.getTime()), time: d.waitUntil ? clock(d.waitUntil) : '' })}
            </span>
          ) : null}
        </>
      }
      actions={actions}
    >
      {d.status === 'stopped' ? <StoppedNote d={d} /> : null}
      <About d={d} />
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(460px,100%),1fr))' }}>
        {d.providers.map((p) => (
          <ProviderCard key={p.id} p={p} live={live} warn={guidanceFor.has(p.id)} />
        ))}
      </div>
      <div className="flex flex-wrap items-start gap-4">
        <section className="card min-w-0 flex-[999_1_640px] overflow-hidden">
          <div className="sec">
            <div className="flex min-w-0 flex-col gap-0.5">
              <h2 className="h2">{t(m.copiesTitle, { count: d.counts.copies })}</h2>
              <span className="k">
                {live ? `${t(m.copiesWaitingLead)}${d.rounds > 1 && d.lastSentAt ? ` ${t(m.copiesSecondRound, { time: clock(d.lastSentAt) })}` : ''}` : t(m.copiesLead)}
              </span>
            </div>
            {live ? (
              <span className="pl-live">
                <span className="dot" aria-hidden="true" />
                {t(m.live)}
              </span>
            ) : null}
          </div>
          <Matrix test={d} selected={selected} onSelect={(id) => select(id === selected ? undefined : id)} live={live} />
        </section>
        {selected ? <CopyDetailPanel test={d} copyId={selected} onClose={() => select(undefined)} /> : null}
      </div>
      {d.guidance.map((g) => (
        <GuidanceCard key={g.providerId} g={g} d={d} />
      ))}
      <div className="note n" style={{ flexWrap: 'nowrap' }} role="note">
        <InfoIcon />
        <div className="min-w-0 flex-[1_1_320px]">{t(m.sampleNote, { count: d.seeds.length })}</div>
      </div>
      <ConfirmDialog
        open={confirm === 'delete'}
        onOpenChange={(o) => !o && setConfirm(undefined)}
        tone="bad"
        badge={t(m.deleteCopiesNow)}
        title={t(m.deleteCopiesTitle)}
        description={t(m.deleteCopiesBody, { count: d.counts.found })}
        confirmLabel={t(m.deleteCopiesConfirm)}
        busyLabel={t(m.deleting)}
        onConfirm={() => act('delete-copies')}
      />
      <ConfirmDialog
        open={confirm === 'stop'}
        onOpenChange={(o) => !o && setConfirm(undefined)}
        tone="warn"
        badge={t(m.stopWaiting)}
        title={t(m.stopTitle)}
        description={t(m.stopBody, { missing })}
        confirmLabel={t(m.stopConfirm)}
        busyLabel={t(m.stopping)}
        confirmVariant="primary"
        onConfirm={() => act('stop')}
      />
    </Page>
  )
}

function InfoIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#4B5563" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="mt-px flex-none">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </svg>
  )
}

function StoppedNote({ d }: { d: TestDetail }) {
  const t = useT()
  const gone = d.copies.filter((c) => c.status === 'not_arrived' && c.reason === 'stopped')
  const minutes = d.stoppedAt ? Math.max(0, Math.round((new Date(d.stoppedAt).getTime() - new Date(d.startedAt).getTime()) / 60000)) : 0
  return (
    <div className="note n" style={{ flexWrap: 'nowrap' }} role="note">
      <InfoIcon />
      <div className="min-w-0 flex-[1_1_320px]">
        {t(m.stoppedNote, { name: d.stoppedBy ?? '', time: d.stoppedAt ? clock(d.stoppedAt) : '', minutes, count: gone.length, first: gone[0]?.seedProviderName ?? '' })}
      </div>
    </div>
  )
}

function Fact({ label, value, sub }: { label: string; value: ReactNode; sub?: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="k">{label}</span>
      <span className="text-[13px] font-semibold text-ink">{value}</span>
      {sub ? <span className="k text-ink-2!">{sub}</span> : null}
    </div>
  )
}

function About({ d }: { d: TestDetail }) {
  const t = useT()
  const names = d.providers.map((p) => p.name)
  const contentValue =
    d.contentKind === 'transactional'
      ? t(m.contentTransactional, { name: d.contentName, key: d.contentKey ?? '', version: d.contentVersion ?? 0 })
      : d.contentKind === 'template'
        ? t(m.contentTemplate, { name: d.contentName })
        : d.contentKind === 'digest'
          ? t(m.contentDigest, { name: d.contentName })
          : t(m.contentCampaign, { name: d.contentName, note: d.contentNote, review: d.origin === 'review' ? 'yes' : 'no', sent: d.contentAt ? shortDate(d.contentAt) : '' })
  const contentSub = [
    d.variant ? t(m.aboutVariant, { variant: d.variant }) : '',
    d.language ? languageName(d.language) : '',
    d.personalization ? t(m.aboutTestData, { name: d.personalization }) : d.fallbackGreeting ? t(m.aboutFallbacks, { greeting: d.fallbackGreeting }) : t(m.aboutFallbacksPlain),
  ]
    .filter(Boolean)
    .join(' · ')
  const end =
    d.status === 'finished' ? (
      <Fact label={t(m.aboutFinished)} value={d.finishedAt ? clock(d.finishedAt) : '–'} sub={d.finishReason === 'all_found' ? t(m.aboutFinishedAll) : t(m.aboutFinishedTimeout)} />
    ) : d.status === 'stopped' ? (
      <Fact
        label={t(m.aboutStopped)}
        value={d.stoppedAt ? clock(d.stoppedAt) : '–'}
        sub={t(m.aboutStoppedBy, { name: d.stoppedBy ?? '', minutes: d.stoppedAt ? Math.round((new Date(d.stoppedAt).getTime() - new Date(d.startedAt).getTime()) / 60000) : 0 })}
      />
    ) : (
      <Fact label={t(m.aboutWaiting)} value={d.waitUntil ? t(m.aboutKeepsLooking, { time: clock(d.waitUntil) }) : '–'} sub={t(m.inboxSoFar, { found: d.counts.found, copies: d.counts.copies })} />
    )
  const rule = d.deleteAfter === 'never' ? m.copiesRuleNever : d.deleteAfter === 'finished' ? m.copiesRuleFinished : m.copiesRule24h
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(m.aboutTitle)}</h2>
      </div>
      <div className="pad pl-about grid gap-x-6 gap-y-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(300px,100%),1fr))' }}>
        <Fact label={t(m.aboutStarted)} value={`${weekdayDate(d.startedAt)}, ${clock(d.startedAt)}`} sub={t(m.aboutStartedBy, { origin: d.origin, name: d.startedBy })} />
        {end}
        <Fact label={t(m.aboutContent)} value={contentValue} sub={contentSub} />
        <Fact label={t(m.aboutFrom)} value={`${d.fromName} <${d.fromEmail}>`} sub={t(m.streamName, { stream: d.stream })} />
        <Fact
          label={t(m.aboutThrough)}
          value={andList(names)}
          sub={
            d.through === 'compare'
              ? t(m.throughCompare, { count: d.counts.copies, gap: d.gapMinutes })
              : d.through === 'route'
                ? t(m.throughRoute, { count: d.counts.copies })
                : t(m.throughOne, { count: d.counts.copies })
          }
        />
        {d.copiesDeletedAt ? (
          <Fact label={t(m.aboutCopies)} value={t(m.copiesDeletedAt, { date: dayTime(d.copiesDeletedAt) })} sub={d.copiesDeletedBy ? t(m.copiesDeletedBy, { name: d.copiesDeletedBy }) : t(rule)} />
        ) : d.deleteAfter === 'never' ? (
          <Fact label={t(m.aboutCopies)} value={t(m.copiesKept)} sub={t(rule)} />
        ) : (
          <Fact label={t(m.aboutCopies)} value={d.deleteAt ? t(m.copiesUntil, { date: dayTime(d.deleteAt) }) : t(m.copiesUntilFinished)} sub={t(rule)} />
        )}
      </div>
    </section>
  )
}

const names = new Map<string, Intl.DisplayNames>()
function languageName(code: string): string {
  try {
    let n = names.get('en')
    if (!n) {
      n = new Intl.DisplayNames(['en'], { type: 'language' })
      names.set('en', n)
    }
    return n.of(code) ?? code
  } catch {
    return code
  }
}

const providerTone = { healthy: 'ok', warming_up: 'info', degraded: 'warn', down: 'bad', sandbox: 'warn', paused: 'neutral' } as const

function ProviderStatus({ p }: { p: ProviderResult }) {
  const t = useT()
  const s = p.status as keyof typeof providerTone
  const label =
    s === 'warming_up'
      ? t(m.providerStatusWarming, { day: p.warmDay ?? 0, days: p.warmDays ?? 0 })
      : t({ healthy: m.providerStatusHealthy, degraded: m.providerStatusDegraded, down: m.providerStatusDown, paused: m.providerStatusPaused, sandbox: m.providerStatusSandbox }[s] ?? m.providerStatusHealthy)
  return (
    <Badge tone={providerTone[s] ?? 'neutral'} dot>
      {label}
    </Badge>
  )
}

function ProviderCard({ p, live, warn }: { p: ProviderResult; live: boolean; warn: boolean }) {
  const t = useT()
  const c = p.counts
  const tabName = c.tabs.length === 1 ? t(m.legendTabNamed, { tab: c.tabs[0]! }) : t(m.legendTab)
  const legend: [string, string, number][] = [
    [FOLDER.inbox, t(m.legendInbox), c.inbox],
    [FOLDER.tab, tabName, c.tab],
    [FOLDER.spam, t(m.legendSpam), c.spam],
    [FOLDER.notArrived, t(m.legendNotArrived), c.notArrived],
  ]
  if (live) legend.push(['var(--line-2)', t(m.legendWaiting), c.waiting])
  const slow = p.slowest
  const timing = [
    p.fastest && (!slow || slow.folder === 'inbox') ? t(m.fastest, { time: arrive(p.fastest.ms), provider: p.fastest.seedProviderName }) : '',
    slow ? (slow.folder === 'inbox' ? t(m.slowest, { time: arrive(slow.ms), provider: slow.seedProviderName }) : t(m.slowestIn, { time: arrive(slow.ms), provider: slow.seedProviderName, folder: slow.folderName })) : '',
  ].filter(Boolean)
  const a = p.auth
  const failed = [a.spf.fail ? 'SPF' : '', a.dkim.fail ? 'DKIM' : '', a.dmarc.fail ? 'DMARC' : ''].filter(Boolean)
  const res = (x: { pass: number; fail: number }) => (x.fail ? 'fail' : x.pass ? 'pass' : undefined)
  return (
    <section className="card min-w-0" style={{ borderColor: warn ? '#FCD34D' : undefined }}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-[3px]">
          <h2 className="h2 flex items-center gap-2">
            <Pv adapter={p.adapter}>{p.name}</Pv>
          </h2>
          <span className="k">{[p.detail, p.ip].filter(Boolean).join(' · ')}</span>
        </div>
        <ProviderStatus p={p} />
      </div>
      <div className="pad flex flex-col gap-3">
        <span className="text-[22px] font-bold tracking-[-0.01em]">
          {live ? t(m.inboxSoFar, { found: c.found, copies: c.copies }) : t(m.inboxHeadline, { inbox: c.inbox, copies: c.copies })}
        </span>
        <PBar counts={c} lg waiting={live} />
        <div className="pl-plg">
          {legend.map(([color, label, n]) => (
            <span key={label} className={n ? undefined : 'z'}>
              <Sw color={color} />
              {label}
              <b className="font-semibold text-ink">{n}</b>
            </span>
          ))}
        </div>
        <div className="pl-chk2 mt-0.5">
          {p.medianMs != null ? (
            <div>
              <span className="flex w-full min-w-0 flex-wrap justify-between gap-x-3.5 gap-y-1.5">
                <span className="k flex-none">{t(m.medianLabel)}</span>
                <span className="pl-sv">
                  {live ? (
                    <span className="k">{t(m.medianSoFar, { inbox: c.inbox, median: arrive(p.medianMs) })}</span>
                  ) : (
                    <>
                      <span className="pl-tm">{arrive(p.medianMs)}</span>
                      {timing.length ? <span className="k"> · {timing.join(' · ')}</span> : null}
                    </>
                  )}
                </span>
              </span>
            </div>
          ) : null}
          {live ? null : (
            <div>
              <span className="flex w-full min-w-0 flex-wrap justify-between gap-x-3.5 gap-y-1.5">
                <span className="k flex-none">{t(m.authLabel)}</span>
                <span className="pl-sv">
                  {a.checked ? (
                    <>
                      <AuthMarks spf={res(a.spf)} dkim={res(a.dkim)} dmarc={res(a.dmarc)} />{' '}
                      <span className="k">
                        {failed.length
                          ? t(m.authSome, { failed: Math.max(a.spf.fail, a.dkim.fail, a.dmarc.fail) })
                          : a.checked < c.copies
                            ? t(m.authArrived, { count: a.checked })
                            : t(m.authAll, { count: a.checked })}
                      </span>
                    </>
                  ) : (
                    <span className="k">{t(m.authNone)}</span>
                  )}
                </span>
              </span>
            </div>
          )}
          {!live && a.dkimDomain ? (
            <div>
              <span className="flex w-full min-w-0 flex-wrap justify-between gap-x-3.5 gap-y-1.5">
                <span className="k flex-none">{t(m.signedLabel)}</span>
                <span className="pl-sv">
                  <span className="k">
                    <T msg={m.signedLine} values={{ dkim: a.dkimDomain, selector: a.dkimSelector ?? '', spf: a.spfDomain ?? '', mono: (ch) => <span className="mono text-[12px] text-ink">{ch}</span> }} />
                  </span>
                </span>
              </span>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  )
}

function LookIcon({ kind }: { kind: 'ok' | 'lead' | 'bad' }) {
  if (kind === 'ok')
    return (
      <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="m3 8 3.5 3.5L13 5" />
      </svg>
    )
  if (kind === 'bad')
    return (
      <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#B91C1C" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="m4 4 8 8M12 4l-8 8" />
      </svg>
    )
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 8h10M9 4l4 4-4 4" />
    </svg>
  )
}

function GuidanceCard({ g, d }: { g: Guidance; d: TestDetail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const failed = [g.auth.spf.fail ? 'SPF' : '', g.auth.dkim.fail ? 'DKIM' : '', g.auth.dmarc.fail ? 'DMARC' : ''].filter(Boolean)
  const failCount = Math.max(g.auth.spf.fail, g.auth.dkim.fail, g.auth.dmarc.fail)
  const warming = g.status === 'warming_up' && g.warmDay
  const r = g.readiness
  const cause =
    g.cause === 'readiness'
      ? t(m.causeReadiness, { evidence: r?.evidence.join(' ') ?? '' })
      : g.cause === 'auth'
        ? t(m.causeAuth, { failed: andList(failed), count: failCount, provider: g.providerName })
        : g.cause === 'warmup'
          ? t(g.other ? m.causeWarmup : m.causeWarmupAlone, { provider: g.providerName, day: g.warmDay ?? 0, days: g.warmDays ?? 0 })
          : t(m.causeReputation)
  const looks: { icon: 'ok' | 'lead' | 'bad'; title: string; body: string }[] = [
    {
      icon: failed.length ? (g.cause === 'auth' ? 'lead' : 'bad') : 'ok',
      title: t(m.lookAuth),
      body: failed.length ? t(m.lookAuthFailed, { failed: andList(failed), count: failCount }) : g.authProviders > 2 ? t(m.lookAuthPassAll) : g.authProviders === 2 ? t(m.lookAuthPassBoth) : t(m.lookAuthPass),
    },
    {
      icon: r && r.blocks > 0 ? (g.cause === 'readiness' ? 'lead' : 'bad') : 'ok',
      title: t(m.lookReadiness),
      body:
        d.contentKind !== 'campaign'
          ? t(m.lookReadinessNotCampaign)
          : !r
            ? t(m.lookReadinessNone)
            : r.blocks > 0
              ? t(m.lookReadinessBlocks, { blocks: r.blocks, labels: andList(r.blocking) })
              : t(m.lookReadinessPass, { name: d.contentName, passed: r.passed, warnings: r.warnings }),
    },
    {
      icon: g.other ? (g.other.spam < g.spam ? 'ok' : 'lead') : 'ok',
      title: t(m.lookOther),
      body: g.other ? t(m.lookOtherResult, { provider: g.other.name, spam: g.other.spam, copies: g.other.copies, inbox: g.other.inbox }) : g.cause === 'readiness' ? t(m.lookOtherNoneBlocked) : t(m.lookOtherNone),
    },
    {
      icon: g.cause === 'warmup' || g.cause === 'reputation' ? 'lead' : 'ok',
      title: warming ? t(m.lookWarmup, { provider: g.providerName }) : t(m.lookState, { provider: g.providerName }),
      body: [
        warming
          ? g.hostRepute || !g.ip
            ? t(m.lookWarmupLine, { day: g.warmDay ?? 0, days: g.warmDays ?? 0 })
            : t(m.lookWarmupNoHR, { day: g.warmDay ?? 0, days: g.warmDays ?? 0, ip: g.ip })
          : g.status === 'healthy'
            ? t(m.lookStateHealthy)
            : t(m.lookStateOther, { status: g.status }),
        g.earlier ? t(m.lookEarlier, { content: g.earlier.contentName, inbox: g.earlier.inbox, copies: g.earlier.copies, provider: d.providers.find((p) => p.id === g.providerId)?.short ?? g.providerName, date: shortDate(g.earlier.startedAt) }) : '',
      ]
        .filter(Boolean)
        .join(' '),
    },
  ]
  const buttons: ReactNode[] = []
  if (g.cause === 'readiness' && d.contentId && d.contentKind === 'campaign') {
    buttons.push(
      <ButtonLink key="review" size="sm" variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: d.contentId }}>
        {t(m.openReview)}
      </ButtonLink>,
      <ButtonLink key="edit" size="sm" to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: d.contentId }}>
        {t(m.editComposer)}
      </ButtonLink>,
    )
  }
  if (g.cause === 'auth') {
    buttons.push(
      <ButtonLink key="senders" size="sm" variant="primary" to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
        {t(m.openSenders)}
      </ButtonLink>,
    )
  }
  if (g.cause === 'warmup' || g.cause === 'reputation') {
    if (warming)
      buttons.push(
        <ButtonLink key="warmup" size="sm" to="/w/$workspace/providers/warmup" params={{ workspace: slug }}>
          {t(m.warmupPlan)}
        </ButtonLink>,
      )
    buttons.push(
      <ButtonLink key="health" size="sm" to="/w/$workspace/deliverability" params={{ workspace: slug }}>
        {t(m.healthLink)}
      </ButtonLink>,
    )
    if (g.hostRepute && g.ip)
      buttons.push(
        <ButtonAnchor key="hr" size="sm" href={HOSTREPUTE} icon="external">
          {t(m.checkInHostRepute, { ip: g.ip })}
        </ButtonAnchor>,
      )
  }
  return (
    <section className="card min-w-0" style={{ borderColor: '#FCD34D' }} aria-label={t(m.guidanceLabel)}>
      <div className="sec" style={{ background: '#FFFBEB', borderRadius: '10px 10px 0 0', borderBottomColor: '#FDE68A' }}>
        <div className="flex min-w-0 flex-nowrap items-start gap-2.5">
          <span className="mt-0.5 inline-flex flex-none">
            <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <circle cx="8" cy="8" r="6.2" />
              <path d="M8 4.8v3.6M8 10.9v.1" />
            </svg>
          </span>
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2">
              {g.at.length
                ? t(m.guidanceTitleAt, { provider: g.providerName, spam: g.spam, copies: g.copies, places: andList(g.at) })
                : t(m.guidanceTitle, { provider: g.providerName, spam: g.spam, copies: g.copies })}
            </h2>
            <span className="k" style={{ color: '#78350F' }}>
              {t(m.guidanceRules)}
            </span>
          </div>
        </div>
      </div>
      <div className="pad flex flex-col gap-3.5">
        <p className="m-0 max-w-[880px] text-[14px] leading-[1.55] text-[#1F2937]">{cause}</p>
        <div className="flex flex-col gap-1">
          <span className="k font-semibold text-ink-nav!">{t(m.lookAt)}</span>
          <div className="pl-chk2">
            {looks.map((l) => (
              <div key={l.title}>
                <span className="mt-px inline-flex flex-none">
                  <LookIcon kind={l.icon} />
                </span>
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className="font-semibold">{l.title}</span>
                  <span className="k text-ink-nav!">{l.body}</span>
                </span>
              </div>
            ))}
          </div>
        </div>
        {buttons.length ? <div className="flex flex-wrap gap-2">{buttons}</div> : null}
      </div>
    </section>
  )
}
