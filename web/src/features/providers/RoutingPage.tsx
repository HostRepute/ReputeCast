import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Field, Input, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Banner, Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { ago, clock, count, trimPct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { onProviderColor, perSec, providerColor, ProvidersTabs, streamName, throwIfForbidden, type FailoverPolicy, type Stream } from './common'
import { BulkWarningView } from './BulkWarning'
import { m } from './messages'
import { RouteTester } from './RouteTester'
import { mr } from './routing.messages'
import {
  afterSave,
  bulkIssues,
  changedStreams,
  discardStreams,
  fieldText,
  keepEditsOnLatest,
  mailboxName,
  providerOption,
  rebase,
  sameDraft,
  setPrimary,
  setShare,
  startEditing,
  toSaveInput,
  type Condition,
  type Draft,
  type Editing,
  type RoutingConfig,
  type RoutingProvider,
  type StreamDraft,
} from './routing-model'

const routingKey = ['get', '/api/v1/workspaces/{workspace}/routing'] as const
const providersKey = ['get', '/api/v1/workspaces/{workspace}/providers'] as const

/** 7.3 Routing & failover: per-stream primary, split, conditions, fallback and failover policy. */
export function RoutingPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  if (q.isError || q.isPending) {
    return (
      <Page title={t(m.title)} filters={<ProvidersTabs />}>
        {q.isError ? (
          <ErrorPanel title={t(mr.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)} {t(mr.loadFailedBody)}
          </ErrorPanel>
        ) : (
          <RoutingSkeleton />
        )}
      </Page>
    )
  }
  return <RoutingEditor cfg={q.data} />
}

