import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Field, Input, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ago, clock, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { onProviderColor, perSec, providerColor, ProvidersTabs, streamName, throwIfForbidden, type FailoverPolicy, type Stream } from './common'
import { BulkWarningView } from './BulkWarning'
import { RouteTester } from './RouteTester'
import {
  bulkIssues,
  changedStreams,
  fieldText,
  mailboxName,
  providerOption,
  sameDraft,
  setPrimary,
  setShare,
  toDraft,
  toInput,
  type Condition,
  type Draft,
  type RoutingConfig,
  type RoutingProvider,
  type StreamDraft,
} from './routing-model'

const routingKey = ['get', '/api/v1/workspaces/{workspace}/routing'] as const
const providersKey = ['get', '/api/v1/workspaces/{workspace}/providers'] as const

/** 7.3 Routing & failover: per-stream primary, split, conditions, fallback and failover policy. */
export function RoutingPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  if (q.isError || q.isPending) {
    return (
      <Page title="Sending providers & routing" filters={<ProvidersTabs />}>
        {q.isError ? (
          <ErrorPanel title="Couldn't load the routing" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)} Mail keeps going out on the saved routing.
          </ErrorPanel>
        ) : (
          <RoutingSkeleton />
        )}
      </Page>
    )
  }
  return <RoutingEditor key={q.data.savedAt ?? 'never'} cfg={q.data} />
}

