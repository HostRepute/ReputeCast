import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { clock, count, dayMonthDate, pct, rate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import {
  durationText,
  emailPrefix,
  inEdges,
  nodeById,
  nodeRef,
  nodeTitle,
  outEdges,
  pathLabel,
  TRIGGER_KINDS,
  triggerKindLabel,
  typeColors,
  typeName,
  wholePct,
  type FlowCondition,
  type FlowDuration,
  type FlowGraph,
  type FlowNode,
  type NodeData,
  type Refs,
  type StepType,
  type TriggerConfig,
} from './graph'
import { mg } from './graph.messages'
import { mp } from './panels.messages'
import type { StepStats } from './view'
import { StepLanguages } from '@/features/languages/automation'
import { ScoreTriggerFields, scoreTriggerDefaults } from '@/features/scoring/ScoreTrigger'
import { isOneOfLabel } from '@/features/scoring/triggerText'
import { PollTriggerProps } from '@/features/polls/trigger'
import { BoughtConditionProps, BoughtTriggerProps } from '@/features/products/automation'
import { boughtConditionName } from '@/features/products/automationText'
import { MilestoneTriggerFields } from '@/features/grow/MilestoneTrigger'
import { useChannelSetUp } from '@/features/channels/hooks'
import { m as smsStep } from '@/features/sms/automation.messages'
import { SmsStepProps, SmsStepStats } from '@/features/sms/AutomationStep'
import { m as pushStep } from '@/features/push/automation.messages'
import { PushStepProps, PushStepStats } from '@/features/push/AutomationStep'
import { m as waStep } from '@/features/whatsapp/automation.messages'
import { WhatsAppStepProps, WhatsAppStepStats, WhatsAppStepSub } from '@/features/whatsapp/AutomationStep'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { isStoreTrigger, PlacedOrderFields, StoreTriggerFields, storeTriggerDefaults, useStoreTriggerOptions } from '@/features/commerce/automation'

type Options = Schemas['AutomationOptions']

export const DRAG_TYPE = 'application/x-reputecast-step'

/** The step type being dragged, readable during dragover (data is not). */
export function draggedStepType(types: readonly string[]): StepType | undefined {
  const t = types.find((x) => x.startsWith(`${DRAG_TYPE}-`))
  return t ? (t.slice(DRAG_TYPE.length + 1) as StepType) : undefined
}

/** The palette's Steps group; the goal item also makes exits. */
const stepItems: StepType[] = ['delay', 'branch', 'filter', 'split', 'webhook', 'list', 'goal']

export interface PaletteItem {
  type: StepType
  trigger?: TriggerConfig['kind']
}

const bold = (chunks: ReactNode[]) => <b className="n">{chunks}</b>
const mono = (chunks: ReactNode[]) => <span className="mono">{chunks}</span>

/** Steps to drag onto the canvas (or press to add after the selected step). */
export function Palette({ disabledReason, onAdd }: { disabledReason?: string; onAdd: (item: PaletteItem) => void }) {
  const tr = useT()
  const sms = useChannelSetUp('sms')
  // Push and WhatsApp (X22) join once they're set up, as SMS does.
  const push = useChannelSetUp('push')
  const whatsapp = useChannelSetUp('whatsapp')
  // Messages (phase 8): Send email, then each channel the workspace has set up.
  const messages = [
    { type: 'email' as const, label: typeName('email') },
    ...(sms ? [{ type: 'sms' as const, label: tr(smsStep.palette) }] : []),
    ...(push ? [{ type: 'push' as const, label: tr(pushStep.palette) }] : []),
    ...(whatsapp ? [{ type: 'whatsapp' as const, label: tr(waStep.palette) }] : []),
  ]
  const item = (key: string, label: string, color: string, value: PaletteItem) => (
    <button
      key={key}
      type="button"
      className="fe-pal"
      draggable={!disabledReason}
      aria-disabled={disabledReason ? true : undefined}
      title={disabledReason ?? tr(mp.addAfter, { step: label.toLowerCase() })}
      onDragStart={(e) => {
        e.dataTransfer.setData(DRAG_TYPE, JSON.stringify(value))
        e.dataTransfer.setData(`${DRAG_TYPE}-${value.type}`, '')
        e.dataTransfer.effectAllowed = 'move'
      }}
      onClick={() => !disabledReason && onAdd(value)}
    >
      <span className="sw" style={{ background: color }} aria-hidden="true" />
      {label}
    </button>
  )
  return (
    <>
      <div className="grp">{tr(mp.triggers)}</div>
      <div className="flex flex-col gap-1.5 px-3 pb-2">{TRIGGER_KINDS.map((kind) => item(kind, triggerKindLabel(kind), typeColors.trigger, { type: 'trigger', trigger: kind }))}</div>
      <div className="grp">{tr(smsStep.messages)}</div>
      <div className="flex flex-col gap-1.5 px-3 pb-2">{messages.map((s) => item(s.type, s.label, typeColors[s.type], { type: s.type }))}</div>
      <div className="grp">{tr(mp.steps)}</div>
      <div className="flex flex-col gap-1.5 px-3 pb-2">
        {stepItems.map((type) => item(type, type === 'goal' ? tr(mp.goalOrExit) : typeName(type), typeColors[type], { type }))}
      </div>
      {sms ? <p className="k m-0 px-3 pb-2">{tr(smsStep.paletteNote)}</p> : null}
    </>
  )
}

function stepSummary(tr: ReturnType<typeof useT>, n: FlowNode, st: StepStats | undefined): string {
  if (!st) return tr(mp.summaryNone)
  switch (n.type) {
    case 'trigger':
      return tr(mp.summaryEntered, { count: count(st.entered) })
    case 'email':
      return tr(mp.summaryEmail, { sent: count(st.sent), rate: pct(st.delivered ? (st.openedHuman / st.delivered) * 100 : 0) })
    case 'goal':
      return tr(mp.summaryReached, { count: count(st.goalReached) })
    case 'exit':
      return tr(mp.summaryExited, { count: count(st.exited) })
    case 'branch':
    case 'split':
      return st.paths.map((p) => `${pathLabel(p.path)} ${count(p.count)}`).join(' · ') || tr(mp.summaryEntered, { count: count(st.entered) })
    case 'webhook':
      return tr(mp.summaryCalls, { calls: count(st.entered), failed: count(st.failed) })
  }
  return tr(mp.summaryHere, { entered: count(st.entered), here: count(st.inFlowNow) })
}

/** Left panel › Stats: every step with its key number; a list view of the canvas. */
export function FlowStatsList({ graph, order, refs, stats, selected, onSelect }: { graph: FlowGraph; order: string[]; refs: Refs; stats: Map<string, StepStats>; selected?: string; onSelect: (id: string) => void }) {
  const tr = useT()
  return (
    <div className="flex flex-col">
      {order.map((id) => {
        const n = nodeById(graph, id)
        if (!n) return null
        return (
          <button key={id} type="button" className="fe-steprow" aria-current={selected === id ? 'true' : undefined} onClick={() => onSelect(id)}>
            <span className="overline" style={{ color: typeColors[n.type] }}>
              {typeName(n.type)}
            </span>
            <span className="font-semibold">{nodeTitle(graph, n, refs)}</span>
            <span className="k">{stepSummary(tr, n, stats.get(id))}</span>
          </button>
        )
      })}
    </div>
  )
}

export interface StepPanelProps {
  graph: FlowGraph
  node: FlowNode
  refs: Refs
  options?: Options
  stats?: StepStats
  hasStats: boolean
  editable: boolean
  editReason?: string
  automationId: string
  automationName: string
  routeLabel: string
  routePrimary: string[]
  issues: string[]
  onChange: (data: NodeData) => void
  /** Turns a goal into an exit and back. */
  onChangeType?: (type: StepType) => void
  onDuplicate: () => void
  onDelete: () => void
  onPreview: (emailId: string, version: number) => void
  onEditEmail: (emailId: string, version: number) => void
}

const subs: Record<StepType, MessageDescriptor> = {
  trigger: mp.subTrigger,
  delay: mp.subDelay,
  email: mp.subEmail,
  branch: mp.subBranch,
  filter: mp.subFilter,
  split: mp.subSplit,
  webhook: mp.subWebhook,
  list: mp.subList,
  goal: mp.subGoal,
  exit: mp.subExit,
  sms: mp.subSms,
  push: mp.subPush,
  whatsapp: mp.subWhatsapp,
}

export function Prop({ title, children, id }: { title: string; children: ReactNode; id?: string }) {
  return (
    <div className="fe-prop" role="group" aria-labelledby={id}>
      <span className="tt" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

function Check({ checked, onChange, disabled, children }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean; children: ReactNode }) {
  return (
    <label className="fe-chk">
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
      {children}
    </label>
  )
}

function DurationInput({ value, onChange, disabled, label }: { value?: FlowDuration; onChange: (d: FlowDuration) => void; disabled?: boolean; label: string }) {
  const tr = useT()
  const d = value ?? { amount: 1, unit: 'days' as const }
  return (
    <div className="flex gap-1.5">
      <input
        className="in w-20"
        type="number"
        min={1}
        aria-label={tr(mp.amountLabel, { label })}
        disabled={disabled}
        value={d.amount}
        onChange={(e) => onChange({ ...d, amount: Math.max(0, Number(e.target.value) || 0) })}
      />
      <select className="in flex-1" aria-label={tr(mp.unitLabel, { label })} disabled={disabled} value={d.unit} onChange={(e) => onChange({ ...d, unit: e.target.value as FlowDuration['unit'] })}>
        <option value="minutes">{tr(mp.unitMinutes)}</option>
        <option value="hours">{tr(mp.unitHours)}</option>
        <option value="days">{tr(mp.unitDays)}</option>
        <option value="weeks">{tr(mp.unitWeeks)}</option>
      </select>
    </div>
  )
}

function ConditionEditor({ graph, node, value, onChange, refs, options, disabled, withWindow, allowNone }: {
  graph: FlowGraph
  node: FlowNode
  value?: FlowCondition
  onChange: (c: FlowCondition | undefined) => void
  refs: Refs
  options?: Options
  disabled?: boolean
  withWindow?: boolean
  allowNone?: boolean
}) {
  const c = value
  const emails = graph.nodes.filter((n) => n.type === 'email' && n.id !== node.id)
  const tr = useT()
  const texts = graph.nodes.filter((n) => n.type === 'sms' && n.id !== node.id)
  const sms = useChannelSetUp('sms') || Boolean(c?.kind.startsWith('sms_'))
  const kinds: { value: FlowCondition['kind'] | ''; label: string }[] = [
    ...(allowNone ? [{ value: '' as const, label: tr(mp.condReaching) }] : []),
    { value: 'opened', label: tr(mp.condOpened) },
    { value: 'clicked', label: tr(mp.condClicked) },
    { value: 'tag', label: tr(mp.condTag) },
    { value: 'list', label: tr(mp.condList) },
    { value: 'event', label: tr(mp.condEvent) },
    { value: 'field', label: tr(mp.condField) },
    // Digital products (X19).
    { value: 'bought_product', label: boughtConditionName() },
    // Shopify and WooCommerce (design expansion, X20).
    { value: 'placed_order', label: tr(mp.condPlacedOrder) },
    // Phase 8: texts.
    ...(sms
      ? [
          { value: 'sms_consent' as const, label: tr(smsStep.condConsent) },
          { value: 'sms_clicked' as const, label: tr(smsStep.condClicked) },
          { value: 'sms_replied' as const, label: tr(smsStep.condReplied) },
        ]
      : []),
  ]
  const set = (patch: Partial<FlowCondition>) => onChange({ ...(c ?? { kind: 'opened' }), ...patch } as FlowCondition)
  return (
    <>
      <select
        className="in"
        aria-label={tr(mp.condition)}
        disabled={disabled}
        value={c?.kind ?? ''}
        onChange={(e) => {
          const k = e.target.value as FlowCondition['kind'] | ''
          if (!k) return onChange(undefined)
          onChange({ kind: k, ...(withWindow && (k === 'opened' || k === 'clicked' || k === 'sms_clicked' || k === 'sms_replied') ? { within: c?.within ?? { amount: 2, unit: 'days' } } : {}), humanOnly: k === 'opened' ? true : undefined })
        }}
      >
        {kinds.map((k) => (
          <option key={k.value} value={k.value}>
            {k.label}
          </option>
        ))}
      </select>
      {c && (c.kind === 'opened' || c.kind === 'clicked') ? (
        <>
          <select className="in" aria-label={tr(mp.whichEmail)} disabled={disabled} value={c.step ?? ''} onChange={(e) => set({ step: e.target.value || undefined })}>
            <option value="">{tr(mp.anyEmail)}</option>
            {emails.map((e) => (
              <option key={e.id} value={e.id}>
                {nodeRef(graph, e, refs)}
              </option>
            ))}
          </select>
          {c.kind === 'clicked' ? (
            <input
              className="in"
              aria-label={tr(mp.linkContains)}
              placeholder={tr(mp.linkContainsPlaceholder)}
              disabled={disabled}
              value={c.urlContains ?? ''}
              onChange={(e) => set({ urlContains: e.target.value || undefined })}
            />
          ) : (
            <Check checked={Boolean(c.humanOnly)} disabled={disabled} onChange={(v) => set({ humanOnly: v || undefined })}>
              {tr(mp.humanOnly)}
            </Check>
          )}
          {withWindow ? (
            <>
              <span className="k">{tr(mp.waitUpTo)}</span>
              <DurationInput label={tr(mp.window)} value={c.within} disabled={disabled} onChange={(d) => set({ within: d })} />
            </>
          ) : null}
        </>
      ) : null}
      {c?.kind === 'tag' ? <input className="in" aria-label={tr(mp.tag)} list="fe-tags" placeholder={tr(mp.tag)} disabled={disabled} value={c.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} /> : null}
      {c?.kind === 'list' ? (
        <select className="in" aria-label={tr(mp.list)} disabled={disabled} value={c.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
          <option value="">{tr(mp.pickList)}</option>
          {options?.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      ) : null}
      {c?.kind === 'sms_consent' ? (
        <select className="in" aria-label={tr(smsStep.consent)} disabled={disabled} value={c.listId ?? ''} onChange={(e) => set({ listId: e.target.value || undefined })}>
          <option value="">{tr(smsStep.anyList)}</option>
          {options?.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      ) : null}
      {c && (c.kind === 'sms_clicked' || c.kind === 'sms_replied') ? (
        <>
          <select className="in" aria-label={tr(smsStep.whichText)} disabled={disabled} value={c.step ?? ''} onChange={(e) => set({ step: e.target.value || undefined })}>
            <option value="">{tr(smsStep.anyText)}</option>
            {texts.map((n) => (
              <option key={n.id} value={n.id}>
                {nodeRef(graph, n, refs)}
              </option>
            ))}
          </select>
          {withWindow ? (
            <>
              <span className="k">{tr(smsStep.waitUpTo)}</span>
              <DurationInput label={tr(smsStep.window)} value={c.within} disabled={disabled} onChange={(d) => set({ within: d })} />
            </>
          ) : null}
        </>
      ) : null}
      {c?.kind === 'event' ? (
        <input className="in mono" aria-label={tr(mp.event)} list="fe-events" placeholder={'teammate.invited'} disabled={disabled} value={c.event ?? ''} onChange={(e) => set({ event: e.target.value })} />
      ) : null}
      {/* Bought a product (digital products, X19). */}
      {c?.kind === 'bought_product' ? <BoughtConditionProps c={c} disabled={disabled} set={set} products={options?.products} /> : null}
      {c?.kind === 'placed_order' ? <PlacedOrderFields c={c} set={set} disabled={disabled} /> : null}
      {c?.kind === 'field' ? (
        <>
          <input className="in" aria-label={tr(mp.field)} placeholder={tr(mp.fieldPlaceholder)} disabled={disabled} value={c.field ?? ''} onChange={(e) => set({ field: e.target.value })} />
          <select className="in" aria-label={tr(mp.comparison)} disabled={disabled} value={c.op ?? 'equals'} onChange={(e) => set({ op: e.target.value })}>
            <option value="equals">{tr(mp.opIs)}</option>
            <option value="not_equals">{tr(mp.opIsNot)}</option>
            <option value="contains">{tr(mp.opContains)}</option>
            <option value="is_one_of">{isOneOfLabel()}</option>
            <option value="is_set">{tr(mp.opIsSet)}</option>
            <option value="is_not_set">{tr(mp.opIsEmpty)}</option>
          </select>
          {c.op !== 'is_set' && c.op !== 'is_not_set' ? (
            <input className="in" aria-label={tr(mp.value)} placeholder={tr(mp.value)} disabled={disabled} value={c.value ?? ''} onChange={(e) => set({ value: e.target.value })} />
          ) : null}
        </>
      ) : null}
    </>
  )
}

/** The emails of the workspace as select options: "name · v5". */
function EmailSelect({ value, version, options, disabled, onPick, onCreate }: { value?: string; version?: number; options?: Options; disabled?: boolean; onPick: (id: string, version: number) => void; onCreate: () => void }) {
  const tr = useT()
  const current = options?.emails.find((e) => e.id === value)
  return (
    <select
      aria-label={tr(mp.email)}
      className="in"
      disabled={disabled}
      value={value ? `${value}@${version ?? current?.latestVersion ?? 1}` : ''}
      onChange={(e) => {
        if (e.target.value === 'new') return onCreate()
        const [id, v] = e.target.value.split('@')
        if (id) onPick(id, Number(v))
      }}
    >
      {!value ? <option value="">{tr(mp.pickEmail)}</option> : null}
      {current ? (
        <option value={`${current.id}@${version ?? current.latestVersion}`}>
          {current.name} · v{version ?? current.latestVersion}
        </option>
      ) : null}
      {current && version !== undefined && version < current.latestVersion ? (
        <option value={`${current.id}@${current.latestVersion}`}>{tr(mp.emailNewer, { name: current.name, version: current.latestVersion })}</option>
      ) : null}
      {options?.emails
        .filter((e) => e.id !== value)
        .map((e) => (
          <option key={e.id} value={`${e.id}@${e.latestVersion}`}>
            {e.name} · v{e.latestVersion}
          </option>
        ))}
      <option value="new">{tr(mp.createFromTemplate)}</option>
    </select>
  )
}

function emailHeader(graph: FlowGraph, node: FlowNode, refs: Refs): string {
  const name = node.data.email?.emailId ? (refs.emails.get(node.data.email.emailId)?.name ?? '') : ''
  return node.data.label || emailPrefix(name).replace(' · ', ' ') || nodeTitle(graph, node, refs)
}

/** Right panel › Step: the selected step's settings, rule by rule. */
export function StepPanel(p: StepPanelProps) {
  const { graph, node, refs, options, stats: st, editable } = p
  const tr = useT()
  const { slug } = useWorkspace()
  const d = node.data
  const disabled = !editable
  const set = (patch: Partial<NodeData>) => p.onChange({ ...d, ...patch })
  const qc = useQueryClient()
  const createEmail = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automation-emails')
  const position = graph.nodes.filter((n) => n.type === 'email').findIndex((n) => n.id === node.id) + 1
  const title =
    node.type === 'email'
      ? `${typeName('email')} · ${emailHeader(graph, node, refs)}`
      : `${typeName(node.type)} · ${node.type === 'trigger' ? triggerKindLabel(d.trigger?.kind ?? 'joined_list') : nodeRef(graph, node, refs)}`
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">{title}</span>
        <span className="k">{node.type === 'whatsapp' ? <WhatsAppStepSub step={d.whatsapp} fallback={tr(subs.whatsapp)} /> : tr(subs[node.type])}</span>
      </div>
      {p.issues.length > 0 ? (
        <div className="fe-prop">
          <Note tone="bad">{p.issues.join(' ')}</Note>
        </div>
      ) : null}
      {node.type === 'trigger' ? <TriggerProps {...p} /> : null}
      {node.type === 'delay' ? (
        <Prop title={tr(mp.wait)} id="p-wait">
          <DurationInput label={tr(mp.wait)} value={d.delay} disabled={disabled} onChange={(v) => set({ delay: v })} />
          <span className="k">{tr(mp.waitNote)}</span>
        </Prop>
      ) : null}
      {node.type === 'email' ? (
        <>
          <Prop title={tr(mp.email)} id="p-email">
            <EmailSelect
              value={d.email?.emailId}
              version={d.email?.emailVersion}
              options={options}
              disabled={disabled}
              onPick={(id, v) => {
                const e = options?.emails.find((x) => x.id === id)
                set({ email: { ...d.email, emailId: id, emailVersion: v, subject: d.email?.subject || e?.subject } })
              }}
              onCreate={() =>
                createEmail.mutate(
                  {
                    params: { path: { workspace: slug } },
                    body: { name: tr(mp.newEmailName, { position }), subject: d.email?.subject || tr(mp.newEmailSubject), automationId: p.automationId },
                  },
                  {
                    onSuccess: (e) => {
                      set({ email: { ...d.email, emailId: e.id, emailVersion: e.latestVersion, subject: d.email?.subject || e.subject } })
                      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/options'] })
                    },
                    onError: (err) => toast({ kind: 'error', title: tr(mp.createEmailFailed), detail: errorMessage(err) }),
                  },
                )
              }
            />
            <div className="flex gap-1.5">
              <Button
                className="mini-btn"
                disabledReason={!d.email?.emailId ? tr(mp.pickEmailFirst) : editable ? undefined : p.editReason}
                onClick={() => d.email?.emailId && p.onEditEmail(d.email.emailId, d.email.emailVersion ?? 1)}
              >
                {tr(mp.editInComposer)}
              </Button>
              <Button className="mini-btn" disabledReason={!d.email?.emailId ? tr(mp.pickEmailFirst) : undefined} onClick={() => d.email?.emailId && p.onPreview(d.email.emailId, d.email.emailVersion ?? 1)}>
                {tr(mp.preview)}
              </Button>
            </div>
          </Prop>
          {d.email?.emailId ? (
            <StepLanguages
              emailId={d.email.emailId}
              version={d.email.emailVersion ?? 1}
              disabledReason={editable ? undefined : p.editReason}
              onEditSource={() => d.email?.emailId && p.onEditEmail(d.email.emailId, d.email.emailVersion ?? 1)}
              onVersion={(v) => set({ email: { ...d.email, emailVersion: v } })}
            />
          ) : null}
          <Prop title={tr(mp.subject)} id="p-subject">
            <input aria-labelledby="p-subject" className="in" disabled={disabled} value={d.email?.subject ?? ''} onChange={(e) => set({ email: { ...d.email, subject: e.target.value } })} />
          </Prop>
          <Prop title={tr(mp.sendingRoute)} id="p-route">
            <select aria-labelledby="p-route" className="in" disabled={disabled} value={d.email?.route ?? ''} onChange={(e) => set({ email: { ...d.email, route: e.target.value || undefined } })}>
              <option value="">{tr(mp.routeDefault, { route: p.routeLabel })}</option>
              {options?.providers
                .filter((pr) => !p.routePrimary.includes(pr.id) || pr.id === d.email?.route)
                .map((pr) => (
                  <option key={pr.id} value={pr.id}>
                    {tr(mp.routeOverride, { provider: pr.short })}
                  </option>
                ))}
            </select>
            <span className="k">{tr(mp.failoverNote)}</span>
          </Prop>
          <Prop title={tr(mp.sendWindow)} id="p-window">
            <Check
              checked={Boolean(d.email?.sendWindow?.enabled)}
              disabled={disabled}
              onChange={(v) => set({ email: { ...d.email, sendWindow: { start: d.email?.sendWindow?.start ?? '07:00', end: d.email?.sendWindow?.end ?? '21:00', enabled: v } } })}
            >
              {tr(mp.sendWindowCheck, { start: d.email?.sendWindow?.start ?? '07:00', end: d.email?.sendWindow?.end ?? '21:00' })}
            </Check>
          </Prop>
          <SkipRules {...p} />
          {p.hasStats && st ? (
            <Prop title={tr(mp.liveStats)} id="p-stats">
              <div className="fe-stats">
                <span>
                  <T msg={mp.statSent} values={{ value: count(st.sent), b: bold }} />
                </span>
                <span>
                  <T msg={mp.statDelivered} values={{ value: pct(st.sent ? (st.delivered / st.sent) * 100 : undefined), b: bold }} />
                </span>
                <span>
                  <T msg={mp.statHumanOpens} values={{ value: pct(st.delivered ? (st.openedHuman / st.delivered) * 100 : undefined), b: bold }} />
                </span>
                <span>
                  <T msg={mp.statClicks} values={{ value: pct(st.delivered ? (st.clicked / st.delivered) * 100 : undefined), b: bold }} />
                </span>
                <span>
                  <T msg={mp.statBounces} values={{ value: pct(st.sent ? (st.bounced / st.sent) * 100 : undefined), b: bold }} />
                </span>
                <span>
                  <T msg={mp.statComplaints} values={{ value: rate(st.delivered ? (st.complained / st.delivered) * 100 : undefined), b: bold }} />
                </span>
              </div>
            </Prop>
          ) : null}
        </>
      ) : null}
      {node.type === 'branch' ? (
        <Prop title={tr(mp.condition)} id="p-cond">
          <ConditionEditor graph={graph} node={node} value={d.condition} refs={refs} options={options} disabled={disabled} withWindow onChange={(c) => set({ condition: c })} />
          <span className="k">{d.condition?.within ? tr(mp.branchNoteAfter, { duration: durationText(d.condition.within) }) : tr(mp.branchNote)}</span>
        </Prop>
      ) : null}
      {node.type === 'filter' ? (
        <Prop title={tr(mp.onlyContactsWho)} id="p-filter">
          <ConditionEditor graph={graph} node={node} value={d.condition} refs={refs} options={options} disabled={disabled} onChange={(c) => set({ condition: c })} />
          <span className="k">{tr(mp.othersLeave)}</span>
        </Prop>
      ) : null}
      {node.type === 'split' ? <SplitProps {...p} /> : null}
      {node.type === 'webhook' ? <WebhookProps {...p} /> : null}
      {node.type === 'list' ? <ListProps {...p} /> : null}
      {node.type === 'sms' ? <SmsStepProps data={d} lists={options?.lists ?? []} editable={editable} onChange={p.onChange} /> : null}
      {node.type === 'push' ? <PushStepProps data={d} automationId={p.automationId} editable={editable} onChange={p.onChange} /> : null}
      {node.type === 'whatsapp' ? <WhatsAppStepProps data={d} lists={options?.lists ?? []} editable={editable} onChange={p.onChange} /> : null}
      {(node.type === 'goal' || node.type === 'exit') && p.onChangeType ? (
        <Prop title={tr(mp.endsAs)} id="p-end">
          <select aria-labelledby="p-end" className="in" disabled={disabled} value={node.type} onChange={(e) => p.onChangeType?.(e.target.value as StepType)}>
            <option value="goal">{tr(mp.endGoal)}</option>
            <option value="exit">{tr(mp.endExit)}</option>
          </select>
        </Prop>
      ) : null}
      {node.type === 'goal' ? (
        <>
          <Prop title={tr(mp.goalName)} id="p-goalname">
            <input aria-labelledby="p-goalname" className="in" disabled={disabled} placeholder={tr(mp.goalNamePlaceholder)} value={d.label ?? ''} onChange={(e) => set({ label: e.target.value || undefined })} />
          </Prop>
          <Prop title={tr(mp.reachedWhen)} id="p-goal">
            <ConditionEditor graph={graph} node={node} value={d.condition} refs={refs} options={options} disabled={disabled} allowNone onChange={(c) => set({ condition: c })} />
            <span className="k">{tr(mp.goalNote)}</span>
          </Prop>
        </>
      ) : null}
      {node.type === 'exit' ? (
        <Prop title={tr(mp.name)} id="p-exitname">
          <input aria-labelledby="p-exitname" className="in" disabled={disabled} placeholder={tr(mg.exitDefault)} value={d.label ?? ''} onChange={(e) => set({ label: e.target.value || undefined })} />
        </Prop>
      ) : null}
      <div className="fe-foot">
        <Button className="mini-btn" disabledReason={!editable ? p.editReason : node.type === 'trigger' ? tr(mp.oneTrigger) : undefined} onClick={p.onDuplicate}>
          {tr(mp.duplicate)}
        </Button>
        <Button className="mini-btn" variant="danger" disabledReason={!editable ? p.editReason : undefined} onClick={p.onDelete}>
          {tr(mp.delete)}
        </Button>
      </div>
      <datalist id="fe-tags">
        {options?.tags.map((t) => <option key={t} value={t} />)}
      </datalist>
      <datalist id="fe-events">
        {options?.events.map((t) => <option key={t} value={t} />)}
      </datalist>
    </>
  )
}

function SkipRules({ node, options, editable, onChange }: StepPanelProps) {
  const tr = useT()
  const d = node.data
  const lists = options?.lists ?? []
  const suggested = lists.find((l) => l.name === 'Beta Program') ?? lists[lists.length - 1]
  const listId = d.email?.skipListId ?? suggested?.id
  const listName = lists.find((l) => l.id === listId)?.name
  return (
    <Prop title={tr(mp.skipIf)} id="p-skip">
      <Check checked={(d.email?.skipRecentHours ?? 0) > 0} disabled={!editable} onChange={(v) => onChange({ ...d, email: { ...d.email, skipRecentHours: v ? 4 : undefined } })}>
        {tr(mp.skipRecent, { hours: d.email?.skipRecentHours || 4 })}
      </Check>
      <Check checked={Boolean(d.email?.skipListId)} disabled={!editable || !listId} onChange={(v) => onChange({ ...d, email: { ...d.email, skipListId: v ? listId : undefined } })}>
        {listName ? tr(mp.skipInList, { list: listName }) : tr(mp.skipInAnyList)}
      </Check>
      {d.email?.skipListId ? (
        <select className="in" aria-label={tr(mp.skipListLabel)} disabled={!editable} value={d.email.skipListId} onChange={(e) => onChange({ ...d, email: { ...d.email, skipListId: e.target.value } })}>
          {lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      ) : null}
    </Prop>
  )
}

function TriggerProps({ graph, node, options, editable, onChange, stats, hasStats }: StepPanelProps) {
  const tr = useT()
  const t: TriggerConfig = node.data.trigger ?? { kind: 'joined_list' }
  const disabled = !editable
  const set = (patch: Partial<TriggerConfig>) => onChange({ ...node.data, trigger: { ...t, ...patch } })
  const stores = useStoreTriggerOptions().data?.stores
  return (
    <>
      <Prop title={tr(mp.startsWhen)} id="p-kind">
        <select
          aria-labelledby="p-kind"
          className="in"
          disabled={disabled}
          value={t.kind}
          onChange={(e) => {
            const kind = e.target.value as TriggerConfig['kind']
            onChange({ ...node.data, trigger: { reentry: t.reentry, ...(kind === 'score_crosses' ? scoreTriggerDefaults(options?.scoreBands) : {}), ...storeTriggerDefaults(kind, stores), kind } })
          }}
        >
          {TRIGGER_KINDS.map((k) => (
            <option key={k} value={k}>
              {triggerKindLabel(k)}
            </option>
          ))}
        </select>
        {t.kind === 'joined_list' ? (
          <>
            <select className="in" aria-label={tr(mp.list)} disabled={disabled} value={t.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
              <option value="">{tr(mp.pickList)}</option>
              {options?.lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
            <Check checked={Boolean(t.confirmedOnly)} disabled={disabled} onChange={(v) => set({ confirmedOnly: v || undefined })}>
              {tr(mp.confirmedOnly)}
            </Check>
            <Check checked={Boolean(t.includeImports)} disabled={disabled} onChange={(v) => set({ includeImports: v || undefined })}>
              {tr(mp.includeImports)}
            </Check>
          </>
        ) : null}
        {t.kind === 'form_submitted' ? (
          <select className="in" aria-label={tr(mp.form)} disabled={disabled} value={t.formId ?? ''} onChange={(e) => set({ formId: e.target.value })}>
            <option value="">{tr(mp.pickForm)}</option>
            {options?.forms.map((f) => (
              <option key={f.id} value={f.id}>
                {f.name} ({f.slug})
              </option>
            ))}
          </select>
        ) : null}
        {t.kind === 'tag_added' ? (
          <input className="in" aria-label={tr(mp.tag)} list="fe-tags" placeholder={tr(mp.tagPlaceholder)} disabled={disabled} value={t.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} />
        ) : null}
        {/* Answered a poll or survey (design expansion, X11). */}
        {t.kind === 'poll_answered' ? <PollTriggerProps t={t} disabled={disabled} set={set} /> : null}
        {/* Bought a product (digital products, X19). */}
        {t.kind === 'bought_product' ? <BoughtTriggerProps t={t} disabled={disabled} set={set} products={options?.products} /> : null}
        {/* Order placed, First order and Abandoned checkout (design expansion, X20). */}
        {isStoreTrigger(t.kind) ? <StoreTriggerFields t={t} set={set} disabled={disabled} graph={graph} /> : null}
        {t.kind === 'api_event' ? (
          <>
            <input className="in mono" aria-label={tr(mp.eventName)} list="fe-events" placeholder={'workspace.created'} disabled={disabled} value={t.event ?? ''} onChange={(e) => set({ event: e.target.value })} />
            <span className="k">
              <T msg={mp.apiEventNote} values={{ endpoint: 'POST /api/v1/workspaces/…/automations/events', mono }} />
            </span>
          </>
        ) : null}
        {t.kind === 'score_crosses' ? <ScoreTriggerFields t={t} set={set} disabled={disabled} bands={options?.scoreBands} /> : null}
        {t.kind === 'milestone_reached' ? <MilestoneTriggerFields t={t} set={set} disabled={disabled} milestones={options?.milestones ?? []} /> : null}
        {t.kind === 'date_field' ? (
          <>
            <select className="in" aria-label={tr(mp.dateField)} disabled={disabled} value={t.field ?? ''} onChange={(e) => set({ field: e.target.value })}>
              <option value="">{tr(mp.pickDateField)}</option>
              <option value="created_at">{tr(mp.subscribedOn)}</option>
              {options?.fields
                .filter((f) => f.type === 'date')
                .map((f) => (
                  <option key={f.key} value={f.key}>
                    {f.label}
                  </option>
                ))}
            </select>
            <div className="flex items-center gap-1.5">
              <input
                className="in w-20"
                type="number"
                aria-label={tr(mp.days)}
                disabled={disabled}
                value={Math.abs(t.offsetDays ?? 0)}
                onChange={(e) => set({ offsetDays: (t.offsetDays ?? 0) > 0 ? Number(e.target.value) : -Number(e.target.value) })}
              />
              <select
                className="in flex-1"
                aria-label={tr(mp.beforeOrAfter)}
                disabled={disabled}
                value={(t.offsetDays ?? 0) > 0 ? 'after' : 'before'}
                onChange={(e) => set({ offsetDays: e.target.value === 'after' ? Math.abs(t.offsetDays ?? 0) : -Math.abs(t.offsetDays ?? 0) })}
              >
                <option value="before">{tr(mp.daysBefore)}</option>
                <option value="after">{tr(mp.daysAfter)}</option>
              </select>
            </div>
            <div className="flex items-center gap-1.5">
              <input className="in w-24" type="time" aria-label={tr(mp.at)} disabled={disabled} value={t.at ?? '09:00'} onChange={(e) => set({ at: e.target.value })} />
              <Check checked={Boolean(t.yearly)} disabled={disabled} onChange={(v) => set({ yearly: v || undefined })}>
                {tr(mp.everyYear)}
              </Check>
            </div>
          </>
        ) : null}
      </Prop>
      {t.kind === 'score_crosses' || t.kind === 'milestone_reached' ? null : (
        <Prop title={tr(mp.reentry)} id="p-reentry">
          <select aria-labelledby="p-reentry" className="in" disabled={disabled} value={t.reentry ?? 'once'} onChange={(e) => set({ reentry: e.target.value })}>
            <option value="once">{tr(mp.oncePerContact)}</option>
            <option value="always">{tr(mp.everyTime)}</option>
          </select>
        </Prop>
      )}
      {hasStats && stats ? (
        <Prop title={tr(mp.liveStats)} id="p-tstats">
          <div className="fe-stats">
            <span>
              <T msg={mp.entered} values={{ value: count(stats.entered), b: bold }} />
            </span>
          </div>
        </Prop>
      ) : null}
    </>
  )
}

function SplitProps({ node, editable, onChange }: StepPanelProps) {
  const tr = useT()
  const vs = node.data.variants ?? []
  const sum = vs.reduce((s, v) => s + v.weight, 0)
  const set = (next: typeof vs) => onChange({ ...node.data, variants: next })
  return (
    <Prop title={tr(mp.variants)} id="p-variants">
      {vs.map((v, i) => (
        <div key={v.key} className="flex items-center gap-1.5">
          <span className="n w-6">{v.key}</span>
          <input
            className="in w-20"
            type="number"
            min={0}
            max={100}
            aria-label={tr(mp.variantWeight, { variant: v.key })}
            disabled={!editable}
            value={v.weight}
            onChange={(e) => set(vs.map((x, j) => (j === i ? { ...x, weight: Number(e.target.value) || 0 } : x)))}
          />
          <span className="k">%</span>
        </div>
      ))}
      <span className={sum === 100 ? 'k' : 'err'}>{sum === 100 ? tr(mp.weightsOk) : tr(mp.weightsOff, { sum: wholePct(sum) })}</span>
      {editable && vs.length < 5 ? (
        <div>
          <Button className="mini-btn" onClick={() => set([...vs, { key: String.fromCharCode(65 + vs.length), weight: 0 }])}>
            {tr(mp.addVariant)}
          </Button>
        </div>
      ) : null}
    </Prop>
  )
}

function WebhookProps({ node, editable, onChange, automationId }: StepPanelProps) {
  const tr = useT()
  const { slug } = useWorkspace()
  const w = node.data.webhook ?? {}
  const [secret, setSecret] = useState<string | null>(null)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/automations/{id}/steps/{node}/secret')
  const set = (patch: Partial<typeof w>) => onChange({ ...node.data, webhook: { ...w, ...patch } })
  return (
    <>
      <Prop title={tr(mp.name)} id="p-wname">
        <input aria-labelledby="p-wname" className="in" disabled={!editable} placeholder={tr(mp.webhookNamePlaceholder)} value={w.name ?? ''} onChange={(e) => set({ name: e.target.value || undefined })} />
      </Prop>
      <Prop title={tr(mp.url)} id="p-wurl">
        <div className="flex gap-1.5">
          <select className="in w-[78px]" aria-label={tr(mp.method)} disabled={!editable} value={w.method ?? 'POST'} onChange={(e) => set({ method: e.target.value })}>
            {/* HTTP methods, not words. */}
            <option>{'POST'}</option>
            <option>{'PUT'}</option>
          </select>
          <input aria-labelledby="p-wurl" className="in mono flex-1" disabled={!editable} placeholder={'https://'} value={w.url ?? ''} onChange={(e) => set({ url: e.target.value })} />
        </div>
        <span className="k">{tr(mp.publicOnly)}</span>
      </Prop>
      <Prop title={tr(mp.signingSecret)} id="p-wsecret">
        {secret === null ? (
          <div className="flex items-center gap-2">
            <span className="mono">{w.hasSecret ? '••••••••••••' : tr(mp.notSet)}</span>
            <Button className="mini-btn ml-auto" disabledReason={!editable ? tr(mp.secretReason) : undefined} onClick={() => setSecret('')}>
              {w.hasSecret ? tr(mp.replace) : tr(mp.set)}
            </Button>
          </div>
        ) : (
          <form
            className="flex flex-col gap-1.5"
            onSubmit={(e) => {
              e.preventDefault()
              save.mutate(
                { params: { path: { workspace: slug, id: automationId, node: node.id } }, body: { secret } },
                {
                  onSuccess: () => {
                    setSecret(null)
                    onChange({ ...node.data, webhook: { ...w, hasSecret: secret !== '' } })
                    toast({ title: secret ? tr(mp.secretSaved) : tr(mp.secretRemoved), detail: tr(mp.neverShown) })
                  },
                  onError: (err) => toast({ kind: 'error', title: tr(mp.secretFailed), detail: errorMessage(err) }),
                },
              )
            }}
          >
            <input className="in mono" type="password" autoComplete="off" aria-label={tr(mp.newSecret)} placeholder={tr(mp.secretPlaceholder)} value={secret} onChange={(e) => setSecret(e.target.value)} />
            <div className="flex gap-1.5">
              <Button type="submit" className="mini-btn" variant="primary" loading={save.isPending}>
                {tr(common.save)}
              </Button>
              <Button className="mini-btn" onClick={() => setSecret(null)}>
                {tr(common.cancel)}
              </Button>
            </div>
          </form>
        )}
        <span className="k">
          <T msg={mp.signatureNote} values={{ header: 'X-ReputeCast-Signature', mono }} />
        </span>
      </Prop>
    </>
  )
}

function ListProps({ node, options, editable, onChange }: StepPanelProps) {
  const tr = useT()
  const a = node.data.action ?? { kind: 'add_tag' as const }
  const set = (patch: Partial<typeof a>) => onChange({ ...node.data, action: { ...a, ...patch } })
  return (
    <Prop title={tr(mp.action)} id="p-action">
      <select aria-labelledby="p-action" className="in" disabled={!editable} value={a.kind} onChange={(e) => onChange({ ...node.data, action: { kind: e.target.value as typeof a.kind } })}>
        <option value="add_list">{tr(mp.addToList)}</option>
        <option value="remove_list">{tr(mp.removeFromList)}</option>
        <option value="add_tag">{tr(mp.addTag)}</option>
        <option value="remove_tag">{tr(mp.removeTag)}</option>
        <option value="unsubscribe_all">{tr(mp.unsubscribeAll)}</option>
      </select>
      {a.kind === 'add_list' || a.kind === 'remove_list' ? (
        <select className="in" aria-label={tr(mp.list)} disabled={!editable} value={a.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
          <option value="">{tr(mp.pickList)}</option>
          {options?.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      ) : null}
      {a.kind === 'add_tag' || a.kind === 'remove_tag' ? (
        <input className="in" aria-label={tr(mp.tag)} list="fe-tags" placeholder={tr(mp.tag)} disabled={!editable} value={a.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} />
      ) : null}
    </Prop>
  )
}

/** Right panel › Stats: the selected step's numbers and, for webhooks, the delivery log. */
export function StepStatsPanel({ node, stats: st, automationId, hasStats }: { node: FlowNode; stats?: StepStats; automationId: string; hasStats: boolean }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const calls = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automations/{id}/steps/{node}/calls',
    { params: { path: { workspace: slug, id: automationId, node: node.id }, query: { limit: 30 } } },
    { enabled: node.type === 'webhook' },
  )
  const rows: [MessageDescriptor, string][] = st
    ? [
        [mp.rowEntered, count(st.entered)],
        [mp.rowHereNow, count(st.inFlowNow)],
        [mp.rowMovedOn, count(st.completed)],
        ...(node.type === 'email'
          ? ([
              [mp.rowSent, count(st.sent)],
              [mp.rowSkipped, count(st.skipped)],
              [mp.rowDelivered, count(st.delivered)],
              [mp.rowHumanOpens, count(st.openedHuman)],
              [mp.rowAllOpens, count(st.opened)],
              [mp.rowClicks, count(st.clicked)],
              [mp.rowUnsubscribes, count(st.unsubscribed)],
            ] as [MessageDescriptor, string][])
          : []),
        ...(node.type === 'goal' ? ([[mp.rowReached, count(st.goalReached)]] as [MessageDescriptor, string][]) : []),
        ...(node.type === 'webhook' ? ([[mp.rowFailedCalls, count(st.failed)]] as [MessageDescriptor, string][]) : []),
        [mp.rowFinished, count(st.exited)],
        [mp.rowLeftEarly, count(st.dropped)],
      ]
    : []
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">{tr(mp.statsTitle, { type: typeName(node.type) })}</span>
        <span className="k">{hasStats ? tr(mp.statsAcross) : tr(mp.statsStart)}</span>
      </div>
      {st ? (
        <div className="fe-prop">
          <div className="fe-stats">
            {rows.map(([label, v]) => (
              <span key={label.id}>
                <T msg={label} values={{ value: v, b: bold }} />
              </span>
            ))}
          </div>
          {st.paths.length > 0 ? <span className="k">{st.paths.map((p) => `${pathLabel(p.path)} ${count(p.count)}`).join(' · ')}</span> : null}
          {st.providers.length > 0 ? (
            <span className="k">{st.providers.map((p) => (p.fallback ? tr(mp.providerFailover, { provider: p.short, count: count(p.sent) }) : `${p.short} ${count(p.sent)}`)).join(' · ')}</span>
          ) : null}
        </div>
      ) : (
        <div className="fe-prop">
          <span className="k">{tr(mp.noContacts)}</span>
        </div>
      )}
      {node.type === 'sms' ? <SmsStepStats nodeId={node.id} automationId={automationId} hasStats={hasStats} /> : null}
      {node.type === 'push' ? <PushStepStats nodeId={node.id} automationId={automationId} hasStats={hasStats} skipped={st?.skipped ?? 0} /> : null}
      {node.type === 'whatsapp' ? <WhatsAppStepStats nodeId={node.id} automationId={automationId} hasStats={hasStats} skipped={st?.skipped ?? 0} /> : null}
      {node.type === 'webhook' ? (
        <div className="fe-prop">
          <span className="tt">{tr(mp.deliveryLog)}</span>
          {calls.data?.calls.length ? (
            <ul className="m-0 flex list-none flex-col gap-1 p-0">
              {calls.data.calls.map((c) => (
                <li key={c.id} className="flex items-center justify-between gap-2 text-[12px]">
                  <span className={c.statusCode >= 200 && c.statusCode < 300 ? 'text-ok-fg' : 'text-bad-fg'}>{c.statusCode || tr(mp.callError)}</span>
                  <span className="k flex-1 truncate">{c.error || tr(mp.callDuration, { ms: count(c.durationMs) })}</span>
                  <span className="k">{tr(mp.callTime, { date: dayMonthDate(c.calledAt), time: clock(c.calledAt) })}</span>
                </li>
              ))}
            </ul>
          ) : (
            <span className="k">{calls.isPending ? tr(common.loading) : tr(mp.noCalls)}</span>
          )}
        </div>
      ) : null}
    </>
  )
}

/** The order the stats list and keyboard additions follow. */
export function graphOrder(g: FlowGraph): string[] {
  const t = g.nodes.find((n) => n.type === 'trigger')
  const seen = new Set<string>()
  const out: string[] = []
  const queue = t ? [t.id] : []
  while (queue.length) {
    const id = queue.shift()!
    if (seen.has(id)) continue
    seen.add(id)
    out.push(id)
    for (const e of outEdges(g, id)) queue.push(e.target)
  }
  for (const n of g.nodes) if (!seen.has(n.id)) out.push(n.id)
  return out
}

export { inEdges }
