import { keepPreviousData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useI18n, useT } from '@/i18n'
import { andList, clock, dayMonthDate, orList, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, dayTime, providerColor, useRefreshPlacement, useTests, type Plan, type PlanProvider, type RunInput } from './shared'

export type RunOrigin = 'placement' | 'review' | 'composer' | 'warmup' | 'template'
type ContentRef = RunInput['content']
type Through = RunInput['through']

/** What an entry point already knows; everything stays editable. */
export type RunPreset = {
  origin: RunOrigin
  content?: ContentRef
  through?: Through
  providers?: string[]
  /** The entry point's own name in the eyebrow: "Weekly Digest #142", "Postal, day 12 of 21". */
  from?: string
}

const names = new Map<string, Intl.DisplayNames>()

/** A content language's name in the app's language ("de" → "German"). */
function languageName(code: string, locale: string): string {
  try {
    let n = names.get(locale)
    if (!n) {
      n = new Intl.DisplayNames([locale], { type: 'language' })
      names.set(locale, n)
    }
    return n.of(code) ?? code
  } catch {
    return code
  }
}

/** "$0", "$0.001". */
function cost(v: number): string {
  return v === 0 ? '$0' : `$${Number(v.toPrecision(2))}`
}

const statusTone: Record<PlanProvider['status'], Tone> = { healthy: 'ok', warming_up: 'info', degraded: 'warn', down: 'bad', sandbox: 'warn', paused: 'neutral' }

/**
 * Run a placement test: one copy to each seed mailbox, through the route,
 * one provider or several. The plan comes from the server on every change,
 * so the counts, costs and problems are the ones the test would start with.
 */