function RoutingEditor({ cfg }: { cfg: RoutingConfig }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('providers:manage')
  // The edits survive refetches: streams nobody edited follow the saved
  // routing, edited ones keep their edits and the version they started from.
  const [ed, setEd] = useState<Editing>(() => startEditing(cfg))
  const [seen, setSeen] = useState(cfg)
  if (seen !== cfg) {
    setSeen(cfg)
    setEd((e) => rebase(e, cfg))
  }
  const { draft } = ed
  const saved = ed.base
  const setDraft = (next: Draft | ((d: Draft) => Draft)) => setEd((e) => ({ ...e, draft: typeof next === 'function' ? next(e.draft) : next }))
  // Streams a save was refused for: someone changed them meanwhile.
  const [conflict, setConflict] = useState<Stream[]>([])
  const [conditionFor, setConditionFor] = useState<Stream | null>(null)
  const [saving, setSaving] = useState(false)
  const dirty = !sameDraft(draft, saved)
  const issues = bulkIssues(draft, cfg.providers)
  const warningView = dirty && issues.length > 0
  const changedList = changedStreams(draft, saved)
  const changes = changedList.length
  const update = (s: Stream, fn: (sd: StreamDraft) => StreamDraft) => setDraft((d) => ({ ...d, [s]: fn(d[s]) }))

  const save = async (next: Draft, acknowledge = false): Promise<boolean> => {
    setSaving(true)
    const body = toSaveInput(ed, next, acknowledge)
    const which = body.streams.map((x) => x.stream as Stream)
    try {
      const res = await http.PUT('/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } }, body })
      const answer = res.data
      if (answer) setEd((e) => afterSave({ ...e, draft: next }, answer, which))
      setConflict([])
      toast({ kind: 'success', title: t(mr.saved), detail: t(mr.savedDetail) })
      await Promise.all([qc.invalidateQueries({ queryKey: routingKey }), qc.invalidateQueries({ queryKey: providersKey })])
      return true
    } catch (e) {
      if (e instanceof ApiError && e.code === 'routing_changed') {
        const stale = ((e.problem.meta?.streams as string[] | undefined) ?? which) as Stream[]
        setConflict(stale)
        await qc.invalidateQueries({ queryKey: routingKey })
        return false
      }
      const detail = e instanceof ApiError && e.code === 'bulk_terms' ? e.message : errorMessage(e)
      toast({ kind: 'error', title: t(mr.saveFailed), detail })
      return false
    } finally {
      setSaving(false)
    }
  }

  const actions = (
    <>
      {dirty ? (
        <Badge tone="warn" dot>
          {issues.length ? t(mr.unsavedWarnings, { changes, warnings: issues.length }) : t(mr.unsaved, { changes })}
        </Badge>
      ) : cfg.savedAt ? (
        <span className="k">{cfg.savedBy ? t(mr.savedAgoBy, { ago: ago(cfg.savedAt), name: cfg.savedBy }) : t(mr.savedAgo, { ago: ago(cfg.savedAt) })}</span>
      ) : null}
      {dirty ? <Button onClick={() => setEd(startEditing(cfg))}>{t(mr.discard)}</Button> : null}
      <Button
        variant="primary"
        disabledReason={reason}
        loading={saving}
        onClick={() => {
          if (!dirty) {
            toast({ kind: 'info', title: t(mr.upToDate), detail: t(mr.upToDateDetail) })
            return
          }
          if (warningView) {
            document.getElementById('pv-fix')?.scrollIntoView({ block: 'center' })
            return
          }
          void save(draft)
        }}
      >
        {t(mr.saveRouting)}
      </Button>
    </>
  )

  if (warningView) {
    return (
      <Page title={t(m.title)} filters={<ProvidersTabs />} actions={actions} contentClassName="max-w-[1120px]">
        <BulkWarningView
          cfg={cfg}
          draft={draft}
          saved={saved}
          issues={issues}
          saving={saving}
          onDraft={setDraft}
          onSave={async (next, acknowledge) => {
            await save(next, acknowledge)
          }}
          readOnlyReason={reason}
        />
      </Page>
    )
  }

  return (
    <Page title={t(m.title)} filters={<ProvidersTabs />} actions={actions}>
      <UnsavedChangesGuard
        dirty={dirty}
        title={t(mr.leaveTitle)}
        description={t(mr.leaveBody, { streams: formatList(changedList.map((s) => streamName[s])), count: changes })}
        saveLabel={t(mr.saveAndLeave)}
        onSave={async () => {
          if (!(await save(draft))) throw new Error(t(mr.saveFailed))
        }}
      />
      {conflict.length ? (
        <ConflictNote
          streams={conflict}
          onDiscard={() => {
            setEd((e) => discardStreams(e, cfg, conflict))
            setConflict([])
          }}
          onKeep={() => {
            setEd((e) => keepEditsOnLatest(e, cfg, conflict))
            setConflict([])
          }}
        />
      ) : null}
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-[2_1_600px] flex-col gap-4">
          <StreamCard stream="broadcast" cfg={cfg} sd={draft.broadcast} onChange={(fn) => update('broadcast', fn)} onAddCondition={() => setConditionFor('broadcast')} readOnlyReason={reason} />
          <StreamCard stream="automation" cfg={cfg} sd={draft.automation} onChange={(fn) => update('automation', fn)} onAddCondition={() => setConditionFor('automation')} readOnlyReason={reason} />
          <StreamCard stream="system" cfg={cfg} sd={draft.system} onChange={(fn) => update('system', fn)} onAddCondition={() => setConditionFor('system')} readOnlyReason={reason} />
          <StreamCard
            stream="transactional"
            cfg={cfg}
            sd={draft.transactional}
            onChange={(fn) => update('transactional', fn)}
            onAddCondition={() => setConditionFor('transactional')}
            readOnlyReason={reason}
          />
        </div>
        <aside className="flex max-w-[460px] min-w-0 flex-[1_1_360px] flex-col gap-4" aria-label={t(mr.asideLabel)}>
          <RouteTester cfg={cfg} draft={dirty ? draft : undefined} />
          <SharedCard cfg={cfg} />
        </aside>
      </div>
      {conditionFor ? (
        <ConditionDialog
          stream={conditionFor}
          cfg={cfg}
          onClose={() => setConditionFor(null)}
          onAdd={(c) => {
            update(conditionFor, (sd) => ({ ...sd, conditions: [...sd.conditions, c] }))
            setConditionFor(null)
          }}
        />
      ) : null}
    </Page>
  )
}

const streamTag: Record<Stream, MessageDescriptor> = {
  broadcast: mr.tagBroadcast,
  automation: mr.tagAutomation,
  system: mr.tagSystem,
  transactional: mr.tagTransactional,
}

/** One stream's route: primary and fallback, split, conditions and failover. */
function StreamCard({
  stream,
  cfg,
  sd,
  onChange,
  onAddCondition,
  readOnlyReason,
}: {
  stream: Stream
  cfg: RoutingConfig
  sd: StreamDraft
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  onAddCondition: () => void
  readOnlyReason?: string
}) {
  const t = useT()
  const view = cfg.streams.find((s) => s.stream === stream)
  const { slug, me } = useWorkspace()
  // Transactional mail is "from Acme Cloud": the product name in its settings.
  const settings = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/transactional/settings',
    { params: { path: { workspace: slug } } },
    { enabled: stream === 'transactional' && me.permissions.includes('transactional:view') },
  )
  const tag = stream === 'transactional' && settings.data?.productName ? t(mr.tagTransactionalNamed, { product: settings.data.productName }) : t(streamTag[stream])
  const volume = view?.volume ? t(mr.volume, { per: view.volume.per, count: count(view.volume.count) }) : t(mr.noMailYet)
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const primary = sd.splits[0] ? byId.get(sd.splits[0].providerId) : undefined
  const ro = Boolean(readOnlyReason)
  const titleId = `pv-stream-${stream}`
  const addBroadcastCondition = (
    <Button size="sm" className="self-start" disabledReason={readOnlyReason} onClick={onAddCondition}>
      {t(mr.addConditionBroadcast)}
    </Button>
  )
  return (
    <section className="card flex flex-col" aria-labelledby={titleId}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <div className="flex items-center gap-2.5 max-md:flex-wrap">
          <h2 className="h2" id={titleId}>
            {streamName[stream]}
          </h2>
          <Badge>{tag}</Badge>
        </div>
        <span className="k">
          {volume}
          {stream === 'transactional' && view?.senderDomains?.length ? ` · ${view.senderDomains.join(', ')}` : ''}
        </span>
      </div>
      <div className="flex flex-col gap-3.5 px-5 py-[18px]">
        {/* Problems with the saved route the server found (an unavailable provider…); bulk terms have their own review. */}
        {cfg.issues
          .filter((i) => i.stream === stream && i.code !== 'bulk_terms')
          .map((i) => (
            <Note key={`${i.code}-${i.field ?? ''}-${i.providerId ?? ''}`} tone={i.blocking ? 'bad' : 'warn'}>
              {i.message}
            </Note>
          ))}
        <Grid min={240} gap={14}>
          <Field label={t(mr.primary)}>
            <Select value={primary?.id ?? ''} disabled={ro} onChange={(e) => onChange((s) => setPrimary(s, e.target.value))}>
              {!primary ? <option value="">{t(mr.chooseProvider)}</option> : null}
              {cfg.providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {providerOption(p)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t(mr.fallback)}>
            <Select value={sd.fallback} disabled={ro} onChange={(e) => onChange((s) => ({ ...s, fallback: e.target.value }))}>
              <option value="">{stream === 'broadcast' ? t(mr.noneAsk) : t(mr.none)}</option>
              {cfg.providers
                .filter((p) => p.id !== primary?.id)
                .map((p) => (
                  <option key={p.id} value={p.id}>
                    {providerOption(p, { warming: true })}
                  </option>
                ))}
            </Select>
          </Field>
        </Grid>
        {sd.splits.length > 1 ? <SplitEditor sd={sd} byId={byId} onChange={onChange} readOnly={ro} /> : null}
        {sd.conditions.length ? (
          <Conditions sd={sd} cfg={cfg} onChange={onChange} readOnly={ro}>
            {stream === 'broadcast' ? addBroadcastCondition : null}
          </Conditions>
        ) : null}
        {stream === 'transactional' ? (
          <TransactionalParts cfg={cfg} sd={sd} onChange={onChange} onAddCondition={onAddCondition} readOnlyReason={readOnlyReason} />
        ) : stream === 'broadcast' ? (
          <>
            {sd.conditions.length ? null : addBroadcastCondition}
            <FailoverPolicy sd={sd} cfg={cfg} onChange={onChange} readOnly={ro} />
          </>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-2.5">
              <Badge tone={sd.failover === 'pause_and_ask' ? 'warn' : 'ok'} dot>
                {t(mr.failoverBadge, { policy: sd.failover })}
              </Badge>
              <span className="k">{stream === 'system' ? t(mr.systemNote) : t(mr.automationNote)}</span>
            </div>
            {stream === 'automation' ? (
              <Button size="sm" className="self-start" disabledReason={readOnlyReason} onClick={onAddCondition}>
                {t(mr.addCondition)}
              </Button>
            ) : null}
          </>
        )}
      </div>
    </section>
  )
}

/**
 * The Transactional stream (ProvidersRouting): first in line on providers
 * that also send campaigns, automatic failover, and conditions by category
 * or mailbox provider; bulk-marketing terms don't apply to it.
 */
function TransactionalParts({
  cfg,
  sd,
  onChange,
  onAddCondition,
  readOnlyReason,
}: {
  cfg: RoutingConfig
  sd: StreamDraft
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  onAddCondition: () => void
  readOnlyReason?: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const ro = Boolean(readOnlyReason)
  const cards = $api.useQuery('get', '/api/v1/workspaces/{workspace}/providers', { params: { path: { workspace: slug } } })
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const broadcast = cfg.streams.find((s) => s.stream === 'broadcast')
  // The provider of this route that also carries campaigns, and its rate.
  const shared = [sd.splits[0]?.providerId, sd.fallback]
    .filter(Boolean)
    .map((id) => byId.get(id!))
    .find((p) => p && broadcast?.splits.some((x) => x.providerId === p.id))
  const rate = shared ? cards.data?.providers.find((c) => c.id === shared.id)?.rate.max : undefined
  const id = useId()
  return (
    <>
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-[10px] border border-line px-3.5 py-3">
        <span className="flex min-w-0 flex-[1_1_300px] flex-col">
          <span className="font-semibold" id={`${id}-p`}>
            {t(mr.firstInLine)}
          </span>
          <span className="k">
            {shared
              ? rate
                ? t(mr.firstInLineSharedRate, { provider: shared.short, rate: perSec(rate) })
                : t(mr.firstInLineShared, { provider: shared.short })
              : t(mr.firstInLineAny)}
          </span>
        </span>
        <input
          type="checkbox"
          role="switch"
          className="tg"
          aria-labelledby={`${id}-p`}
          checked={sd.priority}
          disabled={ro}
          onChange={(e) => onChange((s) => ({ ...s, priority: e.target.checked }))}
        />
      </div>
      <div className="flex flex-wrap items-center gap-2.5">
        <Badge tone={sd.failover === 'pause_and_ask' ? 'warn' : 'ok'} dot>
          {t(mr.failoverBadge, { policy: sd.failover })}
        </Badge>
      </div>
      <span className="k">{t(mr.txConditions, { count: sd.conditions.length })}</span>
      <div className="flex flex-wrap items-center justify-end gap-2.5">
        <span className="flex flex-wrap gap-2">
          <Button size="sm" disabledReason={readOnlyReason} onClick={onAddCondition}>
            {t(mr.addCondition)}
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/transactional" params={{ workspace: slug }}>
            {t(mr.templatesAndLog)}
          </ButtonLink>
        </span>
      </div>
    </>
  )
}

/** The weighted split bar and the share of the secondary providers. */
function SplitEditor({
  sd,
  byId,
  onChange,
  readOnly,
}: {
  sd: StreamDraft
  byId: Map<string, RoutingProvider>
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  readOnly: boolean
}) {
  const t = useT()
  const warming = sd.splits.map((s) => byId.get(s.providerId)).find((p) => p?.warmingUp)
  const total = sd.splits.reduce((n, s) => n + s.weight, 0) || 1
  const label = sd.splits.map((s) => `${byId.get(s.providerId)?.short ?? t(mr.unknown)} ${trimPct(s.weight)}`).join(', ')
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between max-md:flex-wrap">
        <span className="text-[13px] font-semibold">{t(mr.weightedSplit)}</span>
        <span className="k">{warming ? t(mr.splitUseWarming, { provider: warming.short }) : t(mr.splitUse)}</span>
      </div>
      <div className="flex h-7 gap-0.5 overflow-hidden rounded-[8px]" role="img" aria-label={t(mr.splitLabel, { split: label })}>
        {sd.splits.map((s, i) => {
          const p = byId.get(s.providerId)
          if (!p || s.weight <= 0) return null
          return (
            <div
              key={s.providerId}
              className="flex items-center overflow-hidden text-[12.5px] font-semibold whitespace-nowrap"
              style={{ width: `${(s.weight / total) * 100}%`, background: providerColor[p.color], color: onProviderColor[p.color], padding: i === 0 ? '0 10px' : '0 8px' }}
            >
              {i === 0 ? `${p.name} · ${trimPct(s.weight)}` : `${p.short} ${trimPct(s.weight)}`}
            </div>
          )
        })}
      </div>
      {sd.splits.slice(1).map((s) => {
        const p = byId.get(s.providerId)
        if (!p) return null
        return (
          <div key={s.providerId} className="flex flex-wrap items-center gap-2.5">
            <ShareInput provider={p} value={s.weight} readOnly={readOnly} onChange={(v) => onChange((x) => setShare(x, p.id, v))} />
            {p.shareSteps.length ? (
              <span className="k">{t(mr.plannerRaises, { steps: p.shareSteps.map((st) => t(mr.plannerStep, { share: trimPct(st.share), day: st.day })).join(', ') })}</span>
            ) : null}
          </div>
        )
      })}
    </div>
  )
}

function ShareInput({ provider, value, onChange, readOnly }: { provider: RoutingProvider; value: number; onChange: (v: number) => void; readOnly: boolean }) {
  const t = useT()
  // While typing, the field shows what was typed; otherwise the draft's share.
  const [typing, setTyping] = useState<string | null>(null)
  return (
    <label className="lbl flex-row items-center gap-2">
      {t(mr.shareLabel, { provider: provider.short })}
      <input
        className="in w-[70px]"
        inputMode="numeric"
        value={typing ?? String(value)}
        disabled={readOnly}
        aria-label={t(mr.sharePercent, { provider: provider.short })}
        onFocus={() => setTyping(String(value))}
        onChange={(e) => {
          setTyping(e.target.value)
          const n = Number(e.target.value)
          if (e.target.value.trim() !== '' && !Number.isNaN(n)) onChange(n)
        }}
        onBlur={() => setTyping(null)}
      />
      <span>%</span>
    </label>
  )
}

/** Conditions evaluated before the split. */
function Conditions({
  sd,
  cfg,
  onChange,
  readOnly,
  children,
}: {
  sd: StreamDraft
  cfg: RoutingConfig
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  readOnly: boolean
  children?: ReactNode
}) {
  const t = useT()
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const lists = new Map(cfg.lists.map((l) => [l.id, l.name]))
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] font-semibold">{sd.splits.length > 1 ? t(mr.conditionsBeforeSplit) : t(mr.conditionsBeforePrimary)}</span>
      {sd.conditions.map((c, i) => {
        const values = c.field === 'list' ? c.values.map((v) => lists.get(v) ?? v) : c.field === 'mailbox_provider' ? c.values.map((v) => mailboxName[v] ?? v) : c.values
        const split = (c.splits ?? []).filter((x) => x.weight > 0)
        const p = byId.get(c.providerId ?? split[0]?.providerId ?? '')
        const then =
          split.length > 1
            ? split.map((x) => t(mr.condShare, { provider: byId.get(x.providerId)?.name ?? t(mr.condUnknownProvider), share: trimPct(x.weight) })).join(' · ')
            : p
              ? t(mr.condOnly, { provider: p.name })
              : t(mr.condUnknownProvider)
        const note = [c.note, split.length > 1 ? t(mr.condSticky) : ''].filter(Boolean).join(' · ')
        return (
          <div key={i} className="pv-cond">
            <span className="k">{t(mr.condIf)}</span>
            <span className="pv-chip">{fieldText[c.field]}</span>
            <span className="pv-chip">{values.length > 1 ? t(mr.condIsOneOf) : t(mr.condIs)}</span>
            <span className="pv-chip">{values.join(', ')}</span>
            <span className="k">{t(mr.condThen)}</span>
            <span className="pv-chip then">{then}</span>
            {note ? <span className="k ml-auto">{note}</span> : <span className="ml-auto" />}
            <Button
              size="sm"
              aria-label={t(mr.removeCondition, { field: fieldText[c.field], values: values.join(', ') })}
              disabled={readOnly}
              onClick={() => onChange((s) => ({ ...s, conditions: s.conditions.filter((_, j) => j !== i) }))}
            >
              {t(mr.remove)}
            </Button>
          </div>
        )
      })}
      {children}
    </div>
  )
}

