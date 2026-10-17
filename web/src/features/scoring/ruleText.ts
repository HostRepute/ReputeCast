import type { Schemas } from '@/api/client'
import { formatList, t } from '@/i18n'
import { m } from './messages'

export type Rule = Schemas['ScoringRule']
export type Options = Schemas['ScoringOptions']
export type CapKind = Rule['cap']['kind']

/** "+2", "−5": points with a real minus sign. */
export function signed(n: number): string {
  return n > 0 ? `+${n}` : n < 0 ? `−${Math.abs(n)}` : '0'
}

function listName(o: Options | undefined, id: string | undefined): string {
  return o?.lists.find((l) => l.id === id)?.name ?? ''
}

function formName(o: Options | undefined, id: string | undefined): string {
  return o?.forms.find((f) => f.id === id)?.name ?? ''
}


/** What a rule counts, its row's first line: "Click on a link containing acme.io/pricing". */
export function ruleWhen(r: Rule, o?: Options): string {
  switch (r.kind) {
    case 'open':
      return r.listId ? t(m.whenOpenList, { list: listName(o, r.listId) }) : t(m.whenOpen)
    case 'click':
      return r.contains ? t(m.whenClickContains, { text: r.contains }) : t(m.whenClick)
    case 'form': {
      const name = formName(o, r.formId)
      return name ? t(m.whenForm, { form: name }) : t(m.whenFormGone)
    }
    case 'api_event':
      return t(m.whenApiEvent, { event: r.event ?? '' })
    case 'soft_bounce':
      return t(m.whenSoftBounce)
    case 'unsubscribe':
      return r.listId ? t(m.whenUnsubscribeList, { list: listName(o, r.listId) }) : t(m.whenUnsubscribe)
    case 'field': {
      const field = r.field ?? ''
      const values = formatList(r.values ?? [], r.op === 'is_not' ? 'conjunction' : 'disjunction')
      if (r.op === 'is_set') return t(m.whenFieldSet, { field })
      return r.op === 'is_not' ? t(m.whenFieldNot, { field, values }) : t(m.whenField, { field, values })
    }
  }
  return ''
}

/** Parts of the first line to set in code style (the text a link contains, an event, a field). */
export function ruleCode(r: Rule): string | undefined {
  switch (r.kind) {
    case 'click':
      return r.contains || undefined
    case 'api_event':
      return r.event || undefined
    case 'field':
      return r.field || undefined
  }
  return undefined
}

/** The second line: what kind of rule it is and how it counts. */
export function ruleDetail(r: Rule): string {
  switch (r.kind) {
    case 'open':
      return t(m.detailOpen)
    case 'click':
      return r.contains ? t(m.detailClickContains) : t(m.detailClick)
    case 'form':
      return t(m.detailForm)
    case 'api_event':
      return t(m.detailApiEvent)
    case 'soft_bounce':
      return t(m.detailSoftBounce)
    case 'unsubscribe':
      return r.listId ? t(m.detailUnsubscribeList) : t(m.detailUnsubscribe)
    case 'field':
      return t(m.detailField)
  }
  return ''
}

/** The cap column: "6 a week", "Once a month", "While true", "–". */
export function capText(r: Rule): string {
  const p = r.cap.points ?? 0
  switch (r.cap.kind) {
    case 'once':
      return t(m.capOnce)
    case 'once_day':
      return t(m.capOnceDay)
    case 'once_week':
      return t(m.capOnceWeek)
    case 'once_month':
      return t(m.capOnceMonth)
    case 'points_day':
      return t(m.capPointsDay, { points: p })
    case 'points_week':
      return t(m.capPointsWeek, { points: p })
    case 'points_month':
      return t(m.capPointsMonth, { points: p })
    case 'while_true':
      return t(m.capWhileTrue)
  }
  return t(m.capNone)
}

/** The cap in a sentence ("cap 6 a week", "cap once", "no cap"). */
export function capPhrase(r: Rule): string {
  if (r.cap.kind === 'none') return t(m.noCap)
  return t(m.capLower, { cap: capText(r).toLowerCase() })
}

/** The cap after "cap" on the phone list ("cap 6 a week", "cap none"). */
export function capInline(r: Rule): string {
  return r.cap.kind === 'none' ? t(m.capNoneWord) : capText(r).toLowerCase()
}

export const capKinds: CapKind[] = ['none', 'once', 'once_day', 'once_week', 'once_month', 'points_day', 'points_week', 'points_month']