export function RunDialog({ open, onOpenChange, preset }: { open: boolean; onOpenChange: (open: boolean) => void; preset: RunPreset }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshPlacement()
  const deleteAfter = useTests().data?.deleteAfter ?? '24h'
  const [content, setContent] = useState<ContentRef | undefined>(preset.content)
  const [variant, setVariant] = useState<string>()
  const [language, setLanguage] = useState<string>()
  const [personal, setPersonal] = useState<'fallbacks' | 'data'>(preset.content?.kind === 'transactional' ? 'data' : 'fallbacks')
  const [testData, setTestData] = useState<string>()
  const [through, setThrough] = useState<Through>(preset.through ?? 'route')
  const [providers, setProviders] = useState<string[] | undefined>(preset.providers)
  const [senderId, setSenderId] = useState<string>()
  const [seeds, setSeeds] = useState<string[]>()
  const [openedAt] = useState(() => new Date().toISOString())
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const body: RunInput | undefined = content
    ? {
        content,
        origin: preset.origin,
        through,
        providers: through === 'route' ? undefined : providers,
        senderId,
        seeds,
        variant,
        language,
        // Test data without a set picked yet is the first one, as the select shows.
        personalization: personal === 'data' ? 'data' : undefined,
        testData: personal === 'data' ? testData : undefined,
      }
    : undefined
  const plan = $api.useQuery('post', `${base}/plan`, { params: { path: { workspace: slug } }, body: body as RunInput }, { enabled: Boolean(body), placeholderData: keepPreviousData })
  const p = plan.data && content && plan.data.content.id === content.id ? plan.data : undefined

  const pickContent = (c: ContentRef) => {
    setContent(c)
    setVariant(undefined)
    setLanguage(undefined)
    setTestData(undefined)
    setSenderId(undefined)
    setPersonal(c.kind === 'transactional' ? 'data' : 'fallbacks')
  }
  const pickThrough = (v: Through) => {
    setThrough(v)
    setProviders(v === 'route' ? undefined : v === 'provider' && providers?.length ? providers.slice(0, 1) : undefined)
    setSenderId(undefined)
  }

  const start = async (override?: Partial<RunInput>) => {
    if (!body) return
    setBusy(true)
    setError(undefined)
    const res = await http.POST(`${base}/tests`, { params: { path: { workspace: slug } }, body: { ...body, ...override } })
    setBusy(false)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    const d = res.data
    void refresh()
    onOpenChange(false)
    toast({ kind: 'success', title: t(m.testStarted), detail: t(m.testStartedDetail, { copies: d.counts.copies, seeds: d.seeds.length }) })
    void navigate({ to: '/w/$workspace/deliverability/placement/$testId', params: { workspace: slug, testId: d.id } })
  }

  const alone = p?.alone ? p.providers.find((o) => o.id === p.alone) : undefined
  const blocked = !content ? t(m.pickContent) : !p ? t(m.loading) : p.canStart ? undefined : (p.problems[0] ?? p.senderProblem ?? t(m.runProblems))
  const extra = p ? p.problems.filter((x) => x !== p.compareProblem && x !== p.senderProblem) : []
  const foot = t(deleteAfter === 'never' ? m.copiesFootNever : deleteAfter === 'finished' ? m.copiesFootFinished : m.copiesFoot24h)

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => !busy && onOpenChange(v)}
      size="lg"
      className="pl-run"
      hideClose
      eyebrow={<span className="k">{t(m.openedFrom, { origin: preset.origin, name: preset.from ?? '', time: dayTime(openedAt) })}</span>}
      title={t(m.runTitle)}
      description={t(m.runLead)}
      footer={
        <>
          {p?.canStart ? <span className="k mr-auto flex-[1_1_240px]">{foot}</span> : null}
          <Button className={p?.canStart ? undefined : 'mr-auto'} onClick={() => onOpenChange(false)}>
            {t(m.cancel)}
          </Button>
          {alone ? (
            <Button
              onClick={() => {
                setThrough('provider')
                setProviders([alone.id])
              }}
            >
              {t(m.testAlone, { provider: alone.name })}
            </Button>
          ) : null}
          <Button variant="primary" loading={busy} busyLabel={t(m.starting)} disabledReason={busy ? undefined : blocked} onClick={() => void start()}>
            {t(m.startTest)}
          </Button>
        </>
      }
    >
      {error ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} className="mt-px flex-none" />
          <span>{error}</span>
        </div>
      ) : null}
      <div className="flex flex-col">
        <Row label={t(m.rowContent)}>
          <ContentRow content={content} plan={p} origin={preset.origin} onPick={pickContent} />
        </Row>
        {content && !p ? (
          plan.isError ? (
            <div className="note b" role="alert">
              <Icon name="alertCircle" size={16} className="mt-px flex-none" />
              <span>
                {t(m.loadPlanError)} · {errorMessage(plan.error)}
              </span>
            </div>
          ) : (
            <div className="flex flex-col gap-3 py-3" aria-busy="true">
              <Skeleton w="60%" h={14} />
              <Skeleton w="100%" h={44} r={8} />
              <Skeleton w="100%" h={44} r={8} />
            </div>
          )
        ) : null}
        {p ? (
          <>
            <VersionRow plan={p} variant={variant} language={language} onVariant={setVariant} onLanguage={setLanguage} />
            <PersonalRow plan={p} mode={personal} onMode={setPersonal} testData={testData} onTestData={setTestData} />
            <Row label={t(m.rowThrough)}>
              <ThroughRow plan={p} through={through} providers={providers ?? p.chosen} onThrough={pickThrough} onProviders={setProviders} />
            </Row>
            <FromRow plan={p} senderId={senderId} onSender={setSenderId} />
            <SeedsRow plan={p} seeds={seeds} onSeeds={setSeeds} />
            {p.copies > 0 ? <CopiesRow plan={p} through={through} /> : null}
            {extra.length ? (
              <div className="note w mt-2" role="status">
                <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
                <div className="flex min-w-0 flex-col gap-1">
                  <span className="font-semibold">{t(m.runProblems)}</span>
                  {extra.map((x) => (
                    <span key={x}>{x}</span>
                  ))}
                </div>
              </div>
            ) : null}
          </>
        ) : null}
      </div>
    </Dialog>
  )
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="pl-dsec">
      <div className="dl">{label}</div>
      <div className="flex min-w-0 flex-col gap-2">{children}</div>
    </div>
  )
}

function ContentIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="flex-none text-ink-3">
      <path d="M4 2.5h5.5L12.5 5.5v8h-8.5z" />
      <path d="M9.5 2.5v3h3M6 8.5h4M6 11h4" />
    </svg>
  )
}

/** "Campaign · scheduled Tue 14 Oct, 09:00 · Weekly Digest list · 31,604 recipients". */
function contentLine(c: Plan['content'], origin: RunOrigin, t: ReturnType<typeof useT>): string {
  const out: string[] = []
  if (c.kind === 'transactional') {
    out.push(t(m.kindTransactional), t(m.contentPublished, { key: c.key ?? '', version: c.version ?? 0 }))
    if (c.category) out.push(c.category)
    return out.join(' · ')
  }
  if (c.kind === 'template') return t(m.kindTemplate)
  if (c.kind === 'digest') {
    out.push(t(m.kindDigest), t(m.contentNextIssue))
    if (c.audience) out.push(t(m.contentAudience, { audience: c.audience }))
    if (c.recipients) out.push(t(m.contentRecipients, { count: c.recipients }))
    return out.join(' · ')
  }
  out.push(t(m.kindCampaign))
  if (c.status === 'scheduled' && c.scheduledAt) out.push(t(m.contentScheduledFor, { date: dayTime(c.scheduledAt) }))
  else if (c.status === 'sent' && c.sentAt) out.push(t(m.contentSentOn, { date: dayTime(c.sentAt) }))
  else {
    out.push(t(m.contentDraft, { version: c.version ?? 0 }))
    if (c.scheduledAt) out.push(t(m.contentFor, { date: dayTime(c.scheduledAt) }))
  }
  if (origin === 'warmup' && c.status === 'sent') out.push(t(m.contentLastSent))
  else {
    if (c.audience && c.status === 'scheduled') out.push(t(m.contentAudience, { audience: c.audience }))
    if (c.recipients) out.push(t(m.contentRecipients, { count: c.recipients }))
  }
  return out.join(' · ')
}

function ContentRow({ content, plan, origin, onPick }: { content?: ContentRef; plan?: Plan; origin: RunOrigin; onPick: (c: ContentRef) => void }) {
  const t = useT()
  const [picking, setPicking] = useState(!content)
  if (picking || !content) {
    return (
      <ContentPicker
        onPick={(c) => {
          onPick(c)
          setPicking(false)
        }}
        onCancel={content ? () => setPicking(false) : undefined}
      />
    )
  }
  return (
    <>
      <div className="pl-picker">
        <ContentIcon />
        <div className="pt">
          {plan ? (
            <>
              <span className="text-[14px] font-semibold">{plan.content.name}</span>
              <span className="k">{contentLine(plan.content, origin, t)}</span>
            </>
          ) : (
            <>
              <Skeleton w={160} h={14} />
              <Skeleton w={260} h={10} />
            </>
          )}
        </div>
        <Button size="sm" onClick={() => setPicking(true)}>
          {t(m.change)}
        </Button>
      </div>
      <span className="k">{t(m.contentHelp)}</span>
    </>
  )
}

