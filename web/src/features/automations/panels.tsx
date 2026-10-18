import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { count, pct, rate } from '@/lib/format'
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
  triggerKindLabels,
  typeColors,
  typeNames,
  type FlowCondition,
  type FlowDuration,
  type FlowGraph,
  type FlowNode,
  type NodeData,
  type Refs,
  type StepType,
  type TriggerConfig,
} from './graph'
import type { StepStats } from './view'
import { StepLanguages } from '@/features/languages/automation'
import { ScoreTriggerFields, scoreTriggerDefaults } from '@/features/scoring/ScoreTrigger'
import { isOneOfLabel, scoreTriggerName } from '@/features/scoring/triggerText'
import { MilestoneTriggerFields } from '@/features/grow/MilestoneTrigger'
import { milestoneTriggerName } from '@/features/grow/triggerText'

type Options = Schemas['AutomationOptions']

export const DRAG_TYPE = 'application/x-reputecast-step'

/** The step type being dragged, readable during dragover (data is not). */
export function draggedStepType(types: readonly string[]): StepType | undefined {
  const t = types.find((x) => x.startsWith(`${DRAG_TYPE}-`))
  return t ? (t.slice(DRAG_TYPE.length + 1) as StepType) : undefined
}

const triggerItems: { kind: TriggerConfig['kind']; label: string }[] = [
  { kind: 'joined_list', label: 'Joined list' },
  { kind: 'form_submitted', label: 'Form submitted' },
  { kind: 'tag_added', label: 'Tag added' },
  { kind: 'api_event', label: 'API event' },
  { kind: 'date_field', label: 'Date field' },
  // Contact scoring (X13).
  {
    kind: 'score_crosses',
    get label() {
      return scoreTriggerName()
    },
  },
  // Referral milestones (Grow).
  {
    kind: 'milestone_reached',
    get label() {
      return milestoneTriggerName()
    },
  },
]

const stepItems: { type: StepType; label: string }[] = [
  { type: 'delay', label: 'Delay' },
  { type: 'email', label: 'Send email' },
  { type: 'branch', label: 'Branch' },
  { type: 'filter', label: 'Filter' },
  { type: 'split', label: 'A/B split' },
  { type: 'webhook', label: 'Webhook' },
  { type: 'list', label: 'List action' },
  { type: 'goal', label: 'Goal / exit' },
]

export interface PaletteItem {
  type: StepType
  trigger?: TriggerConfig['kind']
}

/** Steps to drag onto the canvas (or press to add after the selected step). */
export function Palette({ disabledReason, onAdd }: { disabledReason?: string; onAdd: (item: PaletteItem) => void }) {
  const item = (key: string, label: string, color: string, value: PaletteItem) => (
    <button
      key={key}
      type="button"
      className="fe-pal"
      draggable={!disabledReason}
      aria-disabled={disabledReason ? true : undefined}
      title={disabledReason ?? `Add ${label.toLowerCase()} after the selected step`}
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
      <div className="grp">Triggers</div>
      <div className="flex flex-col gap-1.5 px-3 pb-2">{triggerItems.map((t) => item(t.kind, t.label, typeColors.trigger, { type: 'trigger', trigger: t.kind }))}</div>
      <div className="grp">Steps</div>
      <div className="flex flex-col gap-1.5 px-3 pb-2">{stepItems.map((s) => item(s.type, s.label, typeColors[s.type], { type: s.type }))}</div>
    </>
  )
}

function stepSummary(n: FlowNode, st: StepStats | undefined): string {
  if (!st) return 'no stats yet'
  switch (n.type) {
    case 'trigger':
      return `${count(st.entered)} entered`
    case 'email':
      return `${count(st.sent)} sent · ${pct(st.delivered ? (st.openedHuman / st.delivered) * 100 : 0)} human opens`
    case 'goal':
      return `${count(st.goalReached)} reached`
    case 'exit':
      return `${count(st.exited)} exited`
    case 'branch':
    case 'split':
      return st.paths.map((p) => `${pathLabel(p.path)} ${count(p.count)}`).join(' · ') || `${count(st.entered)} entered`
    case 'webhook':
      return `${count(st.entered)} calls · ${count(st.failed)} failed`
  }
  return `${count(st.entered)} entered · ${count(st.inFlowNow)} here now`
}

