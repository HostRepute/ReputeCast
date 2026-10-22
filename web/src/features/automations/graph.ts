import type { Schemas } from '@/api/client'
import { scoreTriggerDefaults } from '@/features/scoring/ScoreTrigger'
import { isOneOfPhrase, scoreTriggerDetail, scoreTriggerName, scoreTriggerRef } from '@/features/scoring/triggerText'
import { boughtGoalTitle, boughtPhrase, boughtRef, boughtTriggerName } from '@/features/products/automationText'
import { milestoneTriggerDetail, milestoneTriggerName, milestoneTriggerRef } from '@/features/grow/triggerText'
import { m as smsStep } from '@/features/sms/automation.messages'
import { t as tt, type MessageDescriptor } from '@/i18n'
import { pct } from '@/lib/format'
import { mg } from './graph.messages'

/*
 * Flow graph helpers shared by the canvas, the step panel and the tests:
 * step titles (they mirror internal/automations/describe.go so the canvas and
 * the publish diff name steps the same way), edge geometry, entrance timing
 * and the "pulse through steps" timeline.
 */

export type FlowGraph = Schemas['AutomationGraph']
export type FlowNode = Schemas['AutomationNode']
export type FlowEdge = Schemas['AutomationEdge']
export type NodeData = Schemas['AutomationNodeData']
export type StepType = FlowNode['type']
export type TriggerConfig = Schemas['AutomationTrigger']
export type FlowCondition = Schemas['AutomationCondition']
export type FlowDuration = Schemas['AutomationDuration']

/** Names of things steps point at, from the options endpoint. */
export interface Refs {
  lists: Map<string, string>
  forms: Map<string, string>
  emails: Map<string, { name: string; latestVersion: number; subject: string }>
  providers: Map<string, { name: string; short: string; status: string }>
  fields: Map<string, { label: string; type: string }>
  /** Polls and surveys the trigger "Answered a poll or survey" names ("poll:<id>" → name). */
  polls?: Map<string, string>
  /** Products the trigger and goal "Bought a product" name (id → name). */
  products?: Map<string, string>
  /** Grow's referral milestones: "3 referrals · The Acme Field Guide". */
  milestones?: Map<string, string>
  /** Stores the store triggers watch (id → name; X20). */
  stores?: Map<string, string>
}

export const emptyRefs: Refs = { lists: new Map(), forms: new Map(), emails: new Map(), providers: new Map(), fields: new Map(), milestones: new Map() }

export function refsFrom(o: Schemas['AutomationOptions'] | undefined): Refs {
  if (!o) return emptyRefs
  return {
    lists: new Map(o.lists.map((l) => [l.id, l.name])),
    forms: new Map(o.forms.map((f) => [f.id, f.slug])),
    emails: new Map(o.emails.map((e) => [e.id, { name: e.name, latestVersion: e.latestVersion, subject: e.subject }])),
    providers: new Map(o.providers.map((p) => [p.id, { name: p.name, short: p.short, status: p.status }])),
    fields: new Map(o.fields.map((f) => [f.key, { label: f.label, type: f.type }])),
    // Digital products (X19): published first, then others that sold.
    products: new Map((o.products ?? []).map((p) => [p.id, p.name])),
    milestones: new Map(o.milestones.map((x) => [x.id, x.name])),
  }
}

const typeNames: Record<StepType, MessageDescriptor> = {
  trigger: mg.typeTrigger,
  delay: mg.typeDelay,
  email: mg.typeEmail,
  branch: mg.typeBranch,
  filter: mg.typeFilter,
  split: mg.typeSplit,
  webhook: mg.typeWebhook,
  list: mg.typeList,
  goal: mg.typeGoal,
  exit: mg.typeExit,
  sms: mg.typeSms,
  push: mg.typePush,
  whatsapp: mg.typeWhatsapp,
}

/** The palette's name for a step type: "Send email", "A/B split". */
export function typeName(type: StepType): string {
  return tt(typeNames[type])
}