function ContentPicker({ onPick, onCancel }: { onPick: (c: ContentRef) => void; onCancel?: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [q, setQ] = useState('')
  const [debounced, setDebounced] = useState('')
  useEffect(() => {
    const id = setTimeout(() => setDebounced(q.trim()), 200)
    return () => clearTimeout(id)
  }, [q])
  const list = $api.useQuery('get', `${base}/content`, { params: { path: { workspace: slug }, query: { q: debounced } } }, { placeholderData: keepPreviousData })
  const items = list.data?.items ?? []
  const label = useId()
  return (
    <div className="flex flex-col gap-2">
      <label className="pl-search" id={label}>
        <Icon name="search" size={15} className="text-ink-3" />
        <input aria-label={t(m.contentSearch)} placeholder={t(m.contentSearch)} value={q} onChange={(e) => setQ(e.target.value)} autoFocus />
      </label>
      <div className="rounded-lg border border-line px-3" role="list" aria-labelledby={label} aria-busy={list.isFetching || undefined}>
        {list.isPending ? (
          <div className="pl-pickrow">
            <span className="k">{t(m.contentLoading)}</span>
          </div>
        ) : items.length === 0 ? (
          <div className="pl-pickrow">
            <span className="k">{t(m.contentNone, { q: debounced })}</span>
          </div>
        ) : (
          items.slice(0, 8).map((c) => (
            <div className="pl-pickrow" role="listitem" key={`${c.kind}-${c.id}`}>
              <ContentIcon />
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="font-semibold">{c.name}</span>
                <span className="k">{optionLine(c, t)}</span>
              </span>
              <Button size="sm" onClick={() => onPick({ kind: c.kind, id: c.id })} aria-label={t(m.pickNamed, { name: c.name })}>
                {t(m.pick)}
              </Button>
            </div>
          ))
        )}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="k">{t(m.contentHelp)}</span>
        {onCancel ? (
          <Button size="sm" variant="ghost" onClick={onCancel}>
            {t(m.cancel)}
          </Button>
        ) : null}
      </div>
    </div>
  )
}

function optionLine(c: { kind: string; note: string; at: string }, t: ReturnType<typeof useT>): string {
  if (c.kind === 'transactional') return `${t(m.kindTransactional)} · ${c.note}`
  if (c.kind === 'template') return t(m.kindTemplate)
  if (c.kind === 'digest') return `${t(m.kindDigest)} · ${t(m.optionNextIssue, { state: c.note })}`
  switch (c.note) {
    case 'scheduled':
      return `${t(m.kindCampaign)} · ${t(m.contentScheduledFor, { date: dayTime(c.at) })}`
    case 'sent':
      return `${t(m.kindCampaign)} · ${t(m.contentSentOn, { date: shortDate(c.at) })}`
  }
  return `${t(m.kindCampaign)} · ${t(m.contentDraft, { version: 0 })}`
}

function VersionRow({ plan, variant, language, onVariant, onLanguage }: { plan: Plan; variant?: string; language?: string; onVariant: (v: string) => void; onLanguage: (v: string) => void }) {
  const t = useT()
  const { locale } = useI18n()
  const c = plan.content
  const vid = useId()
  const lid = useId()
  if (c.variants.length === 0 && c.languages.length <= 1) {
    return (
      <Row label={t(m.rowVersion)}>
        <span className="pt-2 text-[13px] text-ink-2">{c.kind === 'campaign' ? t(m.noConditional) : t(m.noVersions)}</span>
      </Row>
    )
  }
  return (
    <Row label={t(m.rowVersion)}>
      <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(220px,100%),1fr))' }}>
        {c.variants.length ? (
          <div className="flex min-w-0 flex-col gap-1.5">
            <label htmlFor={vid} className="text-[12.5px] font-semibold">
              {t(m.abVariant)}
            </label>
            <select id={vid} className="in" value={variant ?? c.variants[0]!.key} onChange={(e) => onVariant(e.target.value)}>
              {c.variants.map((v) => (
                <option key={v.key} value={v.key}>
                  {t(m.variantOption, { key: v.key, subject: v.subject })}
                </option>
              ))}
            </select>
          </div>
        ) : null}
        {c.languages.length > 1 ? (
          <div className="flex min-w-0 flex-col gap-1.5">
            <label htmlFor={lid} className="text-[12.5px] font-semibold">
              {t(m.language)}
            </label>
            <select id={lid} className="in" value={language ?? c.languages.find((l) => l.source)?.code ?? c.languages[0]!.code} onChange={(e) => onLanguage(e.target.value)}>
              {c.languages.map((l) => (
                <option key={l.code} value={l.code}>
                  {l.source ? t(m.languageSource, { name: languageName(l.code, locale) }) : languageName(l.code, locale)}
                </option>
              ))}
            </select>
          </div>
        ) : null}
      </div>
      {c.kind === 'campaign' ? <span className="k">{t(m.noConditional)}</span> : null}
    </Row>
  )
}