export function capKindLabel(k: CapKind): string {
  switch (k) {
    case 'once':
      return t(m.capKindOnce)
    case 'once_day':
      return t(m.capKindOnceDay)
    case 'once_week':
      return t(m.capKindOnceWeek)
    case 'once_month':
      return t(m.capKindOnceMonth)
    case 'points_day':
      return t(m.capKindPointsDay)
    case 'points_week':
      return t(m.capKindPointsWeek)
    case 'points_month':
      return t(m.capKindPointsMonth)
  }
  return t(m.capKindNone)
}

/** The line under the cap select. */
export function capExplain(r: Rule): string {
  switch (r.cap.kind) {
    case 'none':
      return t(m.capExplainNone)
    case 'once':
      return t(m.capExplainOnce)
    case 'once_day':
      return t(m.capExplainOnceDay)
    case 'once_week':
      return t(m.capExplainOnceWeek)
    case 'once_month':
      return t(m.capExplainOnceMonth)
    case 'points_day':
    case 'points_week':
    case 'points_month':
      return t(m.capExplainPoints, { points: r.cap.points ?? 0, period: r.cap.kind.slice(7) })
  }
  return ''
}

export const ruleKinds: Rule['kind'][] = ['open', 'click', 'form', 'api_event', 'soft_bounce', 'unsubscribe', 'field']

export function kindLabel(k: Rule['kind']): string {
  switch (k) {
    case 'open':
      return t(m.kindOpen)
    case 'click':
      return t(m.kindClick)
    case 'form':
      return t(m.kindForm)
    case 'api_event':
      return t(m.kindApiEvent)
    case 'soft_bounce':
      return t(m.kindSoftBounce)
    case 'unsubscribe':
      return t(m.kindUnsubscribe)
  }
  return t(m.kindField)
}

export function kindPick(k: Rule['kind']): string {
  switch (k) {
    case 'open':
      return t(m.pickOpen)
    case 'click':
      return t(m.pickClick)
    case 'form':
      return t(m.pickForm)
    case 'api_event':
      return t(m.pickApiEvent)
    case 'soft_bounce':
      return t(m.pickSoftBounce)
    case 'unsubscribe':
      return t(m.pickUnsubscribe)
  }
  return t(m.pickField)
}

/** A rule's defaults when its kind is picked. */
export function ruleFor(kind: Rule['kind'], id: string, o?: Options): Rule {
  const base: Rule = { id, kind, points: 5, cap: { kind: 'none' }, enabled: true, values: [] }
  switch (kind) {
    case 'open':
      return { ...base, points: 2, cap: { kind: 'points_week', points: 6 } }
    case 'click':
      return { ...base, points: 5, cap: { kind: 'points_week', points: 20 } }
    case 'form':
      return { ...base, points: 20, formId: o?.forms[0]?.id, cap: { kind: 'once' } }
    case 'api_event':
      return { ...base, points: 25, event: o?.apiEvents[0]?.name ?? '', cap: { kind: 'once' } }
    case 'soft_bounce':
      return { ...base, points: -5 }
    case 'unsubscribe':
      return { ...base, points: -20 }
    case 'field':
      return { ...base, points: 10, field: o?.fields[0]?.key ?? '', op: 'is_one_of', values: [], cap: { kind: 'while_true' } }
  }
  return base
}

/** A problem with a rule before it is sent, keyed by field. */
export function ruleProblems(r: Rule): Partial<Record<'points' | 'contains' | 'form' | 'event' | 'field' | 'values' | 'cap', string>> {
  const out: Partial<Record<'points' | 'contains' | 'form' | 'event' | 'field' | 'values' | 'cap', string>> = {}
  if (!Number.isFinite(r.points) || r.points === 0 || r.points < -50 || r.points > 50) out.points = t(m.errPoints)
  if (r.kind === 'click' && r.contains !== undefined && !r.contains.trim()) out.contains = t(m.errContains)
  if (r.kind === 'form' && !r.formId) out.form = t(m.errForm)
  if (r.kind === 'api_event' && !(r.event ?? '').trim()) out.event = t(m.errEvent)
  if (r.kind === 'field') {
    if (!r.field) out.field = t(m.errField)
    if (r.op !== 'is_set' && !(r.values ?? []).length) out.values = t(m.errValues)
  }
  if (r.cap.kind.startsWith('points_')) {
    const cap = r.cap.points ?? 0
    const pts = Math.abs(r.points)
    if (!cap) out.cap = t(m.errCapPoints)
    else if (cap < pts) out.cap = t(m.errCap, { cap, period: r.cap.kind.slice(7), points: pts, kind: r.kind })
  }
  return out
}
