import type { Editor } from 'grapesjs'
import type { ABTest } from '../context'

/**
 * The parts of an email that live outside the canvas: the subject, the
 * preheader and the A/B test. Undo and redo cover them (CO-8): their
 * changes go on GrapesJS's own undo stack, in order with the canvas edits,
 * so ⌘Z and the Undo button take back whatever was changed last.
 */
export interface Meta {
  subject: string
  preheader: string
  abTest: ABTest | null
}

export interface MetaUndo {
  /**
   * The subject, preheader or A/B test changed. Typing becomes one undo
   * step once it pauses; any other change (a test started, a metric picked)
   * is a step of its own at once. `step` makes typed text a step at once too
   * (a recovered draft, which goes back in one undo with its canvas).
   */
  change(next: Meta, opts?: { step?: boolean }): void
  /** Makes typing that hasn't paused yet its undo step (an undo or redo does this first). */
  flush(): void
  /** Takes values that aren't an edit (a reload from the server): no undo step. */
  reset(next: Meta): void
  dispose(): void
}

interface BackboneModel {
  get(key: string): unknown
  set(attrs: Record<string, unknown>, opts?: Record<string, unknown>): unknown
  on(ev: string, fn: (...args: never[]) => void): unknown
  off(ev?: string, fn?: (...args: never[]) => void): unknown
}

type ModelCtor = new (attrs?: Record<string, unknown>) => BackboneModel

/**
 * Backbone's Model, which GrapesJS bundles without exporting it: the class
 * at the root of the editor model's prototype chain (the one that defines
 * changedAttributes). Its undo manager tracks Backbone models only.
 */
function backboneModel(editor: Editor): ModelCtor {
  let proto: object | null = Object.getPrototypeOf(editor.getModel())
  while (proto && !Object.prototype.hasOwnProperty.call(proto, 'changedAttributes')) proto = Object.getPrototypeOf(proto)
  if (!proto) throw new Error('GrapesJS has no Backbone model to track the subject with')
  return (proto as { constructor: ModelCtor }).constructor
}

const attrs = (m: Meta) => ({ subject: m.subject, preheader: m.preheader, abTest: m.abTest })

/** The A/B test without the text typed into it (the variants' subjects). */
function shape(t: ABTest | null): string {
  if (!t) return ''
  return JSON.stringify({ ...t, variants: t.variants.map((v) => ({ ...v, subject: v.subject === undefined ? undefined : '' })) })
}

/** True when only typed text differs (subject, preheader, a variant's subject). */
export function typedOnly(a: Meta, b: Meta): boolean {
  return shape(a.abTest) === shape(b.abTest)
}

/**
 * Tracks the subject, preheader and A/B test on the editor's undo stack.
 * `apply` gets the values an undo or redo brings back. `pause` is how long
 * typing waits before it becomes an undo step.
 */
export function trackMeta(editor: Editor, initial: Meta, apply: (m: Meta) => void, opts: { pause?: number } = {}): MetaUndo {
  const pause = opts.pause ?? 700
  const Model = backboneModel(editor)
  // `_undo` is how GrapesJS's undo manager knows a model's changes are undoable.
  const model = new Model({ _undo: true, ...attrs(initial) })
  const um = editor.UndoManager
  um.add(model)
  let pending: Meta | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  const read = (): Meta => ({
    subject: String(model.get('subject') ?? ''),
    preheader: String(model.get('preheader') ?? ''),
    abTest: (model.get('abTest') as ABTest | null | undefined) ?? null,
  })
  const stack = um.getStack() as unknown as { length: number; at(i: number): { set(key: string, value: unknown): void } | undefined }
  let own = 0
  /**
   * Makes the pending change an undo step. GrapesJS undoes together what
   * happened in the same moment; `alone` keeps this step apart (typing,
   * then a metric picked before the typing paused, are two steps).
   */
  const commit = (alone = true) => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
    if (!pending) return
    const next = pending
    pending = null
    const last = stack.at(stack.length - 1)
    model.set(attrs(next), { rcLocal: true })
    const added = stack.at(stack.length - 1)
    if (alone && added && added !== last) added.set('magicFusionIndex', `rc-meta-${++own}`)
  }
  // An undo or a redo set the model: the composer shows what it brought back.
  const onChange = (_m: unknown, o?: { rcLocal?: boolean }) => {
    if (o?.rcLocal) return
    apply(read())
  }
  model.on('change', onChange)

  // Typing that hasn't paused yet becomes its step before an undo or a redo
  // (the toolbar's buttons, ⌘Z and ⇧⌘Z all come through here).
  const orig = { undo: um.undo, redo: um.redo, hasUndo: um.hasUndo }
  um.undo = function (this: typeof um, all?: boolean) {
    commit()
    return orig.undo.call(um, all)
  }
  um.redo = function (this: typeof um, all?: boolean) {
    commit()
    return orig.redo.call(um, all)
  }
  um.hasUndo = function (this: typeof um) {
    return pending !== null || orig.hasUndo.call(um)
  }

  return {
    change(next, o = {}) {
      const before = pending ?? read()
      if (!o.step && typedOnly(before, next)) {
        pending = next
        if (timer !== undefined) clearTimeout(timer)
        timer = setTimeout(() => commit(), pause)
        return
      }
      // Typing before it is a step of its own, then this change; a `step`
      // goes with what else happens in the same moment (the canvas it came with).
      commit()
      pending = next
      commit(!o.step)
    },
    flush: () => commit(),
    reset(next) {
      if (timer !== undefined) clearTimeout(timer)
      timer = undefined
      pending = null
      um.skip(() => model.set(attrs(next), { rcLocal: true }))
    },
    dispose() {
      if (timer !== undefined) clearTimeout(timer)
      timer = undefined
      pending = null
      model.off('change', onChange)
      um.undo = orig.undo
      um.redo = orig.redo
      um.hasUndo = orig.hasUndo
      um.remove(model)
    },
  }
}