function PersonalRow({ plan, mode, onMode, testData, onTestData }: { plan: Plan; mode: 'fallbacks' | 'data'; onMode: (v: 'fallbacks' | 'data') => void; testData?: string; onTestData: (v: string) => void }) {
  const t = useT()
  const c = plan.content
  const sets = c.testData
  const set = testData ?? sets[0]
  return (
    <Row label={t(m.rowPersonalization)}>
      <div className="flex flex-wrap items-center gap-2.5">
        <div className="seg" role="group" aria-label={t(m.rowPersonalization)}>
          <button type="button" className={mode === 'fallbacks' ? 'on' : undefined} aria-pressed={mode === 'fallbacks'} onClick={() => onMode('fallbacks')}>
            {t(m.fallbacks)}
          </button>
          <button
            type="button"
            className={mode === 'data' ? 'on' : undefined}
            aria-pressed={mode === 'data'}
            aria-disabled={sets.length === 0 || undefined}
            title={sets.length === 0 ? t(m.testDataNone) : undefined}
            onClick={() => sets.length && onMode('data')}
          >
            {t(m.testData)}
          </button>
        </div>
        {mode === 'data' && sets.length ? (
          <select className="in min-w-[200px] flex-1" aria-label={t(m.testDataSet)} value={set} onChange={(e) => onTestData(e.target.value)}>
            {sets.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        ) : null}
      </div>
      <span className="k">
        {mode === 'data' && sets.length
          ? [...c.testValues.map((v) => t(m.testValue, { name: v.name, value: v.value })), ...c.sensitive.map((name) => t(m.sensitiveNote, { name }))].join(' · ')
          : c.fallbackGreeting
            ? t(m.fallbacksHelp, { greeting: c.fallbackGreeting })
            : t(m.fallbacksHelpPlain)}
      </span>
    </Row>
  )
}

function StatusBadge({ o }: { o: PlanProvider }) {
  const t = useT()
  return (
    <Badge tone={statusTone[o.status]} dot className="ml-auto">
      {o.status === 'warming_up'
        ? t(m.providerStatusWarming, { day: o.warmDay ?? 0, days: o.warmDays ?? 0 })
        : t(
            { healthy: m.providerStatusHealthy, degraded: m.providerStatusDegraded, down: m.providerStatusDown, paused: m.providerStatusPaused, sandbox: m.providerStatusSandbox, warming_up: m.providerStatusHealthy }[
              o.status
            ],
          )}
    </Badge>
  )
}

function ThroughRow({ plan, through, providers, onThrough, onProviders }: { plan: Plan; through: Through; providers: string[]; onThrough: (v: Through) => void; onProviders: (v: string[]) => void }) {
  const t = useT()
  const name = useId()
  const option = (value: Through, title: string, body: string, extra?: ReactNode) => {
    const id = `${name}-${value}`
    return (
      <div className={through === value ? 'pl-opt on' : 'pl-opt'}>
        <input id={id} type="radio" name={name} value={value} checked={through === value} onChange={() => onThrough(value)} />
        <div className="ot">
          <label htmlFor={id} className="flex cursor-pointer flex-col gap-0.5">
            <b className="font-semibold">{title}</b>
            {body ? <span className="k text-ink-2!">{body}</span> : null}
          </label>
          {through === value ? extra : null}
        </div>
      </div>
    )
  }
  const usable = plan.providers.filter((o) => o.available)
  // Providers that can send come first, then the ones that are down, then
  // the ones another stream owns.
  const rank = (o: PlanProvider) => (o.available ? 0 : o.streamOnly ? 2 : 1)
  const sorted = [...plan.providers].sort((a, b) => rank(a) - rank(b))
  const one = providers[0] ?? usable[0]?.id ?? ''
  return (
    <>
      {option('route', t(m.throughRouteTitle), t(m.throughRouteBody), <RouteShares plan={plan} />)}
      {option(
        'provider',
        t(m.throughOneTitle),
        through === 'provider' ? '' : t(m.throughOneBody),
        <select className="in mt-1.5 max-w-[320px]" aria-label={t(m.providerPick)} value={one} onChange={(e) => onProviders([e.target.value])}>
          {usable.map((o) => (
            <option key={o.id} value={o.id}>
              {t(m.optionStatus, { name: o.name, status: o.status, day: o.warmDay ?? 0, days: o.warmDays ?? 0 })}
            </option>
          ))}
        </select>,
      )}
      {option(
        'compare',
        t(m.throughCompareTitle),
        t(m.throughCompareBody),
        <div className="mt-2 flex flex-col">
          {sorted.map((o) => (
            <CompareRow
              key={o.id}
              o={o}
              checked={o.available && providers.includes(o.id)}
              onChange={(on) => onProviders(on ? [...providers.filter((x) => x !== o.id), o.id] : providers.filter((x) => x !== o.id))}
            />
          ))}
        </div>,
      )}
      {through === 'compare' && plan.compareProblem ? <CompareProblem plan={plan} providers={providers} /> : null}
    </>
  )
}

function CompareRow({ o, checked, onChange }: { o: PlanProvider; checked: boolean; onChange: (on: boolean) => void }) {
  const t = useT()
  const id = useId()
  const why = o.streamOnly
    ? t(m.streamOnly)
    : o.status === 'down'
      ? t(m.downSince, { time: o.downSince ? clock(o.downSince) : '', reason: o.downReason ?? '' })
      : undefined
  return (
    <div className="border-t border-line-2 py-2 first:border-t-0">
      <div className="flex flex-wrap items-center gap-2.5">
        <input id={id} type="checkbox" checked={checked} disabled={!o.available} onChange={(e) => onChange(e.target.checked)} aria-describedby={why ? `${id}-why` : undefined} />
        <label htmlFor={id} className={o.available ? 'inline-flex items-center gap-2' : 'inline-flex items-center gap-2 text-ink-3'}>
          <span className="pl-sw" style={{ background: providerColor(o.adapter) }} />
          <span className={o.available ? 'font-medium text-ink' : undefined}>{o.name}</span>
          <span className="k">{o.detail}</span>
        </label>
        <StatusBadge o={o} />
      </div>
      {why ? (
        <span id={`${id}-why`} className={o.status === 'down' ? 'mt-1 block pl-[26px] text-[12.5px] text-bad-fg' : 'mt-1 block pl-[26px] text-[12.5px] text-ink-2'}>
          {why}
        </span>
      ) : null}
    </div>
  )
}

function CompareProblem({ plan, providers }: { plan: Plan; providers: string[] }) {
  const t = useT()
  const alone = plan.providers.find((o) => o.id === plan.alone)
  const waiting = plan.providers.filter((o) => !o.available && !o.streamOnly).map((o) => o.name)
  void providers
  return (
    <div className="note w" role="status">
      <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
      <span>{alone && waiting.length ? t(m.compareProblem, { provider: alone.name, other: orList(waiting) }) : t(m.compareProblemNone)}</span>
    </div>
  )
}

/** Which seeds the route gives each provider, and why. */
function RouteShares({ plan }: { plan: Plan }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (plan.route.length === 0) return <span className="k mt-2">{t(m.routeNone)}</span>
  const prov = (id: string) => plan.providers.find((o) => o.id === id)
  const seed = (id: string) => plan.seeds.find((s) => s.id === id)
  return (
    <div className="mt-2 flex flex-col gap-3 rounded-lg border border-line bg-white p-3">
      {plan.route.map((r) => {
        const o = prov(r.providerId)
        return (
          <div className="pl-share" key={r.providerId}>
            <span className="inline-flex items-center gap-2">
              <span className="pl-sw" style={{ background: providerColor(o?.adapter) }} />
              {o?.name}
            </span>
            <span>
              <b className="font-semibold">{t(m.routeSeeds, { count: r.seeds.length })}</b> <ShareReason share={r} plan={plan} seed={seed} />
            </span>
          </div>
        )
      })}
      <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }} className="pl-lnk">
        {t(m.openRouteTester)}
      </Link>
    </div>
  )
}

