import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { count, pct } from '@/lib/format'
import { m } from './builder.messages'
import { severalSteps, syncSteps } from './steps'

/**
 * A form's A/B test as the builder edits it (M60): variant A is the form
 * itself; B and C change its copy, steps and popup triggers. The test
 * (its variants, split and rule) is saved with the form on Save & publish.
 */

export type ABResult = Schemas['SignupFormABResult']
export type ABInput = Schemas['SignupFormABInput']
export type VariantKey = Schemas['SignupFormABVariant']['key']
export type VariantContent = Schemas['SignupFormVariantContent']
type FormContent = Schemas['SignupFormContent']

export interface VariantDraft {
  key: VariantKey
  weight: number
  /** What B or C changes; none for A. */
  content?: VariantContent
}

export interface ABDraft {
  /** None: no test. */
  variants: VariantDraft[]
  autoEnd: boolean
  minViews: number
  /** 0.9, 0.95 or 0.99 */
  confidence: number
}

export const variantKeys: readonly VariantKey[] = ['A', 'B', 'C']
export const confidences = [0.9, 0.95, 0.99] as const

/** What a variant carries of the form's content: its copy, steps and popup triggers. */
export function variantContentOf(c: FormContent | VariantContent): VariantContent {
  return {
    heading: c.heading,
    description: c.description,
    button: c.button,
    afterSubmitText: c.afterSubmitText,
    steps: c.steps?.map((s) => ({ ...s, fields: [...s.fields] })),
    popupDelay: c.popupDelay,
    popupScroll: c.popupScroll,
    popupExit: c.popupExit,
  }
}

/** The draft of a running test, or of none (with the server's default rule). */
export function abDraftOf(r: ABResult | undefined): ABDraft {
  const running = r?.status === 'running'
  return {
    variants: running ? r.variants.map((v) => ({ key: v.key, weight: v.weight, content: v.key === 'A' || !v.content ? undefined : variantContentOf(v.content) })) : [],
    autoEnd: r?.autoEnd ?? false,
    minViews: r?.minViews ?? 500,
    confidence: r?.targetConfidence ?? 0.95,
  }
}

/** An even split that adds up to 100: 50/50, 34/33/33. */
export function evenSplit(n: number): number[] {
  const base = Math.floor(100 / n)
  return Array.from({ length: n }, (_, i) => base + (i < 100 - base * n ? 1 : 0))
}

/**
 * The test with one more variant, a copy of A's copy, steps and popup
 * triggers (A, B on a form without a test), split evenly.
 */
export function addVariant(d: ABDraft, a: FormContent): { draft: ABDraft; key: VariantKey } {
  const current: VariantDraft[] = d.variants.length ? d.variants : [{ key: 'A', weight: 100 }]
  const key = variantKeys[current.length] ?? 'C'
  const next = [...current, { key, weight: 0, content: variantContentOf(a) }]
  const split = evenSplit(next.length)
  return { draft: { ...d, variants: next.map((v, i) => ({ ...v, weight: split[i]! })) }, key }
}

/** "50/50", "34/33/33" */
export function splitText(variants: readonly { weight: number }[]): string {
  return variants.map((v) => v.weight).join('/')
}

/** Whether two tests have the same variants and split (a change restarts the count). */
export function sameVariants(a: readonly VariantDraft[], b: readonly VariantDraft[]): boolean {
  return JSON.stringify(a.map(normalVariant)) === JSON.stringify(b.map(normalVariant))
}

function normalVariant(v: VariantDraft) {
  const c = v.content
  return {
    key: v.key,
    weight: v.weight,
    content: c
      ? { heading: c.heading, description: c.description, button: c.button, afterSubmitText: c.afterSubmitText || '', steps: severalSteps(c.steps) ? c.steps.map((s) => ({ id: s.id, fields: s.fields, heading: s.heading || '', description: s.description || '', button: s.button || '' })) : [], popupDelay: c.popupDelay, popupScroll: c.popupScroll, popupExit: c.popupExit === true }
      : null,
  }
}

/** Whether a test's draft differs from what's saved. */
export function sameAB(a: ABDraft, b: ABDraft): boolean {
  return sameVariants(a.variants, b.variants) && a.autoEnd === b.autoEnd && a.minViews === b.minViews && a.confidence === b.confidence
}

/** The test to save; each variant's steps follow the form's fields. */
export function abInputOf(d: ABDraft, fieldKeys: readonly string[]): ABInput {
  return {
    variants: d.variants.map((v) => {
      if (v.key === 'A' || !v.content) return { key: v.key, weight: v.weight }
      return { key: v.key, weight: v.weight, content: { ...v.content, steps: syncSteps(v.content.steps, fieldKeys) } }
    }),
    autoEnd: d.autoEnd,
    minViews: d.minViews,
    confidence: d.confidence,
  }
}

/**
 * Where a running test stands, in one sentence: "B gets 4.1% vs 3.2% for
 * A (+28%), 97% confidence.", "Not significant yet." or "Too few views
 * yet: 212 of 500 per variant."
 */
export function abSentence(r: ABResult): string {
  if (r.outcome === 'winner' && r.leader && r.runnerUp) {
    const rate = (key: string) => pct(r.variants.find((v) => v.key === key)?.rate ?? 0)
    return t(m.abWinner, { leader: r.leader, leaderRate: rate(r.leader), runnerRate: rate(r.runnerUp), runnerUp: r.runnerUp, lift: `+${pct(r.lift, 0)}`, confidence: pct(r.confidence, 0) })
  }
  if (r.outcome === 'too_small') return t(m.abTooSmall, { n: count(r.fewestViews), min: count(r.minViews) })
  return t(m.abNotSignificant)
}
