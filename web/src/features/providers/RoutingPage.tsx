import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Field, Input, Select } from '@/components/ui/Field'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Banner, Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, formatList, t as tt, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { ago, clock, count, trimPct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { perSec, providerColor, ProvidersTabs, streamName, throwIfForbidden, type FailoverPolicy, type Stream } from './common'
import { BulkWarningView } from './BulkWarning'
import { m } from './messages'
import { RouteTester, useCampaignSimulation } from './RouteTester'
import { mr } from './routing.messages'
import {
  addToSplit,
  afterSave,
  bulkIssues,
  changedStreams,
  changeText,
  discardStreams,
  fieldText,
  keepEditsOnLatest,
  mailboxName,
  moveSplit,
  providerOption,
  rebase,
  removeFromSplit,
  sameDraft,
  sameStream,
  setPrimary,
  setWeight,
  splitTotal,
  startEditing,
  toSaveInput,
  valueNames,
  weightsProblem,
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
  // Only a first load that fails takes the editor's place: a background
  // refresh that fails keeps the editor and its unsaved edits (R3.U4b).
  if (!q.data) {
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
  return <RoutingEditor cfg={q.data} refreshError={q.isError ? q.error : undefined} onRetry={() => void q.refetch()} />
}

function RoutingEditor({ cfg, refreshError, onRetry }: { cfg: RoutingConfig; refreshError?: unknown; onRetry: () => void }) {
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

  // The impact of the unsaved routing (PR-14, R3.U4a): the simulation runs
  // the next broadcast's audience, and each other stream's recent
  // addresses, through both.
  const sim = useCampaignSimulation(dirty ? draft : undefined, !weightsProblem(draft))
  const impact = sim.data?.draft
    ? movedAcross(sim.data, cfg)
    : undefined
  const problem = weightsProblem(draft)
  const [announce, setAnnounce] = useState('')
  const trySave = () => {
    if (warningView) {
      document.getElementById('pv-fix')?.scrollIntoView({ block: 'center' })
      return
    }
    void save(draft)
  }
  const actions = cfg.savedAt ? (
    <span className="k">{cfg.savedBy ? t(mr.savedAgoBy, { ago: ago(cfg.savedAt), name: cfg.savedBy }) : t(mr.savedAgo, { ago: ago(cfg.savedAt) })}</span>
  ) : null

  const discard = () => {
    setEd(startEditing(cfg))
    setAnnounce(t(mr.discarded))
  }

  // Both views keep the edits safe (R3.U4b): leaving asks first, a refused
  // save says why and offers the latest routing, and a failed refresh says
  // the edits are still here.
  const leaveGuard = (canSave: boolean) => (
    <UnsavedChangesGuard
      dirty={dirty}
      title={t(mr.leaveTitle)}
      description={t(mr.leaveBody, { streams: formatList(changedList.map((s) => streamName[s])), count: changes })}
      saveLabel={t(mr.saveAndLeave)}
      onSave={
        canSave
          ? async () => {
              if (!(await save(draft))) throw new Error(t(mr.saveFailed))
            }
          : undefined
      }
    />
  )
  const notices = (
    <>
      {refreshError ? (
        <Banner
          tone="warn"
          className="mb-4"
          title={t(mr.refreshFailed)}
          actions={
            <Button size="sm" onClick={onRetry}>
              {t(common.retry)}
            </Button>
          }
        >
          {errorMessage(refreshError)} {t(mr.refreshFailedBody)}
        </Banner>
      ) : null}
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
    </>
  )

  if (warningView) {
    // As ProvidersBulkWarning draws it: the unsaved change and its warning in
    // the header; Save routing leads to the fix first.
    const unsavedActions = (
      <>
        <Badge tone="warn" dot>
          {t(mr.unsavedWarnings, { changes, warnings: issues.length })}
        </Badge>
        <Button onClick={discard}>{t(mr.discard)}</Button>
        <Button variant="primary" loading={saving} disabledReason={reason} onClick={trySave}>
          {t(mr.saveRouting)}
        </Button>
      </>
    )
    return (
      <Page title={t(m.title)} filters={<ProvidersTabs />} actions={unsavedActions} contentClassName="max-w-[1120px]">
        {/* The bulk-terms warning is fixed before a save, so leaving offers no save here. */}
        {leaveGuard(false)}
        {notices}
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
      {leaveGuard(true)}
      {notices}
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-[2_1_600px] flex-col gap-4">
          {(['broadcast', 'automation', 'system', 'transactional'] as Stream[]).map((st) => (
            <StreamCard
              key={st}
              stream={st}
              cfg={cfg}
              sd={draft[st]}
              unsaved={!sameStream(draft, saved, st)}
              onChange={(fn) => update(st, fn)}
              onAddCondition={() => setConditionFor(st)}
              readOnlyReason={reason}
              announce={setAnnounce}
            />
          ))}
        </div>
        <aside className="flex max-w-[460px] min-w-0 flex-[1_1_360px] flex-col gap-4" aria-label={t(mr.asideLabel)}>
          <RouteTester cfg={cfg} draft={dirty ? draft : undefined} />
          <SharedCard cfg={cfg} />
        </aside>
      </div>
      <span className="vh" role="status" aria-live="polite">
        {announce}
      </span>
      {dirty ? (
        <UnsavedBar
          text={changeText(draft, saved, cfg.providers, impact, sim.data?.impact)}
          cantSave={problem ? t(mr.weightsNot100, { total: problem.total }) : undefined}
          saving={saving}
          readOnlyReason={reason}
          onDiscard={discard}
          onSave={trySave}
        />
      ) : null}
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

/** One stream's route: the split (Broadcasts) or primary and fallback, conditions, the failover trigger and its target's checks. */
function StreamCard({
  stream,
  cfg,
  sd,
  unsaved,
  onChange,
  onAddCondition,
  readOnlyReason,
  announce,
}: {
  stream: Stream
  cfg: RoutingConfig
  sd: StreamDraft
  unsaved: boolean
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  onAddCondition: () => void
  readOnlyReason?: string
  announce: (msg: string) => void
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
  const broadcast = stream === 'broadcast'
  const vet = view?.failoverVet
  return (
    <section className="card flex flex-col" aria-labelledby={titleId}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <div className="flex items-center gap-2.5 max-md:flex-wrap">
          <h2 className="h2" id={titleId}>
            {streamName[stream]}
          </h2>
          <Badge>{tag}</Badge>
          {unsaved ? (
            <Badge tone="warn" dot>
              {t(mr.unsavedBadge)}
            </Badge>
          ) : null}
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
        {broadcast ? (
          <SplitList sd={sd} byId={byId} providers={cfg.providers} stream={stream} onChange={onChange} readOnly={ro} announce={announce} />
        ) : (
          <div className="pv-fsel">
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
            <FallbackSelect stream={stream} cfg={cfg} sd={sd} onChange={onChange} readOnly={ro} />
          </div>
        )}
        {!broadcast && sd.splits.length > 1 ? <SplitList sd={sd} byId={byId} providers={cfg.providers} stream={stream} onChange={onChange} readOnly={ro} announce={announce} /> : null}
        {sd.conditions.length || broadcast ? (
          <Conditions sd={sd} cfg={cfg} onChange={onChange} readOnly={ro}>
            {broadcast ? (
              <>
                <Button size="sm" className="self-start" disabledReason={readOnlyReason} onClick={onAddCondition}>
                  {t(mr.addConditionBroadcast)}
                </Button>
                <span className="k">{t(mr.mxNote)}</span>
              </>
            ) : null}
          </Conditions>
        ) : null}
        {stream === 'transactional' ? (
          <TransactionalParts cfg={cfg} sd={sd} onChange={onChange} onAddCondition={onAddCondition} readOnlyReason={readOnlyReason} />
        ) : broadcast ? (
          <FailoverPolicy sd={sd} cfg={cfg} onChange={onChange} readOnly={ro} vet={vet} />
        ) : (
          <>
            <div className="flex flex-wrap items-start gap-2.5">
              <Badge tone={sd.failover === 'pause_and_ask' ? 'warn' : 'ok'} dot>
                {t(mr.failoverBadge, { policy: sd.failover })}
              </Badge>
              <span className="k flex-[1_1_300px]">
                {stream === 'system'
                  ? t(mr.systemNote)
                  : sd.failover === 'pause_and_ask'
                    ? t(mr.pausesWhenInline)
                    : t(mr.failsOverWhenInline, { minutes: 10 })}
              </span>
            </div>
            {vet ? <TargetChecks vet={vet} stream={stream} primary={primary?.name} /> : null}
            {stream === 'automation' ? (
              <Button size="sm" className="self-start" disabledReason={readOnlyReason} onClick={onAddCondition}>
                {t(mr.addCondition)}
              </Button>
            ) : null}
          </>
        )}
        {stream === 'transactional' && vet ? <TargetChecks vet={vet} stream={stream} primary={primary?.name} domain={view?.senderDomains?.join(', ')} /> : null}
      </div>
    </section>
  )
}

/** The fallback provider of a stream. */
function FallbackSelect({
  stream,
  cfg,
  sd,
  onChange,
  readOnly,
}: {
  stream: Stream
  cfg: RoutingConfig
  sd: StreamDraft
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  readOnly: boolean
}) {
  const t = useT()
  const primary = sd.splits[0]?.providerId
  return (
    <Field label={t(mr.fallback)}>
      <Select value={sd.fallback} disabled={readOnly} onChange={(e) => onChange((s) => ({ ...s, fallback: e.target.value }))}>
        <option value="">{stream === 'broadcast' ? t(mr.noneAsk) : t(mr.none)}</option>
        {cfg.providers
          .filter((p) => p.id !== primary)
          .map((p) => (
            <option key={p.id} value={p.id}>
              {providerOption(p, { warming: true })}
            </option>
          ))}
      </Select>
    </Field>
  )
}

/**
 * The weighted split as an ordered list (PR-14): drag a row or use its
 * arrows; the order only changes how the bar reads, never who goes where
 * (rendezvous hashing). Weights must add up to 100% before a save.
 */
function SplitList({
  sd,
  byId,
  providers,
  stream,
  onChange,
  readOnly,
  announce,
}: {
  sd: StreamDraft
  byId: Map<string, RoutingProvider>
  providers: RoutingProvider[]
  stream: Stream
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  readOnly: boolean
  announce: (msg: string) => void
}) {
  const t = useT()
  const [dragging, setDragging] = useState<number | null>(null)
  const [over, setOver] = useState<number | null>(null)
  const rows = sd.splits
  const total = splitTotal(sd)
  const ok = total === 100
  const move = (from: number, to: number) => {
    if (to < 0 || to >= rows.length || from === to) return
    const p = byId.get(rows[from]!.providerId)
    onChange((x) => moveSplit(x, from, to))
    announce(t(mr.movedTo, { provider: p?.name ?? t(mr.unknown), position: to + 1, count: rows.length }))
  }
  const addable = providers.filter((p) => !rows.some((r) => r.providerId === p.id))
  const sum = total || 1
  const warming = rows.map((x) => byId.get(x.providerId)).find((p) => p?.warmingUp && p.shareSteps.length)
  const listId = useId()
  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <h3 className="m-0 text-[13px] font-semibold" id={listId}>
          {t(mr.weightedSplit)}
        </h3>
        <span className="k">{t(mr.stickyLine)}</span>
      </div>
      {/* The bar only pictures the list below, which says the same in words. */}
      <div className="pv-sbar" aria-hidden="true">
        {rows.map((x) => {
          const p = byId.get(x.providerId)
          if (!p || x.weight <= 0) return null
          return <i key={x.providerId} style={{ width: `${(x.weight / sum) * 100}%`, background: providerColor[p.color] }} />
        })}
      </div>
      <ol className="pv-wl" aria-labelledby={listId}>
        {rows.map((x, i) => {
          const p = byId.get(x.providerId)
          const name = p?.name ?? t(mr.unknown)
          return (
            <li
              key={x.providerId}
              className={cn(dragging === i && 'dragging', over === i && dragging !== null && dragging !== i && 'over')}
              draggable={!readOnly}
              onDragStart={(e) => {
                setDragging(i)
                e.dataTransfer.effectAllowed = 'move'
                e.dataTransfer.setData('text/plain', String(i))
              }}
              onDragOver={(e) => {
                if (dragging === null) return
                e.preventDefault()
                setOver(i)
              }}
              onDragLeave={() => setOver((o) => (o === i ? null : o))}
              onDrop={(e) => {
                e.preventDefault()
                if (dragging !== null) move(dragging, i)
                setDragging(null)
                setOver(null)
              }}
              onDragEnd={() => {
                setDragging(null)
                setOver(null)
              }}
            >
              <svg className="pv-grip" width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                <circle cx="6" cy="4" r="1.2" />
                <circle cx="10" cy="4" r="1.2" />
                <circle cx="6" cy="8" r="1.2" />
                <circle cx="10" cy="8" r="1.2" />
                <circle cx="6" cy="12" r="1.2" />
                <circle cx="10" cy="12" r="1.2" />
              </svg>
              <span className="dot flex-none" style={{ color: p ? providerColor[p.color] : undefined }} aria-hidden="true" />
              <span className="pv-wn">
                <span>{name}</span>
                {p ? <span className="k">{providerNote(p)}</span> : null}
              </span>
              <WeightInput name={name} stream={stream} value={x.weight} readOnly={readOnly} onChange={(v) => onChange((y) => setWeight(y, x.providerId, v))} />
              <span className="pv-mv">
                <button type="button" className="pv-ib" aria-label={t(mr.moveUp, { provider: name })} disabled={readOnly || i === 0} onClick={() => move(i, i - 1)}>
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M8 13V3M4 7l4-4 4 4" />
                  </svg>
                </button>
                <button type="button" className="pv-ib" aria-label={t(mr.moveDown, { provider: name })} disabled={readOnly || i === rows.length - 1} onClick={() => move(i, i + 1)}>
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="M8 3v10M4 9l4 4 4-4" />
                  </svg>
                </button>
                {/* A provider set to 0% can leave the split. */}
                {rows.length > 1 && x.weight === 0 ? (
                  <button type="button" className="pv-ib" aria-label={t(mr.removeFromSplit, { provider: name })} disabled={readOnly} onClick={() => onChange((y) => removeFromSplit(y, x.providerId))}>
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
                      <path d="M4 4l8 8M12 4l-8 8" />
                    </svg>
                  </button>
                ) : null}
              </span>
            </li>
          )
        })}
      </ol>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <span aria-live="polite">
          <Badge tone={ok ? 'ok' : 'bad'} dot>
            {ok ? t(mr.totalOk) : t(mr.totalBad, { total: trimPct(total).replace('%', '') })}
          </Badge>
        </span>
        {readOnly ? null : addable.length ? (
          <Menu trigger={<Button size="sm">{t(mr.addToSplit)}</Button>} align="start" label={t(mr.addToSplit)}>
            {addable.map((p) => (
              <MenuItem key={p.id} onSelect={() => onChange((y) => addToSplit(y, p.id))}>
                {providerOption(p, { warming: true })}
              </MenuItem>
            ))}
          </Menu>
        ) : (
          <Button size="sm" disabledReason={t(mr.allInSplit)}>
            {t(mr.addToSplit)}
          </Button>
        )}
      </div>
      <span className="k">
        {t(mr.dragHint)}
        {warming
          ? ` ${t(mr.plannerRaisesProvider, { provider: warming.short, steps: warming.shareSteps.map((st) => t(mr.plannerStep, { share: trimPct(st.share), day: st.day })).join(', ') })}`
          : ''}
      </span>
    </div>
  )
}

/** "us-east-1 · warm", "mta.acme.io · warming up, day 9 of 21". */
function providerNote(p: RoutingProvider): string {
  const label = p.label.replace(/^self-hosted · /, '')
  const state = p.warmingUp ? (p.warmupDay ? tt(mr.noteWarmingDay, { day: p.warmupDay, days: p.warmupDays ?? 21 }) : tt(mr.noteWarming)) : tt(mr.noteWarm)
  return label ? `${label} · ${state}` : state
}

function WeightInput({ name, stream, value, onChange, readOnly }: { name: string; stream: Stream; value: number; onChange: (v: number) => void; readOnly: boolean }) {
  const t = useT()
  // While typing, the field shows what was typed; otherwise the draft's weight.
  const [typing, setTyping] = useState<string | null>(null)
  return (
    <label className="flex items-center gap-1.5">
      <input
        className="in w-[64px] text-right"
        inputMode="numeric"
        value={typing ?? String(value)}
        disabled={readOnly}
        aria-label={t(mr.weightLabel, { provider: name, stream: streamName[stream] })}
        onFocus={() => setTyping(String(value))}
        onChange={(e) => {
          setTyping(e.target.value)
          const n = Number(e.target.value.replace(/[^0-9]/g, ''))
          if (e.target.value.trim() !== '' && !Number.isNaN(n)) onChange(n)
        }}
        onBlur={() => setTyping(null)}
      />
      <span aria-hidden="true">%</span>
    </label>
  )
}

/** "Fail over when …": the trigger the provider health check really uses. */
function FailoverTrigger({ automatic }: { automatic: boolean }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[13px]">
        <span className="font-semibold">{t(mr.failWhen)}</span> {t(mr.failWhenRule)}
        {automatic ? ` ${t(mr.returnsAfter, { minutes: 10 })}` : null}
      </span>
      <span className="k">{t(mr.silentNoFailover)}</span>
    </div>
  )
}

/** One check of a failover target, as its line says it. */
function checkText(c: Schemas['RoutingVetCheck'], vet: Schemas['RoutingTargetVet']): string {
  const result = c.result === 'warn' ? 'fail' : c.result
  switch (c.key) {
    case 'domain':
      return tt(mr.checkDomain, { result })
    case 'quota': {
      // No quota of its own: a warming provider still has today's cap.
      const warm = vet.checks.find((x) => x.key === 'warm')
      const shape = c.left !== undefined ? 'left' : warm?.left !== undefined ? 'today' : 'none'
      return tt(mr.checkQuota, { result, shape, left: count(c.left ?? warm?.left ?? 0), period: c.period ?? 'none' })
    }
    case 'warm':
      return tt(mr.checkWarm, { result, hasDay: vet.warmupDay ? 'yes' : 'no', day: vet.warmupDay ?? 0, days: vet.warmupDays ?? 0 })
    case 'sandbox':
      return tt(mr.checkSandbox, { result })
    case 'available':
      return tt(mr.checkAvailable, { result })
    case 'bulk_terms':
      return tt(mr.checkBulk, { result })
  }
  return c.key
}

/** The order the board lists a target's checks in. */
const checkOrder = ['available', 'bulk_terms', 'domain', 'quota', 'warm', 'sandbox']

/** "Checked before any failover · Postal": the target's checks, each with a mark. */
function TargetChecks({ vet, stream, primary, domain }: { vet: Schemas['RoutingTargetVet']; stream: Stream; primary?: string; domain?: string }) {
  const t = useT()
  const id = useId()
  const checks = [...vet.checks].sort((a, b) => checkOrder.indexOf(a.key) - checkOrder.indexOf(b.key))
  return (
    <div className="flex flex-col gap-1.5">
      <span className="k font-semibold" id={id}>
        {stream === 'broadcast'
          ? t(mr.checkedBeforeAny, { provider: vet.provider })
          : domain
            ? t(mr.checkedBeforeTakesOn, { provider: vet.provider, domain })
            : t(mr.checkedBeforeTakes, { provider: vet.provider })}
      </span>
      <ul className="pv-chk" aria-labelledby={id}>
        {checks.map((c) => {
          const pass = c.result === 'pass'
          return (
            <li key={c.key} className={pass ? 'y' : c.result === 'unknown' ? 'u' : 'x'}>
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                {pass ? <path d="m3.5 8.5 3 3 6-7" /> : c.result === 'unknown' ? <path d="M4 8h8" /> : <path d="m4.5 4.5 7 7M11.5 4.5l-7 7" />}
              </svg>
              {checkText(c, vet)}
            </li>
          )
        })}
      </ul>
      {stream === 'system' ? null : (
        <span className="k">{stream === 'broadcast' ? t(mr.checksFailBroadcast) : t(mr.checksFailOther, { provider: primary ?? '' })}</span>
      )}
    </div>
  )
}

/** The sticky unsaved-changes bar: what saving changes, Discard and Save. */
function UnsavedBar({
  text,
  cantSave,
  saving,
  readOnlyReason,
  onDiscard,
  onSave,
}: {
  text: string
  cantSave?: string
  saving: boolean
  readOnlyReason?: string
  onDiscard: () => void
  onSave: () => void
}) {
  const t = useT()
  return (
    <div className="pv-unsaved" role="region" aria-label={t(mr.unsavedTitle)}>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-0.5">
        <span className="font-semibold">{t(mr.unsavedTitle)}</span>
        <span className="text-[13px] text-ink-2" aria-live="polite">
          {text}
        </span>
      </div>
      <div className="flex flex-wrap gap-2.5">
        <Button onClick={onDiscard}>{t(mr.discard)}</Button>
        <Button variant="primary" loading={saving} disabledReason={readOnlyReason ?? cantSave} onClick={onSave}>
          {t(mr.saveRouting)}
        </Button>
      </div>
    </div>
  )
}

/** Who moves with the draft: the count, from the provider that loses most to the one that gains most. */
function movedAcross(r: Schemas['RouteCampaignResult'], cfg: RoutingConfig): { moved: number; from?: string; toward?: string } {
  const d = r.draft
  if (!d) return { moved: 0 }
  const before = new Map(r.shares.map((x) => [x.providerId, x.count]))
  let gain: Schemas['RouteCampaignShare'] | undefined
  let loss: Schemas['RouteCampaignShare'] | undefined
  for (const x of d.shares) {
    const diff = x.count - (before.get(x.providerId) ?? 0)
    if (diff > 0 && (!gain || diff > gain.count - (before.get(gain.providerId) ?? 0))) gain = x
  }
  for (const x of r.shares) {
    const after = d.shares.find((y) => y.providerId === x.providerId)?.count ?? 0
    if (x.count - after > 0 && (!loss || x.count - after > loss.count - (d.shares.find((y) => y.providerId === loss!.providerId)?.count ?? 0))) loss = x
  }
  const short = (id?: string) => cfg.providers.find((p) => p.id === id)?.short
  return { moved: d.moved, from: short(loss?.providerId), toward: short(gain?.providerId) }
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
        <span className="k flex-[1_1_300px]">{t(mr.txConditions, { count: sd.conditions.length, auto: sd.failover === 'pause_and_ask' ? 'no' : 'yes' })}</span>
        <span className="ml-auto flex flex-wrap gap-2">
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
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] font-semibold">{sd.splits.length > 1 ? t(mr.conditionsBeforeSplit) : t(mr.conditionsBeforePrimary)}</span>
      {sd.conditions.map((c, i) => {
        const values = valueNames(c, cfg)
        const split = (c.splits ?? []).filter((x) => x.weight > 0)
        const p = byId.get(c.providerId ?? split[0]?.providerId ?? '')
        // A split by weight shows each provider's dot, as the bar does.
        const then: ReactNode =
          split.length > 1
            ? split.map((x, j) => {
                const sp = byId.get(x.providerId)
                return (
                  <span key={x.providerId} className="inline-flex items-center gap-1.5">
                    {j > 0 ? <span aria-hidden="true">·</span> : null}
                    {sp ? <span className="dot" style={{ color: providerColor[sp.color] }} aria-hidden="true" /> : null}
                    {t(mr.condShare, { provider: sp?.name ?? t(mr.condUnknownProvider), share: trimPct(x.weight) })}
                  </span>
                )
              })
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
            <span className={cn('pv-chip then', split.length > 1 && 'gap-1.5')}>{then}</span>
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

/** Broadcasts failover policy: the trigger, the fallback, what happens when it fires, and the target's checks. */
function FailoverPolicy({
  sd,
  cfg,
  onChange,
  readOnly,
  vet,
}: {
  sd: StreamDraft
  cfg: RoutingConfig
  onChange: (fn: (sd: StreamDraft) => StreamDraft) => void
  readOnly: boolean
  vet?: Schemas['RoutingTargetVet']
}) {
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
    <div className="flex flex-col gap-3 rounded-[8px] border border-line bg-ground-3 px-3.5 py-3">
      <span className="text-[13px] font-semibold">{t(mr.failoverPolicy)}</span>
      <FailoverTrigger automatic={sd.failover === 'automatic'} />
      <FallbackSelect stream="broadcast" cfg={cfg} sd={sd} onChange={onChange} readOnly={readOnly} />
      <div className="flex flex-col gap-2" role="radiogroup" aria-labelledby={`${name}-t`}>
        <span className="text-[13px] font-semibold" id={`${name}-t`}>
          {t(mr.whenTriggerFires)}
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
      {vet ? <TargetChecks vet={vet} stream="broadcast" /> : null}
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
        cfg.shared.syncedAt
          ? t(mr.sharedSuppressionSynced, { count: count(cfg.shared.suppressions), pushed: count(cfg.shared.pushed), time: clock(cfg.shared.syncedAt) })
          : t(mr.sharedSuppressionPushed, { count: count(cfg.shared.suppressions), pushed: count(cfg.shared.pushed) }),
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