function ShareReason({ share, plan, seed }: { share: Plan['route'][number]; plan: Plan; seed: (id: string) => Plan['seeds'][number] | undefined }) {
  const t = useT()
  const reason = (r: string): string => {
    if (r.startsWith('condition:')) return t(m.reasonConditionOnly, { condition: conditionName(r.slice(10)) })
    if (r.startsWith('split:')) return t(m.reasonSplit, { weight: Number(r.slice(6)) || 0 })
    if (r === 'fallback') return t(m.reasonFallback)
    return t(m.reasonPrimary, { stream: plan.stream })
  }
  const fallback = share.fallback ? plan.providers.find((o) => o.id === share.fallback)?.name : undefined
  const tail = fallback ? ` · ${t(m.routeFallback, { provider: fallback })}` : ''
  const others = share.reasons.filter((r) => !r.startsWith('condition:'))
  const cond = share.reasons.find((r) => r.startsWith('condition:'))
  if (share.seeds.length === 1) {
    const s = seed(share.seeds[0]!)
    return (
      <span className="k text-ink-2!">
        <span className="mono">{s?.email}</span>
        {`, ${share.reasons.map(reason).join(', ')}${tail}`}
      </span>
    )
  }
  if (cond && others.length && share.matched.length) {
    const who = [...new Set(share.matched.map((id) => seed(id)?.providerName).filter(Boolean))] as string[]
    return (
      <span className="k text-ink-2!">
        {t(m.reasonCondition, { seeds: andList(who), condition: conditionName(cond.slice(10)), rest: others.map(reason).join(', ') })}
        {tail}
      </span>
    )
  }
  return (
    <span className="k text-ink-2!">
      {share.reasons.map(reason).join(', ')}
      {tail}
    </span>
  )
}