function RoutingEditor({ cfg }: { cfg: RoutingConfig }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('providers:manage')
  const [saved] = useState(() => toDraft(cfg))
  const [draft, setDraft] = useState<Draft>(saved)
  const [conditionFor, setConditionFor] = useState<Stream | null>(null)
  const [saving, setSaving] = useState(false)
  const dirty = !sameDraft(draft, saved)
  const issues = bulkIssues(draft, cfg.providers)
  const warningView = dirty && issues.length > 0
  const changes = changedStreams(draft, saved).length
  const update = (s: Stream, fn: (sd: StreamDraft) => StreamDraft) => setDraft((d) => ({ ...d, [s]: fn(d[s]) }))

  const save = async (next: Draft, acknowledge = false) => {
    setSaving(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/routing', { params: { path: { workspace: slug } }, body: toInput(next, acknowledge) })
      toast({ kind: 'success', title: 'Routing saved', detail: 'New sends use it right away; sends in progress keep their route.' })
      await Promise.all([qc.invalidateQueries({ queryKey: routingKey }), qc.invalidateQueries({ queryKey: providersKey })])
    } catch (e) {
      const detail = e instanceof ApiError && e.code === 'bulk_terms' ? e.message : errorMessage(e)
      toast({ kind: 'error', title: "Routing wasn't saved", detail })
    } finally {
      setSaving(false)
    }
  }

  const actions = (
    <>
      {dirty ? (
        <Badge tone="warn" dot>
          Unsaved change{changes > 1 ? 's' : ''}
          {issues.length ? ` · ${issues.length} warning${issues.length > 1 ? 's' : ''}` : ''}
        </Badge>
      ) : cfg.savedAt ? (
        <span className="k">
          Saved {ago(cfg.savedAt)}
          {cfg.savedBy ? ` by ${cfg.savedBy}` : ''}
        </span>
      ) : null}
      {dirty ? <Button onClick={() => setDraft(saved)}>Discard</Button> : null}
      <Button
        variant="primary"
        disabledReason={reason}
        loading={saving}
        onClick={() => {
          if (!dirty) {
            toast({ kind: 'info', title: 'Routing is up to date', detail: 'Change a route, a split or the failover policy first.' })
            return
          }
          if (warningView) {
            document.getElementById('pv-fix')?.scrollIntoView({ block: 'center' })
            return
          }
          void save(draft)
        }}
      >
        Save routing
      </Button>
    </>
  )

  if (warningView) {
    return (
      <Page title="Sending providers & routing" filters={<ProvidersTabs />} actions={actions} contentClassName="max-w-[1120px]">
        <BulkWarningView
          cfg={cfg}
          draft={draft}
          saved={saved}
          issues={issues}
          saving={saving}
          onDraft={setDraft}
          onSave={(next, acknowledge) => save(next, acknowledge)}
          readOnlyReason={reason}
        />
      </Page>
    )
  }

  return (
    <Page title="Sending providers & routing" filters={<ProvidersTabs />} actions={actions}>
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
        <aside className="flex max-w-[460px] min-w-0 flex-[1_1_360px] flex-col gap-4" aria-label="Route tester and shared settings">
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

const streamTag: Record<Stream, string> = {
  broadcast: 'campaigns',
  automation: 'emails sent by flows',
  system: 'opt-in confirmations, preference links, invites',
  transactional: 'password resets, receipts and alerts from your product',
}

function volumeText(v?: Schemas['RoutingVolume']) {
  return v ? `about ${count(v.count)} per ${v.per}` : 'no mail yet'
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
  const view = cfg.streams.find((s) => s.stream === stream)
  const { slug, me } = useWorkspace()
  // Transactional mail is "from Acme Cloud": the product name in its settings.
  const settings = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/transactional/settings',
    { params: { path: { workspace: slug } } },
    { enabled: stream === 'transactional' && me.permissions.includes('transactional:view') },
  )
  const tag = stream === 'transactional' && settings.data?.productName ? `password resets, receipts and alerts from ${settings.data.productName}` : streamTag[stream]
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const primary = sd.splits[0] ? byId.get(sd.splits[0].providerId) : undefined
  const ro = Boolean(readOnlyReason)
  const titleId = `pv-stream-${stream}`
  const addBroadcastCondition = (
    <Button size="sm" className="self-start" disabledReason={readOnlyReason} onClick={onAddCondition}>
      Add condition · list, sender domain or mailbox provider
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
          {volumeText(view?.volume)}
          {stream === 'transactional' && view?.senderDomains?.length ? ` · ${view.senderDomains.join(', ')}` : ''}
        </span>
      </div>
      <div className="flex flex-col gap-3.5 px-5 py-[18px]">
        <Grid min={240} gap={14}>
          <Field label="Primary">
            <Select value={primary?.id ?? ''} disabled={ro} onChange={(e) => onChange((s) => setPrimary(s, e.target.value))}>
              {!primary ? <option value="">Choose a provider</option> : null}
              {cfg.providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {providerOption(p)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Fallback">
            <Select value={sd.fallback} disabled={ro} onChange={(e) => onChange((s) => ({ ...s, fallback: e.target.value }))}>
              <option value="">{stream === 'broadcast' ? 'None · pause and ask' : 'None'}</option>
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
                Failover: {sd.failover === 'pause_and_ask' ? 'pause and ask' : sd.failover === 'warm_only' ? 'warmed-up only' : 'automatic'}
              </Badge>
              <span className="k">
                {stream === 'system'
                  ? 'System mail must always go out; it never waits for a decision'
                  : 'switches after 3 consecutive failed sends or 2 failed health checks; returns to the primary after 10 healthy minutes'}
              </span>
            </div>
            {stream === 'automation' ? (
              <Button size="sm" className="self-start" disabledReason={readOnlyReason} onClick={onAddCondition}>
                Add condition
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
            First in line on shared providers
          </span>
          <span className="k">
            {shared
              ? `When ${shared.short} also sends a campaign, transactional messages go first${rate ? ` within its ${perSec(rate)} per second` : ''}`
              : 'When a provider also sends a campaign, transactional messages go first within its rate'}
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
          Failover: {sd.failover === 'pause_and_ask' ? 'pause and ask' : sd.failover === 'warm_only' ? 'warmed-up only' : 'automatic'}
        </Badge>
      </div>
      <span className="k">
        Conditions by category or mailbox provider: {sd.conditions.length ? sd.conditions.length : 'none'} · bulk-marketing terms don&apos;t apply to this stream
      </span>
      <div className="flex flex-wrap items-center justify-end gap-2.5">
        <span className="flex flex-wrap gap-2">
          <Button size="sm" disabledReason={readOnlyReason} onClick={onAddCondition}>
            Add condition
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/transactional" params={{ workspace: slug }}>
            Templates and log
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
  const warming = sd.splits.map((s) => byId.get(s.providerId)).find((p) => p?.warmingUp)
  const total = sd.splits.reduce((n, s) => n + s.weight, 0) || 1
  const label = sd.splits.map((s) => `${byId.get(s.providerId)?.short ?? 'Unknown'} ${s.weight}%`).join(', ')
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between max-md:flex-wrap">
        <span className="text-[13px] font-semibold">Weighted split</span>
        <span className="k">used for migration and warm-up{warming ? ` · adjusts daily with the ${warming.short} plan` : ''}</span>
      </div>
      <div className="flex h-7 gap-0.5 overflow-hidden rounded-[8px]" role="img" aria-label={`Weighted split: ${label}`}>
        {sd.splits.map((s, i) => {
          const p = byId.get(s.providerId)
          if (!p || s.weight <= 0) return null
          return (
            <div
              key={s.providerId}
              className="flex items-center overflow-hidden text-[12.5px] font-semibold whitespace-nowrap"
              style={{ width: `${(s.weight / total) * 100}%`, background: providerColor[p.color], color: onProviderColor[p.color], padding: i === 0 ? '0 10px' : '0 8px' }}
            >
              {i === 0 ? `${p.name} · ${s.weight}%` : `${p.short} ${s.weight}%`}
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
              <span className="k">
                The warm-up planner raises this automatically: {p.shareSteps.map((st) => `${st.share}% on day ${st.day}`).join(', ')}.
              </span>
            ) : null}
          </div>
        )
      })}
    </div>
  )
}

function ShareInput({ provider, value, onChange, readOnly }: { provider: RoutingProvider; value: number; onChange: (v: number) => void; readOnly: boolean }) {
  // While typing, the field shows what was typed; otherwise the draft's share.
  const [typing, setTyping] = useState<string | null>(null)
  return (
    <label className="lbl flex-row items-center gap-2">
      {provider.short} share
      <input
        className="in w-[70px]"
        inputMode="numeric"
        value={typing ?? String(value)}
        disabled={readOnly}
        aria-label={`${provider.short} percentage`}
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
  const byId = new Map(cfg.providers.map((p) => [p.id, p]))
  const lists = new Map(cfg.lists.map((l) => [l.id, l.name]))
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] font-semibold">Conditions · evaluated before the {sd.splits.length > 1 ? 'split' : 'primary'}</span>
      {sd.conditions.map((c, i) => {
        const values = c.field === 'list' ? c.values.map((v) => lists.get(v) ?? v) : c.field === 'mailbox_provider' ? c.values.map((v) => mailboxName[v] ?? v) : c.values
        const p = byId.get(c.providerId)
        return (
          <div key={i} className="pv-cond">
            <span className="k">IF</span>
            <span className="pv-chip">{fieldText[c.field]}</span>
            <span className="pv-chip">{values.length > 1 ? 'is one of' : 'is'}</span>
            <span className="pv-chip">{values.join(', ')}</span>
            <span className="k">THEN</span>
            <span className="pv-chip then">{p ? `${p.name} only` : 'Unknown provider'}</span>
            {c.note ? <span className="k ml-auto">{c.note}</span> : <span className="ml-auto" />}
            <Button
              size="sm"
              aria-label={`Remove condition: ${fieldText[c.field]} ${values.join(', ')}`}
              disabled={readOnly}
              onClick={() => onChange((s) => ({ ...s, conditions: s.conditions.filter((_, j) => j !== i) }))}
            >
              Remove
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
  const name = useId()
  const warming = cfg.providers.find((p) => p.warmingUp)
  const opts: { value: FailoverPolicy; title: ReactNode; detail?: ReactNode }[] = [
    {
      value: 'pause_and_ask',
      title: (
        <>
          <span className="font-semibold">Pause and ask</span> <span className="k">(default for Broadcasts)</span>
        </>
      ),
      detail: 'If the primary fails mid-send, hold the remaining recipients and notify you. Moving a bulk send to a cold provider can hurt reputation.',
    },
    {
      value: 'warm_only',
      title: <span className="font-semibold">Fail over automatically, only to a warmed-up provider</span>,
      detail: warming
        ? `${warming.short} qualifies on day ${warming.warmupDays ?? 21} of its warm-up. Until then this behaves like pause and ask.`
        : 'Providers still warming up are skipped; with none left, this behaves like pause and ask.',
    },
    { value: 'automatic', title: <span className="font-semibold">Fail over automatically to any fallback</span> },
  ]
  return (
    <div className="flex flex-col gap-2 rounded-[8px] border border-line bg-ground-3 px-3.5 py-3" role="radiogroup" aria-labelledby={`${name}-t`}>
      <span className="text-[13px] font-semibold" id={`${name}-t`}>
        Failover policy
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
  const { slug } = useWorkspace()
  const row = (k: string, v: string) => (
    <div className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
      <span>{k}</span>
      <span className="n">{v}</span>
    </div>
  )
  return (
    <section className="card flex flex-col gap-2.5 px-5 py-[18px]">
      <h2 className="h2">Shared across providers</h2>
      {row('Suppression list', `${count(cfg.shared.suppressions)}${cfg.shared.syncedAt ? ` · synced ${clock(cfg.shared.syncedAt)}` : ''}`)}
      {row('Per-mailbox-provider throttles', 'apply on every route')}
      {row('Auto-pause thresholds', 'per campaign')}
      <Link to="/w/$workspace/providers/events" params={{ workspace: slug }} className="text-[13px] font-medium">
        Open events &amp; sync
      </Link>
    </section>
  )
}

/** Adds a condition: when a field matches, send only through one provider. */
function ConditionDialog({ stream, cfg, onClose, onAdd }: { stream: Stream; cfg: RoutingConfig; onClose: () => void; onAdd: (c: Condition) => void }) {
  const [field, setField] = useState<Condition['field']>(stream === 'transactional' ? 'category' : 'mailbox_provider')
  const [values, setValues] = useState<string[]>([])
  const [providerId, setProviderId] = useState(cfg.providers.find((p) => !p.warmingUp)?.id ?? cfg.providers[0]?.id ?? '')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
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
      title={`Add a ${streamName[stream]} condition`}
      description="Conditions are checked in order before the split; the first match decides the provider."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button
            variant="primary"
            onClick={() => {
              if (!values.length) {
                setError('Pick at least one value.')
                return
              }
              const labels = values.map((v) => choices.find((c) => c.value === v)?.label ?? v)
              onAdd({ field, values, providerId, note: note.trim() || undefined, valueLabels: labels })
            }}
          >
            Add condition
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <Field label="If">
          <Select
            value={field}
            onChange={(e) => {
              setField(e.target.value as Condition['field'])
              setValues([])
              setError('')
            }}
          >
            {stream === 'transactional' ? <option value="category">Category</option> : null}
            <option value="mailbox_provider">Recipient mailbox provider</option>
            {stream === 'transactional' ? null : (
              <>
                <option value="list">List</option>
                <option value="sender_domain">Sender domain</option>
              </>
            )}
          </Select>
        </Field>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1 p-0">is one of</legend>
          {choices.length ? (
            choices.map((c) => <Checkbox key={c.value} label={c.label} checked={values.includes(c.value)} onChange={() => toggle(c.value)} />)
          ) : (
            <span className="k">Nothing to pick yet.</span>
          )}
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
        </fieldset>
        <Field label="Then send only through">
          <Select value={providerId} onChange={(e) => setProviderId(e.target.value)}>
            {cfg.providers.map((p) => (
              <option key={p.id} value={p.id}>
                {providerOption(p, { warming: true })}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Why" optional hint="Shown beside the condition, like “Outlook is strict with new IPs”.">
          <Input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
        </Field>
        {stream === 'broadcast' && cfg.providers.find((p) => p.id === providerId && !p.bulkAllowed) ? (
          <Note tone="warn">This provider&apos;s terms don&apos;t allow bulk marketing on Broadcasts.</Note>
        ) : null}
      </div>
    </Dialog>
  )
}

/** Loading: the stream cards and the tester, as skeletons. */
function RoutingSkeleton() {
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
    <Loading label="Loading the routing">
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