/** Left panel › Stats: every step with its key number; a list view of the canvas. */
export function FlowStatsList({ graph, order, refs, stats, selected, onSelect }: { graph: FlowGraph; order: string[]; refs: Refs; stats: Map<string, StepStats>; selected?: string; onSelect: (id: string) => void }) {
  return (
    <div className="flex flex-col">
      {order.map((id) => {
        const n = nodeById(graph, id)
        if (!n) return null
        return (
          <button key={id} type="button" className="fe-steprow" aria-current={selected === id ? 'true' : undefined} onClick={() => onSelect(id)}>
            <span className="overline" style={{ color: typeColors[n.type] }}>
              {typeNames[n.type]}
            </span>
            <span className="font-semibold">{nodeTitle(graph, n, refs)}</span>
            <span className="k">{stepSummary(n, stats.get(id))}</span>
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

const subs: Record<StepType, string> = {
  trigger: 'Starts the flow',
  delay: 'Holds contacts, then moves on',
  email: 'Automations stream',
  branch: 'Splits contacts by a condition',
  filter: 'Lets matching contacts continue',
  split: 'Splits contacts at random',
  webhook: 'Calls your URL for each contact',
  list: 'Changes lists or tags',
  goal: 'Ends the flow when reached',
  exit: 'Ends the flow',
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
  const d = value ?? { amount: 1, unit: 'days' as const }
  return (
    <div className="flex gap-1.5">
      <input
        className="in w-20"
        type="number"
        min={1}
        aria-label={`${label}: amount`}
        disabled={disabled}
        value={d.amount}
        onChange={(e) => onChange({ ...d, amount: Math.max(0, Number(e.target.value) || 0) })}
      />
      <select className="in flex-1" aria-label={`${label}: unit`} disabled={disabled} value={d.unit} onChange={(e) => onChange({ ...d, unit: e.target.value as FlowDuration['unit'] })}>
        <option value="minutes">minutes</option>
        <option value="hours">hours</option>
        <option value="days">days</option>
        <option value="weeks">weeks</option>
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
  const kinds: { value: FlowCondition['kind'] | ''; label: string }[] = [
    ...(allowNone ? [{ value: '' as const, label: 'Reaching this step' }] : []),
    { value: 'opened', label: 'Opened an email' },
    { value: 'clicked', label: 'Clicked a link' },
    { value: 'tag', label: 'Has a tag' },
    { value: 'list', label: 'Is in a list' },
    { value: 'event', label: 'Sent an API event' },
    { value: 'field', label: 'A contact field' },
  ]
  const set = (patch: Partial<FlowCondition>) => onChange({ ...(c ?? { kind: 'opened' }), ...patch } as FlowCondition)
  return (
    <>
      <select
        className="in"
        aria-label="Condition"
        disabled={disabled}
        value={c?.kind ?? ''}
        onChange={(e) => {
          const k = e.target.value as FlowCondition['kind'] | ''
          if (!k) return onChange(undefined)
          onChange({ kind: k, ...(withWindow && (k === 'opened' || k === 'clicked') ? { within: c?.within ?? { amount: 2, unit: 'days' } } : {}), humanOnly: k === 'opened' ? true : undefined })
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
          <select className="in" aria-label="Which email" disabled={disabled} value={c.step ?? ''} onChange={(e) => set({ step: e.target.value || undefined })}>
            <option value="">Any email in this flow</option>
            {emails.map((e) => (
              <option key={e.id} value={e.id}>
                {nodeRef(graph, e, refs)}
              </option>
            ))}
          </select>
          {c.kind === 'clicked' ? (
            <input className="in" aria-label="Link contains" placeholder="Link contains (optional), e.g. docs" disabled={disabled} value={c.urlContains ?? ''} onChange={(e) => set({ urlContains: e.target.value || undefined })} />
          ) : (
            <Check checked={Boolean(c.humanOnly)} disabled={disabled} onChange={(v) => set({ humanOnly: v || undefined })}>
              Human opens only (ignore Apple Mail Privacy Protection)
            </Check>
          )}
          {withWindow ? (
            <>
              <span className="k">Wait up to</span>
              <DurationInput label="Window" value={c.within} disabled={disabled} onChange={(d) => set({ within: d })} />
            </>
          ) : null}
        </>
      ) : null}
      {c?.kind === 'tag' ? <input className="in" aria-label="Tag" list="fe-tags" placeholder="Tag" disabled={disabled} value={c.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} /> : null}
      {c?.kind === 'list' ? (
        <select className="in" aria-label="List" disabled={disabled} value={c.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
          <option value="">Pick a list</option>
          {options?.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      ) : null}
      {c?.kind === 'event' ? <input className="in mono" aria-label="Event" list="fe-events" placeholder="teammate.invited" disabled={disabled} value={c.event ?? ''} onChange={(e) => set({ event: e.target.value })} /> : null}
      {c?.kind === 'field' ? (
        <>
          <input className="in" aria-label="Field" placeholder="Field, e.g. plan" disabled={disabled} value={c.field ?? ''} onChange={(e) => set({ field: e.target.value })} />
          <select className="in" aria-label="Comparison" disabled={disabled} value={c.op ?? 'equals'} onChange={(e) => set({ op: e.target.value })}>
            <option value="equals">is</option>
            <option value="not_equals">isn't</option>
            <option value="contains">contains</option>
            <option value="is_one_of">{isOneOfLabel()}</option>
            <option value="is_set">is set</option>
            <option value="is_not_set">is empty</option>
          </select>
          {c.op !== 'is_set' && c.op !== 'is_not_set' ? <input className="in" aria-label="Value" placeholder="Value" disabled={disabled} value={c.value ?? ''} onChange={(e) => set({ value: e.target.value })} /> : null}
        </>
      ) : null}
    </>
  )
}

/** The emails of the workspace as select options: "name · v5". */
function EmailSelect({ value, version, options, disabled, onPick, onCreate }: { value?: string; version?: number; options?: Options; disabled?: boolean; onPick: (id: string, version: number) => void; onCreate: () => void }) {
  const current = options?.emails.find((e) => e.id === value)
  return (
    <select
      aria-label="Email"
      className="in"
      disabled={disabled}
      value={value ? `${value}@${version ?? current?.latestVersion ?? 1}` : ''}
      onChange={(e) => {
        if (e.target.value === 'new') return onCreate()
        const [id, v] = e.target.value.split('@')
        if (id) onPick(id, Number(v))
      }}
    >
      {!value ? <option value="">Pick an email</option> : null}
      {current ? (
        <option value={`${current.id}@${version ?? current.latestVersion}`}>
          {current.name} · v{version ?? current.latestVersion}
        </option>
      ) : null}
      {current && version !== undefined && version < current.latestVersion ? (
        <option value={`${current.id}@${current.latestVersion}`}>
          {current.name} · v{current.latestVersion} (newer)
        </option>
      ) : null}
      {options?.emails
        .filter((e) => e.id !== value)
        .map((e) => (
          <option key={e.id} value={`${e.id}@${e.latestVersion}`}>
            {e.name} · v{e.latestVersion}
          </option>
        ))}
      <option value="new">Create new from template</option>
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
  const { slug } = useWorkspace()
  const d = node.data
  const disabled = !editable
  const set = (patch: Partial<NodeData>) => p.onChange({ ...d, ...patch })
  const qc = useQueryClient()
  const createEmail = $api.useMutation('post', '/api/v1/workspaces/{workspace}/automation-emails')
  const position = graph.nodes.filter((n) => n.type === 'email').findIndex((n) => n.id === node.id) + 1
  const title = node.type === 'email' ? `${typeNames.email} · ${emailHeader(graph, node, refs)}` : `${typeNames[node.type]} · ${node.type === 'trigger' ? triggerKindLabels[d.trigger?.kind ?? 'joined_list'] : nodeRef(graph, node, refs)}`
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">{title}</span>
        <span className="k">{subs[node.type]}</span>
      </div>
      {p.issues.length > 0 ? (
        <div className="fe-prop">
          <Note tone="bad">{p.issues.join(' ')}</Note>
        </div>
      ) : null}
      {node.type === 'trigger' ? <TriggerProps {...p} /> : null}
      {node.type === 'delay' ? (
        <Prop title="Wait" id="p-wait">
          <DurationInput label="Wait" value={d.delay} disabled={disabled} onChange={(v) => set({ delay: v })} />
          <span className="k">Contacts already waiting keep their timer until you publish.</span>
        </Prop>
      ) : null}
      {node.type === 'email' ? (
        <>
          <Prop title="Email" id="p-email">
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
                  { params: { path: { workspace: slug } }, body: { name: `Step ${position} "New email"`, subject: d.email?.subject || 'New email', automationId: p.automationId } },
                  {
                    onSuccess: (e) => {
                      set({ email: { ...d.email, emailId: e.id, emailVersion: e.latestVersion, subject: d.email?.subject || e.subject } })
                      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/options'] })
                    },
                    onError: (err) => toast({ kind: 'error', title: "Couldn't create the email", detail: errorMessage(err) }),
                  },
                )
              }
            />
            <div className="flex gap-1.5">
              <Button className="mini-btn" disabledReason={!d.email?.emailId ? 'Pick an email first.' : editable ? undefined : p.editReason} onClick={() => d.email?.emailId && p.onEditEmail(d.email.emailId, d.email.emailVersion ?? 1)}>
                Edit in composer
              </Button>
              <Button className="mini-btn" disabledReason={!d.email?.emailId ? 'Pick an email first.' : undefined} onClick={() => d.email?.emailId && p.onPreview(d.email.emailId, d.email.emailVersion ?? 1)}>
                Preview
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
          <Prop title="Subject" id="p-subject">
            <input aria-labelledby="p-subject" className="in" disabled={disabled} value={d.email?.subject ?? ''} onChange={(e) => set({ email: { ...d.email, subject: e.target.value } })} />
          </Prop>
          <Prop title="Sending route" id="p-route">
            <select aria-labelledby="p-route" className="in" disabled={disabled} value={d.email?.route ?? ''} onChange={(e) => set({ email: { ...d.email, route: e.target.value || undefined } })}>
              <option value="">Automations route · {p.routeLabel} (default)</option>
              {options?.providers
                .filter((pr) => !p.routePrimary.includes(pr.id) || pr.id === d.email?.route)
                .map((pr) => (
                  <option key={pr.id} value={pr.id}>
                    Override: {pr.short} only
                  </option>
                ))}
            </select>
            <span className="k">Automations fail over automatically after 3 failed sends.</span>
          </Prop>
          <Prop title="Send window" id="p-window">
            <Check
              checked={Boolean(d.email?.sendWindow?.enabled)}
              disabled={disabled}
              onChange={(v) => set({ email: { ...d.email, sendWindow: { start: d.email?.sendWindow?.start ?? '07:00', end: d.email?.sendWindow?.end ?? '21:00', enabled: v } } })}
            >
              Only between {d.email?.sendWindow?.start ?? '07:00'} and {d.email?.sendWindow?.end ?? '21:00'} recipient-local; otherwise hold until morning
            </Check>
          </Prop>
          <SkipRules {...p} />
          {p.hasStats && st ? (
            <Prop title="Live stats · 30 d" id="p-stats">
              <div className="fe-stats">
                <span>
                  Sent <b className="n">{count(st.sent)}</b>
                </span>
                <span>
                  Delivered <b className="n">{pct(st.sent ? (st.delivered / st.sent) * 100 : undefined)}</b>
                </span>
                <span>
                  Human opens <b className="n">{pct(st.delivered ? (st.openedHuman / st.delivered) * 100 : undefined)}</b>
                </span>
                <span>
                  Clicks <b className="n">{pct(st.delivered ? (st.clicked / st.delivered) * 100 : undefined)}</b>
                </span>
                <span>
                  Bounces <b className="n">{pct(st.sent ? (st.bounced / st.sent) * 100 : undefined)}</b>
                </span>
                <span>
                  Complaints <b className="n">{rate(st.delivered ? (st.complained / st.delivered) * 100 : undefined)}</b>
                </span>
              </div>
            </Prop>
          ) : null}
        </>
      ) : null}
      {node.type === 'branch' ? (
        <Prop title="Condition" id="p-cond">
          <ConditionEditor graph={graph} node={node} value={d.condition} refs={refs} options={options} disabled={disabled} withWindow onChange={(c) => set({ condition: c })} />
          <span className="k">Contacts who meet it take the Yes path; the others take No{d.condition?.within ? ` after ${durationText(d.condition.within)}` : ''}.</span>
        </Prop>
      ) : null}
      {node.type === 'filter' ? (
        <Prop title="Only contacts who" id="p-filter">
          <ConditionEditor graph={graph} node={node} value={d.condition} refs={refs} options={options} disabled={disabled} onChange={(c) => set({ condition: c })} />
          <span className="k">The others leave the flow here.</span>
        </Prop>
      ) : null}
      {node.type === 'split' ? <SplitProps {...p} /> : null}
      {node.type === 'webhook' ? <WebhookProps {...p} /> : null}
      {node.type === 'list' ? <ListProps {...p} /> : null}
      {(node.type === 'goal' || node.type === 'exit') && p.onChangeType ? (
        <Prop title="Ends the flow as" id="p-end">
          <select aria-labelledby="p-end" className="in" disabled={disabled} value={node.type} onChange={(e) => p.onChangeType?.(e.target.value as StepType)}>
            <option value="goal">Goal: counts toward the goal rate</option>
            <option value="exit">Exit: leaves the flow</option>
          </select>
        </Prop>
      ) : null}
      {node.type === 'goal' ? (
        <>
          <Prop title="Goal name" id="p-goalname">
            <input aria-labelledby="p-goalname" className="in" disabled={disabled} placeholder="e.g. Clicked docs" value={d.label ?? ''} onChange={(e) => set({ label: e.target.value || undefined })} />
          </Prop>
          <Prop title="Reached when the contact" id="p-goal">
            <ConditionEditor graph={graph} node={node} value={d.condition} refs={refs} options={options} disabled={disabled} allowNone onChange={(c) => set({ condition: c })} />
            <span className="k">Contacts who meet the goal anywhere in the flow jump here and finish.</span>
          </Prop>
        </>
      ) : null}
      {node.type === 'exit' ? (
        <Prop title="Name" id="p-exitname">
          <input aria-labelledby="p-exitname" className="in" disabled={disabled} placeholder="Leave the flow · stays on the list" value={d.label ?? ''} onChange={(e) => set({ label: e.target.value || undefined })} />
        </Prop>
      ) : null}
      <div className="fe-foot">
        <Button className="mini-btn" disabledReason={!editable ? p.editReason : node.type === 'trigger' ? 'A flow has one trigger.' : undefined} onClick={p.onDuplicate}>
          Duplicate step
        </Button>
        <Button className="mini-btn" variant="danger" disabledReason={!editable ? p.editReason : undefined} onClick={p.onDelete}>
          Delete step
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
  const d = node.data
  const lists = options?.lists ?? []
  const suggested = lists.find((l) => l.name === 'Beta Program') ?? lists[lists.length - 1]
  const listId = d.email?.skipListId ?? suggested?.id
  const listName = lists.find((l) => l.id === listId)?.name ?? 'a'
  return (
    <Prop title="Skip if" id="p-skip">
      <Check checked={(d.email?.skipRecentHours ?? 0) > 0} disabled={!editable} onChange={(v) => onChange({ ...d, email: { ...d.email, skipRecentHours: v ? 4 : undefined } })}>
        Contact received any email in the last {d.email?.skipRecentHours || 4} hours
      </Check>
      <Check checked={Boolean(d.email?.skipListId)} disabled={!editable || !listId} onChange={(v) => onChange({ ...d, email: { ...d.email, skipListId: v ? listId : undefined } })}>
        Contact is in the {listName} list
      </Check>
      {d.email?.skipListId ? (
        <select className="in" aria-label="Skip contacts in this list" disabled={!editable} value={d.email.skipListId} onChange={(e) => onChange({ ...d, email: { ...d.email, skipListId: e.target.value } })}>
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

function TriggerProps({ node, options, editable, onChange, stats, hasStats }: StepPanelProps) {
  const t: TriggerConfig = node.data.trigger ?? { kind: 'joined_list' }
  const disabled = !editable
  const set = (patch: Partial<TriggerConfig>) => onChange({ ...node.data, trigger: { ...t, ...patch } })
  return (
    <>
      <Prop title="Starts when" id="p-kind">
        <select aria-labelledby="p-kind" className="in" disabled={disabled} value={t.kind} onChange={(e) => {
            const kind = e.target.value as TriggerConfig['kind']
            onChange({ ...node.data, trigger: { ...(kind === 'score_crosses' ? scoreTriggerDefaults(options?.scoreBands) : {}), kind, reentry: t.reentry } })
          }}
        >
          {Object.entries(triggerKindLabels).map(([k, label]) => (
            <option key={k} value={k}>
              {label}
            </option>
          ))}
        </select>
        {t.kind === 'joined_list' ? (
          <>
            <select className="in" aria-label="List" disabled={disabled} value={t.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
              <option value="">Pick a list</option>
              {options?.lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
            <Check checked={Boolean(t.confirmedOnly)} disabled={disabled} onChange={(v) => set({ confirmedOnly: v || undefined })}>
              Double opt-in confirmed only
            </Check>
            <Check checked={Boolean(t.includeImports)} disabled={disabled} onChange={(v) => set({ includeImports: v || undefined })}>
              Include contacts added by an import
            </Check>
          </>
        ) : null}
        {t.kind === 'form_submitted' ? (
          <select className="in" aria-label="Form" disabled={disabled} value={t.formId ?? ''} onChange={(e) => set({ formId: e.target.value })}>
            <option value="">Pick a form</option>
            {options?.forms.map((f) => (
              <option key={f.id} value={f.id}>
                {f.name} ({f.slug})
              </option>
            ))}
          </select>
        ) : null}
        {t.kind === 'tag_added' ? <input className="in" aria-label="Tag" list="fe-tags" placeholder="Tag, e.g. sunset-entered" disabled={disabled} value={t.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} /> : null}
        {t.kind === 'api_event' ? (
          <>
            <input className="in mono" aria-label="Event name" list="fe-events" placeholder="workspace.created" disabled={disabled} value={t.event ?? ''} onChange={(e) => set({ event: e.target.value })} />
            <span className="k">
              Send it with <span className="mono">POST /api/v1/workspaces/…/automations/events</span> and an API key with subscribers:edit.
            </span>
          </>
        ) : null}
        {t.kind === 'score_crosses' ? <ScoreTriggerFields t={t} set={set} disabled={disabled} bands={options?.scoreBands} /> : null}
        {t.kind === 'milestone_reached' ? <MilestoneTriggerFields t={t} set={set} disabled={disabled} milestones={options?.milestones ?? []} /> : null}
        {t.kind === 'date_field' ? (
          <>
            <select className="in" aria-label="Date field" disabled={disabled} value={t.field ?? ''} onChange={(e) => set({ field: e.target.value })}>
              <option value="">Pick a date field</option>
              <option value="created_at">Subscribed on</option>
              {options?.fields
                .filter((f) => f.type === 'date')
                .map((f) => (
                  <option key={f.key} value={f.key}>
                    {f.label}
                  </option>
                ))}
            </select>
            <div className="flex items-center gap-1.5">
              <input className="in w-20" type="number" aria-label="Days" disabled={disabled} value={Math.abs(t.offsetDays ?? 0)} onChange={(e) => set({ offsetDays: (t.offsetDays ?? 0) > 0 ? Number(e.target.value) : -Number(e.target.value) })} />
              <select className="in flex-1" aria-label="Before or after" disabled={disabled} value={(t.offsetDays ?? 0) > 0 ? 'after' : 'before'} onChange={(e) => set({ offsetDays: e.target.value === 'after' ? Math.abs(t.offsetDays ?? 0) : -Math.abs(t.offsetDays ?? 0) })}>
                <option value="before">days before</option>
                <option value="after">days after</option>
              </select>
            </div>
            <div className="flex items-center gap-1.5">
              <input className="in w-24" type="time" aria-label="At" disabled={disabled} value={t.at ?? '09:00'} onChange={(e) => set({ at: e.target.value })} />
              <Check checked={Boolean(t.yearly)} disabled={disabled} onChange={(v) => set({ yearly: v || undefined })}>
                Every year
              </Check>
            </div>
          </>
        ) : null}
      </Prop>
      {t.kind === 'score_crosses' || t.kind === 'milestone_reached' ? null : (
        <Prop title="Re-entry" id="p-reentry">
          <select aria-labelledby="p-reentry" className="in" disabled={disabled} value={t.reentry ?? 'once'} onChange={(e) => set({ reentry: e.target.value })}>
            <option value="once">Once per contact</option>
            <option value="always">Every time, once they've left</option>
          </select>
        </Prop>
      )}
      {hasStats && stats ? (
        <Prop title="Live stats · 30 d" id="p-tstats">
          <div className="fe-stats">
            <span>
              Entered <b className="n">{count(stats.entered)}</b>
            </span>
          </div>
        </Prop>
      ) : null}
    </>
  )
}

function SplitProps({ node, editable, onChange }: StepPanelProps) {
  const vs = node.data.variants ?? []
  const sum = vs.reduce((s, v) => s + v.weight, 0)
  const set = (next: typeof vs) => onChange({ ...node.data, variants: next })
  return (
    <Prop title="Variants" id="p-variants">
      {vs.map((v, i) => (
        <div key={v.key} className="flex items-center gap-1.5">
          <span className="n w-6">{v.key}</span>
          <input
            className="in w-20"
            type="number"
            min={0}
            max={100}
            aria-label={`Weight of variant ${v.key}`}
            disabled={!editable}
            value={v.weight}
            onChange={(e) => set(vs.map((x, j) => (j === i ? { ...x, weight: Number(e.target.value) || 0 } : x)))}
          />
          <span className="k">%</span>
        </div>
      ))}
      <span className={sum === 100 ? 'k' : 'err'}>{sum === 100 ? 'Weights add up to 100%.' : `Weights add up to ${sum}%; make them 100%.`}</span>
      {editable && vs.length < 5 ? (
        <div>
          <Button className="mini-btn" onClick={() => set([...vs, { key: String.fromCharCode(65 + vs.length), weight: 0 }])}>
            Add variant
          </Button>
        </div>
      ) : null}
    </Prop>
  )
}

function WebhookProps({ node, editable, onChange, automationId }: StepPanelProps) {
  const { slug } = useWorkspace()
  const w = node.data.webhook ?? {}
  const [secret, setSecret] = useState<string | null>(null)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/automations/{id}/steps/{node}/secret')
  const set = (patch: Partial<typeof w>) => onChange({ ...node.data, webhook: { ...w, ...patch } })
  return (
    <>
      <Prop title="Name" id="p-wname">
        <input aria-labelledby="p-wname" className="in" disabled={!editable} placeholder="e.g. notify-slack" value={w.name ?? ''} onChange={(e) => set({ name: e.target.value || undefined })} />
      </Prop>
      <Prop title="URL" id="p-wurl">
        <div className="flex gap-1.5">
          <select className="in w-[78px]" aria-label="Method" disabled={!editable} value={w.method ?? 'POST'} onChange={(e) => set({ method: e.target.value })}>
            <option>POST</option>
            <option>PUT</option>
          </select>
          <input aria-labelledby="p-wurl" className="in mono flex-1" disabled={!editable} placeholder="https://" value={w.url ?? ''} onChange={(e) => set({ url: e.target.value })} />
        </div>
        <span className="k">Public addresses only. Contacts continue past the step even if the call fails.</span>
      </Prop>
      <Prop title="Signing secret" id="p-wsecret">
        {secret === null ? (
          <div className="flex items-center gap-2">
            <span className="mono">{w.hasSecret ? '••••••••••••' : 'Not set'}</span>
            <Button className="mini-btn ml-auto" disabledReason={!editable ? 'Editing needs automations:edit.' : undefined} onClick={() => setSecret('')}>
              {w.hasSecret ? 'Replace' : 'Set'}
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
                    toast({ title: secret ? 'Signing secret saved' : 'Signing secret removed', detail: 'It is never shown again.' })
                  },
                  onError: (err) => toast({ kind: 'error', title: "Couldn't save the secret", detail: errorMessage(err) }),
                },
              )
            }}
          >
            <input className="in mono" type="password" autoComplete="off" aria-label="New signing secret" placeholder="At least 16 characters" value={secret} onChange={(e) => setSecret(e.target.value)} />
            <div className="flex gap-1.5">
              <Button type="submit" className="mini-btn" variant="primary" loading={save.isPending}>
                Save
              </Button>
              <Button className="mini-btn" onClick={() => setSecret(null)}>
                Cancel
              </Button>
            </div>
          </form>
        )}
        <span className="k">
          Calls carry <span className="mono">X-ReputeCast-Signature</span> (HMAC-SHA256). The secret is write-only.
        </span>
      </Prop>
    </>
  )
}

function ListProps({ node, options, editable, onChange }: StepPanelProps) {
  const a = node.data.action ?? { kind: 'add_tag' as const }
  const set = (patch: Partial<typeof a>) => onChange({ ...node.data, action: { ...a, ...patch } })
  return (
    <Prop title="Action" id="p-action">
      <select aria-labelledby="p-action" className="in" disabled={!editable} value={a.kind} onChange={(e) => onChange({ ...node.data, action: { kind: e.target.value as typeof a.kind } })}>
        <option value="add_list">Add to a list</option>
        <option value="remove_list">Remove from a list</option>
        <option value="add_tag">Add a tag</option>
        <option value="remove_tag">Remove a tag</option>
        <option value="unsubscribe_all">Unsubscribe from all lists</option>
      </select>
      {a.kind === 'add_list' || a.kind === 'remove_list' ? (
        <select className="in" aria-label="List" disabled={!editable} value={a.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
          <option value="">Pick a list</option>
          {options?.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      ) : null}
      {a.kind === 'add_tag' || a.kind === 'remove_tag' ? <input className="in" aria-label="Tag" list="fe-tags" placeholder="Tag" disabled={!editable} value={a.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} /> : null}
    </Prop>
  )
}

/** Right panel › Stats: the selected step's numbers and, for webhooks, the delivery log. */
export function StepStatsPanel({ node, stats: st, automationId, hasStats }: { node: FlowNode; stats?: StepStats; automationId: string; hasStats: boolean }) {
  const { slug } = useWorkspace()
  const calls = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automations/{id}/steps/{node}/calls',
    { params: { path: { workspace: slug, id: automationId, node: node.id }, query: { limit: 30 } } },
    { enabled: node.type === 'webhook' },
  )
  const rows: [string, string][] = st
    ? [
        ['Entered', count(st.entered)],
        ['Here now', count(st.inFlowNow)],
        ['Moved on', count(st.completed)],
        ...(node.type === 'email'
          ? ([
              ['Sent', count(st.sent)],
              ['Skipped by a rule', count(st.skipped)],
              ['Delivered', count(st.delivered)],
              ['Human opens', count(st.openedHuman)],
              ['All opens', count(st.opened)],
              ['Clicks', count(st.clicked)],
              ['Unsubscribes', count(st.unsubscribed)],
            ] as [string, string][])
          : []),
        ...(node.type === 'goal' ? ([['Reached', count(st.goalReached)]] as [string, string][]) : []),
        ...(node.type === 'webhook' ? ([['Failed calls', count(st.failed)]] as [string, string][]) : []),
        ['Finished here', count(st.exited)],
        ['Left early', count(st.dropped)],
      ]
    : []
  return (
    <>
      <div className="fe-ph">
        <span className="font-semibold">{typeNames[node.type]} · last 30 days</span>
        <span className="k">{hasStats ? 'Counted across versions; steps keep their numbers when you publish.' : 'Stats start when the automation is published.'}</span>
      </div>
      {st ? (
        <div className="fe-prop">
          <div className="fe-stats">
            {rows.map(([k, v]) => (
              <span key={k}>
                {k} <b className="n">{v}</b>
              </span>
            ))}
          </div>
          {st.paths.length > 0 ? <span className="k">{st.paths.map((p) => `${pathLabel(p.path)} ${count(p.count)}`).join(' · ')}</span> : null}
          {st.providers.length > 0 ? <span className="k">{st.providers.map((p) => `${p.short} ${count(p.sent)}${p.fallback ? ' (failover)' : ''}`).join(' · ')}</span> : null}
        </div>
      ) : (
        <div className="fe-prop">
          <span className="k">No contacts reached this step in the last 30 days.</span>
        </div>
      )}
      {node.type === 'webhook' ? (
        <div className="fe-prop">
          <span className="tt">Delivery log</span>
          {calls.data?.calls.length ? (
            <ul className="m-0 flex list-none flex-col gap-1 p-0">
              {calls.data.calls.map((c) => (
                <li key={c.id} className="flex items-center justify-between gap-2 text-[12px]">
                  <span className={c.statusCode >= 200 && c.statusCode < 300 ? 'text-ok-fg' : 'text-bad-fg'}>{c.statusCode || 'error'}</span>
                  <span className="k flex-1 truncate">{c.error || `${c.durationMs} ms`}</span>
                  <span className="k">{new Date(c.calledAt).toLocaleString('en-GB', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}</span>
                </li>
              ))}
            </ul>
          ) : (
            <span className="k">{calls.isPending ? 'Loading…' : 'No calls yet.'}</span>
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
