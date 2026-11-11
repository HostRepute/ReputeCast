import type { Schemas } from '@/api/client'

/**
 * Multi-step forms and choice questions (M60): the steps' bookkeeping the
 * builder does as fields come and go, and a choice question's key and
 * tags. The server checks the same rules on save (steps.go).
 */

export type FormStep = Schemas['SignupFormStep']
export type FormField = Schemas['SignupFormField']
export type ChoiceOption = Schemas['SignupFormChoiceOption']

/** The most steps a form can have. */
export const maxSteps = 4
/** A choice question's answers: at least 2, at most 12. */
export const minOptions = 2
export const maxOptions = 12

const stepIds = ['s1', 's2', 's3', 's4']

/** Whether a form shows several steps: fewer than two is one step. */
export function severalSteps(steps: FormStep[] | undefined): steps is FormStep[] {
  return (steps?.length ?? 0) >= 2
}

/**
 * Steps kept in line with the form's fields: fields that are gone drop
 * out, a field in two steps stays in the first, and fields no step has
 * join the last one. A step left without fields stays, so it can take
 * fields again (saving asks to fill or remove it). One step is no steps.
 */
export function syncSteps(steps: FormStep[] | undefined, keys: readonly string[]): FormStep[] | undefined {
  if (!severalSteps(steps)) return undefined
  const known = new Set(keys)
  const placed = new Set<string>()
  const out = steps.map((s) => ({
    ...s,
    fields: s.fields.filter((k) => {
      if (!known.has(k) || placed.has(k)) return false
      placed.add(k)
      return true
    }),
  }))
  const missing = keys.filter((k) => !placed.has(k))
  if (missing.length) {
    const last = out[out.length - 1]!
    out[out.length - 1] = { ...last, fields: [...last.fields, ...missing] }
  }
  return out
}

/** "Several steps" from one: the email on the first step, every other field on the second. */
export function splitSteps(keys: readonly string[]): FormStep[] {
  return [
    { id: 's1', fields: keys.slice(0, 1) },
    { id: 's2', fields: keys.slice(1) },
  ]
}

/** A new last step, with the first free id (ids stay with their steps, so translations do). */
export function addStep(steps: FormStep[]): FormStep[] {
  if (steps.length >= maxSteps) return steps
  const id = stepIds.find((x) => !steps.some((s) => s.id === x)) ?? `s${steps.length + 1}`
  return [...steps, { id, fields: [] }]
}

/** Removes a step; its fields join the step before it (the first step's join the next). */
export function removeStep(steps: FormStep[], index: number): FormStep[] {
  const gone = steps[index]
  if (!gone || steps.length <= 2) return steps
  const into = index > 0 ? index - 1 : 1
  return steps
    .map((s, i) => (i === into ? { ...s, fields: index > 0 ? [...s.fields, ...gone.fields] : [...gone.fields, ...s.fields] } : s))
    .filter((_, i) => i !== index)
}

/**
 * Moves a field up or down; past the edge of its step it moves into the
 * neighbouring step (last there going up, first going down).
 */
export function moveField(steps: FormStep[], stepIndex: number, fieldIndex: number, dir: -1 | 1): FormStep[] {
  const out = steps.map((s) => ({ ...s, fields: [...s.fields] }))
  const step = out[stepIndex]
  const key = step?.fields[fieldIndex]
  if (!step || key === undefined) return steps
  const to = fieldIndex + dir
  if (to >= 0 && to < step.fields.length) {
    step.fields[fieldIndex] = step.fields[to]!
    step.fields[to] = key
    return out
  }
  const next = out[stepIndex + dir]
  if (!next) return steps
  step.fields.splice(fieldIndex, 1)
  if (dir < 0) next.fields.push(key)
  else next.fields.unshift(key)
  return out
}

/** Whether a field can move that way: not up from the very first place, nor down from the very last. */
export function canMove(steps: FormStep[], stepIndex: number, fieldIndex: number, dir: -1 | 1): boolean {
  if (dir < 0) return stepIndex > 0 || fieldIndex > 0
  return stepIndex < steps.length - 1 || fieldIndex < (steps[stepIndex]?.fields.length ?? 0) - 1
}

/** The built-in fields' keys: no choice question may take one. */
const builtinKeys = ['email', 'first_name', 'last_name', 'company', 'phone']

/** A new choice question's own key: q1, q2…, never a contact field's or another field's. */
export function nextChoiceKey(fields: readonly FormField[], customKeys: readonly string[]): string {
  const taken = new Set([...builtinKeys, ...customKeys, ...fields.map((f) => f.key)])
  for (let n = 1; ; n++) {
    const key = `q${n}`
    if (!taken.has(key)) return key
  }
}

/** The tag an answer adds, from its label: "Product news" → "product-news". */
export function tagOf(label: string): string {
  return label
    .toLowerCase()
    .normalize('NFKD')
    .replace(/\p{M}/gu, '')
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)
}