/** Broadcasts failover policy: pause and ask, warmed-up only, or automatic. */
function FailoverPolicy({ sd, cfg, onChange, readOnly }: { sd: StreamDraft; cfg: RoutingConfig; onChange: (fn: (sd: StreamDraft) => StreamDraft) => void; readOnly: boolean }) {
  const t = useT()
  const name = useId()
  const warming = cfg.providers.find((p) => p.warmingUp)
  const opts: { value: FailoverPolicy; title: ReactNode; detail?: ReactNode }[] = [
    {
      value: 'pause_and_ask',
      title: <T msg={mr.pauseAndAsk} values={{ b: (c) => <span className="font-semibold">{c}</span>, k: (c) => <span className="k">{c}</span> }} />,
      detail: t(mr.pauseDetail),
    },
    {
      value: 'warm_only',
      title: <span className="font-semibold">{t(mr.warmOnly)}</span>,
      detail: warming ? t(mr.warmOnlyDetail, { provider: warming.short, day: warming.warmupDays ?? 21 }) : t(mr.warmOnlyNone),
    },
    { value: 'automatic', title: <span className="font-semibold">{t(mr.automatic)}</span> },
  ]
  return (
    <div className="flex flex-col gap-2 rounded-[8px] border border-line bg-ground-3 px-3.5 py-3" role="radiogroup" aria-labelledby={`${name}-t`}>
      <span className="text-[13px] font-semibold" id={`${name}-t`}>
        {t(mr.failoverPolicy)}
      </span>
      {opts.map((o) => (
        <label key={o.value} className="pv-tg">
          <input type="radio" name={name} checked={sd.failover === o.value} disabled={readOnly} onChange={() => onChange((s) => ({ ...s, failover: o.value }))} />
          <span>
            {o.title}
            {o.detail ? (
              <>
                <br />
                <span className="k">{o.detail}</span>
              </>
            ) : null}
          </span>
        </label>
      ))}
    </div>
  )
}