/** "outlook" → "Outlook". */
function conditionName(v: string): string {
  return v ? v.charAt(0).toUpperCase() + v.slice(1) : v
}

function FromRow({ plan, senderId, onSender }: { plan: Plan; senderId?: string; onSender: (v: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const id = useId()
  const verified = plan.senders.filter((s) => s.verified)
  const problem = Boolean(plan.senderProblem)
  const others = plan.senderAlternatives ?? []
  const missing = andList(plan.senderMissing ?? [])
  return (
    <Row label={t(m.rowFrom)}>
      <select
        id={id}
        className={problem ? 'in border-bad-line!' : 'in'}
        aria-label={t(m.rowFrom)}
        aria-invalid={problem || undefined}
        value={problem ? '' : (senderId ?? plan.sender ?? '')}
        onChange={(e) => onSender(e.target.value)}
        disabled={problem}
      >
        {problem ? <option value="">{t(m.noVerifiedSender)}</option> : null}
        {verified.map((s) => (
          <option key={s.id} value={s.id}>
            {`${s.name} <${s.email}>`}
          </option>
        ))}
      </select>
      {problem ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} className="mt-px flex-none" />
          <div className="flex flex-col items-start gap-2">
            <span>
              {others.length
                ? t(m.noSenderProblem, { missing, stream: plan.stream, domain: plan.senderDomain ?? '', others: orList(others) })
                : t(m.noSenderProblemAlone, { missing, stream: plan.stream, domain: plan.senderDomain ?? '' })}
            </span>
            <ButtonLink size="sm" to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
              {t(m.openSenders)}
            </ButtonLink>
          </div>
        </div>
      ) : (
        <span className="k">{plan.stream === 'transactional' ? t(m.sendersHelpTransactional) : t(m.sendersHelp, { stream: plan.stream })}</span>
      )}
    </Row>
  )
}