/** The trigger kinds, in the palette's and the "Starts when" select's order. */
export const TRIGGER_KINDS: readonly TriggerConfig['kind'][] = [
  'joined_list',
  'form_submitted',
  'tag_added',
  'api_event',
  'date_field',
  'score_crosses',
  'poll_answered',
  // Digital products (X19).
  'bought_product',
  // Grow (referral milestones).
  'milestone_reached',
  // Shopify and WooCommerce (X20).
  'order_placed',
  'first_order',
  'abandoned_checkout',
]

/** A trigger kind's name: "Joined list", "Form submitted". */
export function triggerKindLabel(kind: TriggerConfig['kind']): string {
  switch (kind) {
    case 'joined_list':
      return tt(mg.kindJoinedList)
    case 'form_submitted':
      return tt(mg.kindFormSubmitted)
    case 'tag_added':
      return tt(mg.kindTagAdded)
    case 'api_event':
      return tt(mg.kindApiEvent)
    case 'date_field':
      return tt(mg.kindDateField)
    case 'score_crosses':
      return scoreTriggerName()
    case 'poll_answered':
      return tt(mg.kindPollAnswered)
    case 'bought_product':
      return boughtTriggerName()
    case 'milestone_reached':
      return milestoneTriggerName()
    case 'order_placed':
      return tt(mg.kindOrderPlaced)
    case 'first_order':
      return tt(mg.kindFirstOrder)
    case 'abandoned_checkout':
      return tt(mg.kindAbandonedCheckout)
  }
  return kind
}

/** A trigger kind in a step's header: "joined list" (shown in capitals). */
export function triggerKindWord(kind: string): string {
  return tt(mg.kindWord, { kind })
}

/** Colour of a step type: the palette swatch and the node header. */
export const typeColors: Record<StepType, string> = {
  trigger: '#0F766E',
  delay: '#6B7280',
  email: '#4a3aa7',
  branch: '#B45309',
  filter: '#B45309',
  split: '#B45309',
  webhook: '#1D4ED8',
  list: '#1D4ED8',
  goal: '#0B1220',
  exit: '#0B1220',
  sms: '#0F766E',
  push: '#0F766E',
  whatsapp: '#0F766E',
}

/** Node border colour on the canvas (the default border otherwise). */
export function borderColor(t: StepType): string | undefined {
  switch (t) {
    case 'trigger':
      return '#0F766E'
    case 'branch':
    case 'split':
      return '#F59E0B'
    case 'goal':
    case 'exit':
      return '#0B1220'
  }
  return undefined
}

/** Steps drawn with the compact padding (7px 11px). */
export function isCompact(t: StepType): boolean {
  return t === 'delay' || t === 'split' || t === 'goal' || t === 'exit'
}

export const isFork = (t: StepType) => t === 'branch' || t === 'split'
export const isTerminal = (t: StepType) => t === 'goal' || t === 'exit'

/** "10 minutes", "1 day". */
export function durationText(d?: FlowDuration): string {
  if (!d) return ''
  return tt(mg.duration, { unit: d.unit, n: d.amount })
}

export function durationMs(d?: FlowDuration): number {
  if (!d) return 0
  const m = { minutes: 60_000, hours: 3_600_000, days: 86_400_000, weeks: 604_800_000 }[d.unit] ?? 86_400_000
  return d.amount * m
}