/** Shared across providers: suppression list, throttles, auto-pause. */
function SharedCard({ cfg }: { cfg: RoutingConfig }) {
  const t = useT()
  const { slug } = useWorkspace()
  const row = (k: string, v: string) => (
    <div className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
      <span>{k}</span>
      <span className="n">{v}</span>
    </div>
  )
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-[18px]">
      <h2 className="h2">{t(mr.sharedTitle)}</h2>
      {row(
        t(mr.sharedSuppression),
        cfg.shared.syncedAt ? t(mr.sharedSuppressionSynced, { count: count(cfg.shared.suppressions), time: clock(cfg.shared.syncedAt) }) : count(cfg.shared.suppressions),
      )}
      {row(t(mr.sharedThrottles), t(mr.sharedThrottlesValue))}
      {row(t(mr.sharedThresholds), t(mr.sharedThresholdsValue))}
      <Link to="/w/$workspace/providers/events" params={{ workspace: slug }} className="text-[13px] font-medium">
        {t(mr.openEvents)}
      </Link>
    </section>
  )
}

/**
 * Adds a condition: when a field matches, send through one provider, or a
 * split by weight of its own (PR-12), sticky per address like the stream's.
 */
function ConditionDialog({ stream, cfg, onClose, onAdd }: { stream: Stream; cfg: RoutingConfig; onClose: () => void; onAdd: (c: Condition) => void }) {
  const t = useT()
  const [field, setField] = useState<Condition['field']>(stream === 'transactional' ? 'category' : 'mailbox_provider')
  const [values, setValues] = useState<string[]>([])
  const [providerId, setProviderId] = useState(cfg.providers.find((p) => !p.warmingUp)?.id ?? cfg.providers[0]?.id ?? '')
  const [mode, setMode] = useState<'one' | 'split'>('one')
  const [split, setSplit] = useState<{ providerId: string; weight: number }[]>(() => {
    const ids = cfg.providers.map((p) => p.id)
    return [
      { providerId: providerId || ids[0] || '', weight: 70 },
      { providerId: ids.find((id) => id !== providerId) ?? '', weight: 30 },
    ]
  })
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const [splitError, setSplitError] = useState('')
  const thenName = useId()
  const total = split.reduce((n, x) => n + x.weight, 0)
  const setRow = (i: number, patch: Partial<{ providerId: string; weight: number }>) => setSplit((cur) => cur.map((x, j) => (j === i ? { ...x, ...patch } : x)))
  const choices: { value: string; label: string }[] =
    field === 'mailbox_provider'
      ? Object.entries(mailboxName).map(([value, label]) => ({ value, label }))
      : field === 'list'
        ? cfg.lists.map((l) => ({ value: l.id, label: l.name }))
        : field === 'category'
          ? cfg.categories.map((c) => ({ value: c.id, label: c.name }))
          : cfg.senderDomains.map((d) => ({ value: d, label: d }))
  const toggle = (v: string) => setValues((cur) => (cur.includes(v) ? cur.filter((x) => x !== v) : [...cur, v]))
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mr.conditionTitle, { stream: streamName[stream] })}
      description={t(mr.conditionBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button
            variant="primary"
            onClick={() => {
              if (!values.length) {
                setError(t(mr.pickValue))
                return
              }
              const labels = values.map((v) => choices.find((c) => c.value === v)?.label ?? v)
              if (mode === 'split') {
                const rows = split.filter((x) => x.providerId && x.weight > 0)
                if (new Set(rows.map((x) => x.providerId)).size !== rows.length || rows.length < 2) {
                  setSplitError(t(mr.splitTwoProviders))
                  return
                }
                if (total !== 100) {
                  setSplitError(t(mr.splitMust100))
                  return
                }
                onAdd({ field, values, providerId: rows[0]!.providerId, splits: rows, note: note.trim() || undefined, valueLabels: labels })
                return
              }
              onAdd({ field, values, providerId, note: note.trim() || undefined, valueLabels: labels })
            }}
          >
            {t(mr.addCondition)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <Field label={t(mr.labelIf)}>
          <Select
            value={field}
            onChange={(e) => {
              setField(e.target.value as Condition['field'])
              setValues([])
              setError('')
            }}
          >
            {stream === 'transactional' ? <option value="category">{t(mr.fieldCategory)}</option> : null}
            <option value="mailbox_provider">{t(mr.fieldMailbox)}</option>
            {stream === 'transactional' ? null : (
              <>
                <option value="list">{t(mr.fieldList)}</option>
                <option value="sender_domain">{t(mr.fieldDomain)}</option>
              </>
            )}
          </Select>
        </Field>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1 p-0">{t(mr.legendIsOneOf)}</legend>
          {choices.length ? (
            choices.map((c) => <Checkbox key={c.value} label={c.label} checked={values.includes(c.value)} onChange={() => toggle(c.value)} />)
          ) : (
            <span className="k">{t(mr.nothingToPick)}</span>
          )}
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
        </fieldset>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1 p-0">{t(mr.thenLegend)}</legend>
          <div className="flex flex-wrap gap-x-4 gap-y-1" role="radiogroup" aria-label={t(mr.thenLegend)}>
            <label className="pv-tg">
              <input type="radio" name={thenName} checked={mode === 'one'} onChange={() => setMode('one')} />
              <span>{t(mr.thenOne)}</span>
            </label>
            <label className="pv-tg">
              <input type="radio" name={thenName} checked={mode === 'split'} onChange={() => setMode('split')} disabled={cfg.providers.length < 2} />
              <span>{t(mr.thenSplit)}</span>
            </label>
          </div>
          {mode === 'one' ? (
            <Field label={t(mr.thenSend)}>
              <Select value={providerId} onChange={(e) => setProviderId(e.target.value)}>
                {cfg.providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {providerOption(p, { warming: true })}
                  </option>
                ))}
              </Select>
            </Field>
          ) : (
            <div className="flex flex-col gap-2">
              {split.map((row, i) => {
                const name = cfg.providers.find((p) => p.id === row.providerId)?.name ?? t(mr.splitProviderLabel, { n: i + 1 })
                return (
                  <div key={i} className="flex flex-wrap items-end gap-2">
                    <Field label={t(mr.splitProviderLabel, { n: i + 1 })} className="min-w-[180px] flex-1">
                      <Select value={row.providerId} onChange={(e) => setRow(i, { providerId: e.target.value })}>
                        {cfg.providers.map((p) => (
                          <option key={p.id} value={p.id}>
                            {providerOption(p, { warming: true })}
                          </option>
                        ))}
                      </Select>
                    </Field>
                    <Field label={t(mr.splitShareLabel, { provider: name })} className="w-[110px]">
                      <Input
                        inputMode="numeric"
                        value={String(row.weight)}
                        onChange={(e) => setRow(i, { weight: Math.max(0, Math.min(100, Number(e.target.value.replace(/[^0-9]/g, '')) || 0)) })}
                      />
                    </Field>
                    {split.length > 2 ? (
                      <Button aria-label={t(mr.splitRemoveProvider, { provider: name })} onClick={() => setSplit((cur) => cur.filter((_, j) => j !== i))}>
                        {t(mr.remove)}
                      </Button>
                    ) : null}
                  </div>
                )
              })}
              <div className="flex flex-wrap items-center justify-between gap-2">
                {split.length < Math.min(5, cfg.providers.length) ? (
                  <Button size="sm" onClick={() => setSplit((cur) => [...cur, { providerId: cfg.providers.find((p) => !cur.some((x) => x.providerId === p.id))?.id ?? '', weight: 0 }])}>
                    {t(mr.splitAddProvider)}
                  </Button>
                ) : (
                  <span />
                )}
                <span className={cn('text-[13px]', total === 100 ? 'k' : 'text-warn-fg')} aria-live="polite">
                  {t(mr.splitTotal, { total: trimPct(total) })}
                </span>
              </div>
              <span className="k">{t(mr.thenSplitHint)}</span>
              {splitError ? (
                <span className="err" role="alert">
                  {splitError}
                </span>
              ) : null}
            </div>
          )}
        </fieldset>
        <Field label={t(mr.why)} optional hint={t(mr.whyHint)}>
          <Input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
        </Field>
        {stream === 'broadcast' &&
        cfg.providers.find((p) => (mode === 'one' ? p.id === providerId : split.some((x) => x.weight > 0 && x.providerId === p.id)) && !p.bulkAllowed) ? (
          <Note tone="warn">{t(mr.notBulk)}</Note>
        ) : null}
      </div>
    </Dialog>
  )
}