function SeedsRow({ plan, seeds, onSeeds }: { plan: Plan; seeds?: string[]; onSeeds: (v: string[] | undefined) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useId()
  const connected = plan.seeds.filter((s) => s.status === 'connected')
  const unchecked = plan.seeds.filter((s) => s.status === 'failed' || s.status === 'expired')
  const some = seeds !== undefined
  return (
    <Row label={t(m.rowSeeds)}>
      <div className="flex flex-wrap items-center gap-x-5 gap-y-2 pt-1.5" role="radiogroup" aria-label={t(m.rowSeeds)}>
        <label className="inline-flex items-center gap-2 text-[13px] font-semibold">
          <input type="radio" name={name} checked={!some} onChange={() => onSeeds(undefined)} />
          {connected.length === plan.seeds.length ? t(m.allSeeds, { count: connected.length }) : t(m.allSeedsOf, { count: connected.length, total: plan.seeds.length })}
        </label>
        <label className="inline-flex items-center gap-2 text-[13px]">
          <input type="radio" name={name} checked={some} onChange={() => onSeeds(connected.map((s) => s.id))} />
          {t(m.someSeeds)}
        </label>
      </div>
      {some ? (
        <div className="pl-seedgrid">
          {plan.seeds.map((s) => {
            const ok = s.status === 'connected'
            return (
              <label key={s.id} className={ok ? undefined : 'off'} aria-label={`${s.providerName} ${s.email}`}>
                <input
                  type="checkbox"
                  disabled={!ok}
                  checked={ok && seeds.includes(s.id)}
                  onChange={(e) => onSeeds(e.target.checked ? [...seeds, s.id] : seeds.filter((x) => x !== s.id))}
                />
                <span className="flex min-w-0 flex-col" aria-hidden="true">
                  <b className="font-semibold">{s.providerName}</b>
                  <span className="mono">{s.email}</span>
                </span>
              </label>
            )
          })}
        </div>
      ) : null}
      {unchecked.map((s) => (
        <span key={s.id} className="flex items-start gap-2 text-[12.5px] text-bad-fg">
          <Icon name="alertCircle" size={14} className="mt-0.5 flex-none" />
          <span>
            <T
              msg={s.status === 'expired' ? m.seedUncheckedExpired : m.seedUncheckedFailed}
              values={{
                email: s.email,
                date: s.statusAt ? dayMonthDate(s.statusAt) : '',
                mono: (c) => <span className="mono">{c}</span>,
                link: (c) => (
                  <Link to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} search={{ reconnect: s.id }} className="text-bad-fg! underline">
                    {c}
                  </Link>
                ),
              }}
            />
          </span>
        </span>
      ))}
    </Row>
  )
}

function CopiesRow({ plan, through }: { plan: Plan; through: Through }) {
  const t = useT()
  const prov = (id: string) => plan.providers.find((o) => o.id === id)
  const lines: string[] = []
  for (const c of plan.costs) {
    if (c.warmDay && c.todayCap) {
      lines.push(
        through === 'provider'
          ? t(m.warmupCapOne, { count: c.copies, provider: prov(c.providerId)?.name ?? c.short, cap: c.todayCap })
          : t(m.warmupCap, { count: c.copies, provider: prov(c.providerId)?.name ?? c.short, cap: c.todayCap }),
      )
    }
    if (c.pricing === 'plan' && c.planMonthly) lines.push(t(m.planIncluded, { provider: prov(c.providerId)?.name ?? c.short, count: c.planMonthly }))
  }
  if (through === 'provider' && plan.warmupDays.length) lines.push(t(m.earlierDays, { days: andList(plan.warmupDays.map((day) => t(m.earlierDay, { day }))) }))
  if (plan.content.kind === 'campaign' && plan.content.status === 'scheduled') lines.push(t(m.noContactsTouched))
  return (
    <Row label={t(m.rowCopies)}>
      <div className="pl-sumbar flex-col items-start!">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
          <b className="text-[14px] font-bold">{t(m.copiesCount, { count: plan.copies })}</b>
          {plan.costs.map((c) => (
            <span key={c.providerId} className="inline-flex items-center gap-2">
              <span className="pl-sw" style={{ background: providerColor(prov(c.providerId)?.adapter) }} />
              {t(m.costLine, { provider: c.short, cost: cost(c.cost) })}
            </span>
          ))}
        </div>
        {lines.length ? <span className="text-[12.5px] text-ink-nav">{lines.join(' ')}</span> : null}
      </div>
    </Row>
  )
}