/** `Welcome · step 1 "You're in"` → `Welcome · step 1`. */
export function emailPrefix(name: string): string {
  const i = name.search(/["“]/)
  const p = (i >= 0 ? name.slice(0, i) : name).trim()
  return p.replace(/[\s·]+$/, '')
}

/** The quoted part of an email name, without quotes. */
export function emailQuoted(name: string): string {
  const m = /["“]([^"”]*)["”]?/.exec(name)
  return m ? m[1]! : ''
}

/** `Welcome · step 1` → `step 1`. */
export function emailShortRef(name: string): string {
  let p = emailPrefix(name)
  const i = p.lastIndexOf(' · ')
  if (i >= 0) p = p.slice(i + 3)
  return p || name.replace(/["“”]/g, '').trim()
}

/** Lower-cases ordinary words only: "Step 2" → "step 2", "Gửi email" → "gửi email", "A/B" stays. */
export function lowerFirst(s: string): string {
  if (/^\p{Lu}\p{Ll}/u.test(s)) return s[0]!.toLowerCase() + s.slice(1)
  return s
}

export function capFirst(s: string): string {
  return s ? s[0]!.toUpperCase() + s.slice(1) : s
}

export function nodeById(g: FlowGraph, id: string): FlowNode | undefined {
  return g.nodes.find((n) => n.id === id)
}

export function outEdges(g: FlowGraph, id: string): FlowEdge[] {
  return g.edges.filter((e) => e.source === id)
}

export function inEdges(g: FlowGraph, id: string): FlowEdge[] {
  return g.edges.filter((e) => e.target === id)
}

function emailName(n: FlowNode, refs: Refs): string {
  const id = n.data.email?.emailId
  return id ? (refs.emails.get(id)?.name ?? '') : ''
}

/** How text refers to a step: an email by its prefix ("Step 2"). */
export function nodeRef(g: FlowGraph, n: FlowNode, refs: Refs): string {
  if (n.data.label) return n.data.label
  if (n.type === 'email') {
    const name = emailName(n, refs)
    return emailPrefix(name) || name || typeName('email')
  }
  return nodeTitle(g, n, refs)
}

/** Lower-case reference for running text ("step 1"). */
export function inlineRef(g: FlowGraph, id: string, refs: Refs): string {
  const n = nodeById(g, id)
  if (!n) return tt(mg.aRemovedStep)
  if (n.type === 'email' && !n.data.label) {
    const name = emailName(n, refs)
    if (name) return lowerFirst(emailShortRef(name))
  }
  return lowerFirst(nodeRef(g, n, refs))
}

export function triggerRef(t: TriggerConfig | undefined, refs: Refs): string {
  if (!t) return ''
  switch (t.kind) {
    case 'joined_list':
      return refs.lists.get(t.listId ?? '') ?? tt(mg.aList)
    case 'form_submitted':
      return refs.forms.get(t.formId ?? '') ?? tt(mg.aForm)
    case 'tag_added':
      return t.tag ?? ''
    case 'api_event':
      return t.event ?? ''
    case 'date_field':
      return refs.fields.get(t.field ?? '')?.label ?? t.field ?? ''
    case 'score_crosses':
      return scoreTriggerRef(t)
    case 'poll_answered':
      return refs.polls?.get(t.poll ?? '') ?? tt(mg.aPollOrSurvey)
    case 'bought_product':
      return boughtRef(t.productId, refs.products?.get(t.productId ?? ''))
    case 'milestone_reached':
      return milestoneTriggerRef(t, refs.milestones)
    case 'order_placed':
    case 'first_order':
    case 'abandoned_checkout':
      return refs.stores?.get(t.store ?? '') ?? tt(mg.aStore)
  }
}

export function conditionPhrase(g: FlowGraph, c: FlowCondition | undefined, refs: Refs): string {
  if (!c) return ''
  // Phase 8: texts.
  if (c.kind === 'sms_consent' || c.kind === 'sms_clicked' || c.kind === 'sms_replied') return smsPhrase(g, c, refs)
  const duration = c.within && c.within.amount > 0 ? durationText(c.within) : ''
  const step = c.step ? inlineRef(g, c.step, refs) : tt(mg.anyEmail)
  switch (c.kind) {
    case 'opened':
      return duration ? tt(mg.condOpenedWithin, { step, duration }) : tt(mg.condOpened, { step })
    case 'clicked':
      if (c.urlContains) return duration ? tt(mg.condClickedUrlWithin, { url: c.urlContains, duration }) : tt(mg.condClickedUrl, { url: c.urlContains })
      return duration ? tt(mg.condClickedInWithin, { step, duration }) : tt(mg.condClickedIn, { step })
    case 'tag':
      return tt(mg.condTagged, { tag: c.tag ?? '' })
    case 'list':
      return tt(mg.condInList, { list: refs.lists.get(c.listId ?? '') ?? tt(mg.aList) })
    case 'event':
      return tt(mg.condSentEvent, { event: c.event ?? '' })
    case 'placed_order':
      return tt(mg.condPlacedOrder)
    case 'field': {
      const field = c.field ?? ''
      const value = c.value ?? ''
      switch (c.op) {
        case 'is_set':
          return tt(mg.condFieldSet, { field })
        case 'is_not_set':
          return tt(mg.condFieldEmpty, { field })
        case 'not_equals':
          return tt(mg.condFieldNot, { field, value })
        case 'contains':
          return tt(mg.condFieldContains, { field, value })
        case 'is_one_of':
          return isOneOfPhrase(field, value)
      }
      return tt(mg.condFieldIs, { field, value })
    }
    // Digital products (X19).
    case 'bought_product':
      return boughtPhrase(c.productId, refs.products?.get(c.productId ?? ''))
  }
  return ''
}

export function goalTitle(c: FlowCondition | undefined, refs: Refs): string {
  if (!c) return tt(mg.goalEnd)
  if (c.kind === 'sms_consent' || c.kind === 'sms_clicked' || c.kind === 'sms_replied') return capFirst(smsPhrase({ nodes: [], edges: [] }, c, refs))
  switch (c.kind) {
    case 'clicked':
      return c.urlContains ? tt(mg.goalClickedUrl, { url: c.urlContains }) : tt(mg.goalClickedAny)
    case 'opened':
      return tt(mg.goalOpened)
    case 'tag':
      return tt(mg.goalTagged, { tag: c.tag ?? '' })
    case 'list':
      return tt(mg.goalJoined, { list: refs.lists.get(c.listId ?? '') ?? tt(mg.aList) })
    case 'event':
      return tt(mg.goalSent, { event: c.event ?? '' })
    case 'field':
      return capFirst(conditionPhrase({ nodes: [], edges: [] }, c, refs))
    // Digital products (X19).
    case 'bought_product':
      return boughtGoalTitle(c.productId, refs.products?.get(c.productId ?? ''))
    // Shopify and WooCommerce (X20).
    case 'placed_order':
      return tt(mg.goalPlacedOrder)
  }
  return tt(mg.goalFallback)
}

/** A condition on texts: "has SMS consent for Acme Summit 2026", "clicked a link in Day-before reminder". */
function smsPhrase(g: FlowGraph, c: FlowCondition, refs: Refs): string {
  const text = c.step && g.nodes.some((n) => n.id === c.step) ? inlineRef(g, c.step, refs) : tt(smsStep.phraseAnyText)
  let p: string
  switch (c.kind) {
    case 'sms_consent':
      return c.listId ? tt(smsStep.phraseConsent, { list: refs.lists.get(c.listId) ?? tt(smsStep.aList) }) : tt(smsStep.phraseConsentAny)
    case 'sms_clicked':
      p = tt(smsStep.phraseClicked, { text })
      break
    default:
      p = tt(smsStep.phraseReplied, { text })
  }
  return c.within && c.within.amount > 0 ? tt(smsStep.phraseWithin, { phrase: p, duration: durationText(c.within) }) : p
}

/** The bold line on a canvas node. */
export function nodeTitle(g: FlowGraph, n: FlowNode, refs: Refs): string {
  const d = n.data
  switch (n.type) {
    case 'trigger': {
      if (!d.trigger?.kind) return tt(mg.pickTrigger)
      // "Acme Merch · after 1 hour" (X20).
      if (d.trigger.kind === 'abandoned_checkout') return tt(mg.checkoutAfter, { store: triggerRef(d.trigger, refs), hours: d.trigger.afterHours || 1 })
      return triggerRef(d.trigger, refs) || triggerKindLabel(d.trigger.kind)
    }
    case 'delay':
      return d.delay ? tt(mg.waitFor, { duration: durationText(d.delay) }) : tt(mg.wait)
    case 'email':
      return emailName(n, refs) || tt(mg.pickEmail)
    case 'branch': {
      if (d.label) return d.label
      const p = conditionPhrase(g, d.condition, refs)
      return p ? tt(mg.branchTitle, { condition: capFirst(p) }) : tt(mg.setCondition)
    }
    case 'filter': {
      const p = conditionPhrase(g, d.condition, refs)
      return p ? tt(mg.filterTitle, { condition: p }) : tt(mg.setCondition)
    }
    case 'split': {
      const vs = d.variants ?? []
      const weights = vs.map((v) => v.weight).join(' / ')
      const first = vs[0] ? g.edges.find((e) => e.source === n.id && e.data.path === vs[0]!.key) : undefined
      return first ? tt(mg.splitOn, { weights, step: inlineRef(g, first.target, refs) }) : weights
    }
    case 'sms':
      return d.label || (d.sms?.body ? tt(mg.smsTitle) : tt(mg.writeText))
    // The canvas shows what goes out: the notification's title, the template (AutomationChannels).
    case 'push':
      return d.push?.title || d.label || tt(mg.writeNotification)
    case 'whatsapp':
      return d.whatsapp?.mode === 'text' ? d.label || (d.whatsapp.text ? tt(mg.freeText) : tt(mg.writeMessage)) : d.whatsapp?.template || d.label || tt(mg.pickTemplate)
    case 'webhook':
      return d.webhook?.name || (d.webhook?.url ? hostOf(d.webhook.url) : tt(mg.setUrl))
    case 'list': {
      const a = d.action
      if (!a) return tt(mg.pickAction)
      const list = refs.lists.get(a.listId ?? '') ?? tt(mg.aList)
      switch (a.kind) {
        case 'add_list':
          return tt(mg.addToList, { list })
        case 'remove_list':
          return tt(mg.removeFromList, { list })
        case 'add_tag':
          return tt(mg.addTag, { tag: a.tag ?? '' })
        case 'remove_tag':
          return tt(mg.removeTag, { tag: a.tag ?? '' })
        case 'unsubscribe_all':
          return tt(mg.unsubscribeAll)
      }
      return tt(mg.pickAction)
    }
    case 'goal':
      return goalTitle(d.condition, refs)
    case 'exit':
      return d.label || tt(mg.exitDefault)
  }
}

/** Configuration line under the title ("double opt-in confirmed only"). */
export function nodeDetail(n: FlowNode): string {
  const d = n.data
  switch (n.type) {
    case 'trigger':
      if (d.trigger?.kind === 'joined_list' && d.trigger.confirmedOnly) return tt(mg.confirmedOnly)
      if (d.trigger?.kind === 'api_event') return tt(mg.sentThroughApi)
      if (d.trigger?.kind === 'date_field') {
        const off = d.trigger.offsetDays ?? 0
        return tt(mg.dateDetail, { when: off === 0 ? 'on' : off < 0 ? 'before' : 'after', days: Math.abs(off), yearly: d.trigger.yearly ? 'yes' : 'no', time: d.trigger.at || '09:00' })
      }
      if (d.trigger?.kind === 'score_crosses') return scoreTriggerDetail(d.trigger)
      if (d.trigger?.kind === 'milestone_reached') return milestoneTriggerDetail()
      return ''
    case 'branch':
      if (d.condition?.kind === 'opened' && d.condition.humanOnly) return tt(mg.humanOnly)
      return ''
    case 'webhook':
      return d.webhook?.url ? `${d.webhook.method || 'POST'} ${hostOf(d.webhook.url)}` : ''
  }
  return ''
}

export function hostOf(raw: string): string {
  const s = raw.replace(/^https?:\/\//, '')
  const i = s.search(/[/?#]/)
  return i >= 0 ? s.slice(0, i) : s
}

/** The trigger column: "Joined list · Weekly Digest". */
export function triggerLabel(t: TriggerConfig | undefined, refs: Refs): string {
  if (!t?.kind) return tt(mg.noTrigger)
  const ref = triggerRef(t, refs)
  return ref ? `${triggerKindLabel(t.kind)} · ${ref}` : triggerKindLabel(t.kind)
}

/** Paths a branch or split must connect. */
export function nodePaths(n: FlowNode): string[] {
  if (n.type === 'branch') return ['yes', 'no']
  if (n.type === 'split') return (n.data.variants ?? []).map((v) => v.key)
  return []
}

/** "Yes", "No", or the variant's key ("A"). */
export function pathLabel(p: string): string {
  return p === 'yes' ? tt(mg.pathYes) : p === 'no' ? tt(mg.pathNo) : p
}

/** A whole percent in the region format: "61%". */
export function wholePct(v: number): string {
  return pct(Math.round(v), 0)
}

/** Breadth-first depth from the trigger (longest path), for the entrance. */
export function depths(g: FlowGraph): Map<string, number> {
  const d = new Map<string, number>()
  const indeg = new Map<string, number>()
  for (const n of g.nodes) indeg.set(n.id, 0)
  for (const e of g.edges) if (indeg.has(e.target) && indeg.has(e.source)) indeg.set(e.target, indeg.get(e.target)! + 1)
  const queue = g.nodes.filter((n) => indeg.get(n.id) === 0).map((n) => n.id)
  for (const id of queue) d.set(id, 0)
  while (queue.length) {
    const id = queue.shift()!
    for (const e of g.edges) {
      if (e.source !== id || !indeg.has(e.target)) continue
      d.set(e.target, Math.max(d.get(e.target) ?? 0, (d.get(id) ?? 0) + 1))
      indeg.set(e.target, indeg.get(e.target)! - 1)
      if (indeg.get(e.target) === 0) queue.push(e.target)
    }
  }
  for (const n of g.nodes) if (!d.has(n.id)) d.set(n.id, 0)
  return d
}

/** True when target can already reach source (a new edge would loop). */
export function wouldLoop(g: FlowGraph, source: string, target: string): boolean {
  if (source === target) return true
  const seen = new Set<string>()
  const stack = [target]
  while (stack.length) {
    const id = stack.pop()!
    if (id === source) return true
    if (seen.has(id)) continue
    seen.add(id)
    for (const e of g.edges) if (e.source === id) stack.push(e.target)
  }
  return false
}

export interface EdgeGeometry {
  path: string
  length: number
  /** Bus line height for forks and merges (undefined for straight edges). */
  busY?: number
  /** Label position (top-left) for branch and split paths. */
  label?: { x: number; y: number }
}

/**
 * Orthogonal edges like the board: a straight drop when the steps line up,
 * otherwise down from the source, across on a bus line, and down into the
 * target. sx/sy and tx/ty are the handle points (5px outside each step).
 */
export function edgeGeometry(sx: number, sy: number, tx: number, ty: number, kind: 'branch' | 'split' | 'merge' | 'plain'): EdgeGeometry {
  if (Math.abs(sx - tx) < 0.5) {
    return { path: `M${sx},${sy} L${tx},${ty}`, length: Math.abs(ty - sy) }
  }
  // The board draws a bus as a 2px bar whose top is 24px (branch) or 22px
  // (split) under the step, or 18px above the merge target: its centre line
  // is 1px lower.
  let busY: number
  // Steps have fractional heights; the bars sit on whole pixels like the board's.
  if (kind === 'branch') busY = Math.round(sy - 5) + 25
  else if (kind === 'split') busY = Math.round(sy - 5) + 23
  else if (kind === 'merge') busY = Math.round(ty + 5) - 17
  else busY = (sy + ty) / 2
  busY = Math.min(Math.max(busY, Math.min(sy, ty)), Math.max(sy, ty))
  const length = Math.abs(busY - sy) + Math.abs(tx - sx) + Math.abs(ty - busY)
  const out: EdgeGeometry = { path: `M${sx},${sy} L${sx},${busY} L${tx},${busY} L${tx},${ty}`, length, busY }
  if (kind === 'branch' || kind === 'split') out.label = { x: tx + (kind === 'branch' ? 14 : 8), y: busY - 12 }
  return out
}

/** Speed of a pulse along an edge (px/s) and its length (px), as on the board. */
export const PULSE_SPEED = 220
export const PULSE_LEN = 18
/** Time a pulse rests in a step before it leaves (s). */
export const DWELL = 0.45
/** Node glow: ramp, hold, fade (s). */
export const HIT = { ramp: 0.081, hold: 0.29, fade: 0.2 }
/** Handle flash: grow, hold, shrink (s). */
export const FLASH = { grow: 0.061, hold: 0.16, shrink: 0.18 }

export interface Box {
  x: number
  y: number
  w: number
  h: number
}

export interface Pulse {
  key: string
  horizontal: boolean
  /** Centre of the bar at the start and the end of its run (flow coordinates). */
  from: { x: number; y: number }
  to: { x: number; y: number }
  start: number
  end: number
  opacity: number
}

export interface Timeline {
  period: number
  pulses: Pulse[]
  hits: { node: string; start: number; opacity: number }[]
  flashIn: { node: string; start: number }[]
  flashOut: { node: string; start: number }[]
}

export function edgeKind(g: FlowGraph, e: FlowEdge): 'branch' | 'split' | 'merge' | 'plain' {
  const src = nodeById(g, e.source)
  if (src?.type === 'branch') return 'branch'
  if (src?.type === 'split') return 'split'
  if (inEdges(g, e.target).length > 1) return 'merge'
  return 'plain'
}

/**
 * The pulse-through-steps wave: a token rests in each step, flashes its
 * handles, then runs along every outgoing edge at a constant speed. Paths
 * that carry fewer contacts (a branch's minor path) pulse at 55% opacity.
 * One wave lasts until the last step glows; the wave repeats each period.
 */
export function timeline(g: FlowGraph, boxes: Map<string, Box>, shares?: Map<string, Map<string, number>>): Timeline {
  const out: Timeline = { period: 0, pulses: [], hits: [], flashIn: [], flashOut: [] }
  const trigger = g.nodes.find((n) => n.type === 'trigger')
  if (!trigger || !boxes.has(trigger.id)) return out
  const order = [...depths(g).entries()].sort((a, b) => a[1] - b[1]).map(([id]) => id)
  const arrival = new Map<string, number>([[trigger.id, 0]])
  const strength = new Map<string, number>([[trigger.id, 1]])
  let last = 0
  for (const id of order) {
    const t = arrival.get(id)
    const box = boxes.get(id)
    const node = nodeById(g, id)
    if (t === undefined || !box || !node) continue
    const op = strength.get(id) ?? 1
    out.hits.push({ node: id, start: t, opacity: op })
    if (node.type !== 'trigger') out.flashIn.push({ node: id, start: t })
    last = Math.max(last, t + HIT.ramp + HIT.hold + HIT.fade)
    const edges = outEdges(g, id)
    if (edges.length === 0) continue
    const leave = t + DWELL
    out.flashOut.push({ node: id, start: leave })
    // The minor paths of a branch pulse fainter.
    const share = shares?.get(id)
    const top = share ? Math.max(...share.values()) : 0
    for (const e of edges) {
      const tb = boxes.get(e.target)
      if (!tb) continue
      let eop = op
      if (share && share.size > 1 && (share.get(e.data.path ?? '') ?? 0) < top) eop = Math.min(op, 0.55)
      const sx = box.x + box.w / 2
      const sy = box.y + box.h
      const tx = tb.x + tb.w / 2
      const ty = tb.y
      const kind = edgeKind(g, e)
      const segs: [number, number, number, number][] = []
      if (Math.abs(sx - tx) < 0.5) segs.push([sx, sy, tx, ty])
      else {
        const geo = edgeGeometry(sx, sy + 5, tx, ty - 5, kind)
        // The board's pulses run along the top of the bus bar.
        const busY = geo.busY! - 1
        segs.push([sx, sy, sx, busY], [sx, busY, tx, busY], [tx, busY, tx, ty])
      }
      let clock = leave
      segs.forEach(([x1, y1, x2, y2], i) => {
        const horizontal = y1 === y2
        const len = Math.hypot(x2 - x1, y2 - y1)
        const dirX = Math.sign(x2 - x1)
        const dirY = Math.sign(y2 - y1)
        const half = PULSE_LEN / 2
        const from = { x: x1 - dirX * half, y: y1 - dirY * half }
        const to = { x: x2 + dirX * half, y: y2 + dirY * half }
        const dur = (len + PULSE_LEN) / PULSE_SPEED
        out.pulses.push({ key: `${e.id}:${i}`, horizontal, from, to, start: clock, end: clock + dur, opacity: eop })
        clock += dur
      })
      const prev = arrival.get(e.target)
      arrival.set(e.target, prev === undefined ? clock : Math.max(prev, clock))
      strength.set(e.target, Math.max(strength.get(e.target) ?? 0, eop))
    }
  }
  out.period = Math.max(4, Math.ceil(last + 0.5))
  return out
}

/** Entrance delays (s): steps 150ms apart by depth, edges after their source, labels last. */
export function entrance(g: FlowGraph): { nodes: Map<string, number>; edges: Map<string, number>; labels: Map<string, number> } {
  const d = depths(g)
  const nodeDelay = (id: string) => 0.1 + 0.15 * (d.get(id) ?? 0)
  const nodes = new Map(g.nodes.map((n) => [n.id, nodeDelay(n.id)]))
  const edges = new Map<string, number>()
  const labels = new Map<string, number>()
  for (const e of g.edges) {
    const kind = edgeKind(g, e)
    edges.set(e.id, nodeDelay(e.source) + (kind === 'plain' ? 0.1 : 0.07))
    labels.set(e.id, nodeDelay(e.source) + 0.45)
  }
  return { nodes, edges, labels }
}

/** Keeps only what the API stores from React Flow's nodes and edges. */
export function serialize(nodes: { id: string; type?: string; position: { x: number; y: number }; data: unknown }[], edges: { id: string; source: string; target: string; data?: unknown }[]): FlowGraph {
  return {
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.type as StepType,
      position: { x: Math.round(n.position.x), y: Math.round(n.position.y) },
      data: stripData(n.data as NodeData),
    })),
    edges: edges.map((e) => {
      const path = (e.data as { path?: string } | undefined)?.path
      return { id: e.id, source: e.source, target: e.target, data: path ? { path } : {} }
    }),
  }
}

function stripData(d: NodeData): NodeData {
  if (!d.webhook) return d
  const w = { ...d.webhook } as Record<string, unknown>
  delete w.hasSecret
  return { ...d, webhook: w as NodeData['webhook'] }
}

let seq = 0
/** A short step id unique within the graph ("n14"). */
export function nextNodeId(g: FlowGraph): string {
  let max = 0
  for (const n of g.nodes) {
    const m = /^n(\d+)$/.exec(n.id)
    if (m) max = Math.max(max, Number(m[1]))
  }
  let id = `n${max + 1}`
  while (g.nodes.some((n) => n.id === id)) id = `n${max + 1 + ++seq}`
  return id
}

/** Default settings for a step dropped from the palette. */
export function newStepData(type: StepType, trigger?: TriggerConfig['kind']): NodeData {
  switch (type) {
    case 'trigger':
      if (trigger === 'score_crosses') return { trigger: { kind: trigger, ...scoreTriggerDefaults(), reentry: 'once' } }
      return { trigger: { kind: trigger ?? 'joined_list', confirmedOnly: trigger === 'joined_list' || !trigger ? true : undefined, reentry: 'once' } }
    case 'delay':
      return { delay: { amount: 1, unit: 'days' } }
    case 'email':
      return { email: { sendWindow: { enabled: true, start: '07:00', end: '21:00' }, skipRecentHours: 4 } }
    case 'branch':
      return { condition: { kind: 'opened', within: { amount: 2, unit: 'days' }, humanOnly: true } }
    case 'filter':
      return { condition: { kind: 'tag' } }
    case 'split':
      return { variants: [{ key: 'A', weight: 50 }, { key: 'B', weight: 50 }] }
    case 'webhook':
      return { webhook: { method: 'POST' } }
    case 'list':
      return { action: { kind: 'add_tag' } }
    case 'goal':
      return { condition: { kind: 'clicked' } }
    case 'exit':
      return {}
    case 'sms':
      return { sms: { outside: 'wait' } }
    case 'push':
      return { push: { outside: 'wait', expiryMinutes: 240 } }
    case 'whatsapp':
      return { whatsapp: { mode: 'template', outside: 'wait', outsideWindow: 'skip', variables: [] } }
  }
}