/**
 * A save was refused because someone changed these streams meanwhile (or
 * the warm-up planner raised a share). Nothing was saved and the edits are
 * still here: show the latest routing for those streams, or keep the edits
 * and replace it on the next save.
 */
function ConflictNote({ streams, onDiscard, onKeep }: { streams: Stream[]; onDiscard: () => void; onKeep: () => void }) {
  const t = useT()
  return (
    <Banner
      tone="warn"
      className="mb-4"
      title={t(mr.conflictTitle, { streams: formatList(streams.map((s) => streamName[s])), count: streams.length })}
      actions={
        <>
          <Button size="sm" onClick={onDiscard}>
            {t(mr.conflictShowLatest)}
          </Button>
          <Button size="sm" onClick={onKeep}>
            {t(mr.conflictKeepMine)}
          </Button>
        </>
      }
    >
      {t(mr.conflictBody)}
    </Banner>
  )
}

/** Loading: the stream cards and the tester, as skeletons. */
function RoutingSkeleton() {
  const t = useT()
  const card = (h: number) => (
    <div className="card flex flex-col gap-3.5 px-5 py-[18px]">
      <Skeleton w={160} h={16} />
      <div className="flex gap-3.5">
        <Skeleton h={36} className="flex-1" />
        <Skeleton h={36} className="flex-1" />
      </div>
      <Skeleton h={h} />
    </div>
  )
  return (
    <Loading label={t(mr.loading)}>
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-[2_1_600px] flex-col gap-4">
          {card(120)}
          {card(40)}
          {card(24)}
        </div>
        <div className="flex max-w-[460px] min-w-0 flex-[1_1_360px] flex-col gap-4">{card(220)}</div>
      </div>
    </Loading>
  )
}
