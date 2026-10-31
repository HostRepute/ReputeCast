import type { Editor } from 'grapesjs'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { ABTest } from '../context'
import { trackMeta, typedOnly, type Meta } from './metaUndo'

// CO-8: undo and redo cover the subject, the preheader and the A/B test, on
// the same stack as the canvas (a headless GrapesJS editor here).

let editor: Editor

beforeEach(async () => {
  vi.useFakeTimers()
  const { default: grapesjs } = await import('grapesjs')
  editor = grapesjs.init({ headless: true, storageManager: false, telemetry: false, components: '<div>Hello</div>' })
  later()
  editor.UndoManager.clear()
})

afterEach(() => {
  editor.destroy()
  // GrapesJS's undo manager ends each "moment" on a timer: let it run, or the
  // next test's changes would all count as one moment.
  vi.runOnlyPendingTimers()
  vi.useRealTimers()
})

const meta = (subject: string, more: Partial<Meta> = {}): Meta => ({ subject, preheader: '', abTest: null, ...more })
const ab = (variants: ABTest['variants'], more: Partial<ABTest> = {}): ABTest => ({ kind: 'subject', testPercent: 20, metric: 'human_opens', durationHours: 2, variants, ...more })

/** Ends GrapesJS's "same moment" (one undo step takes back what happened in one moment). */
const later = (ms = 5) => vi.advanceTimersByTime(ms)

describe('trackMeta', () => {
  it('undoes typing in one step per pause, and redoes it', () => {
    const shown: Meta[] = []
    const t = trackMeta(editor, meta('Weekly'), (m) => shown.push(m), { pause: 500 })
    t.change(meta('Weekly d'))
    later()
    t.change(meta('Weekly digest'))
    later(600)
    t.change(meta('Weekly digest #142'))
    later(600)
    const um = editor.UndoManager
    um.undo()
    expect(shown.at(-1)?.subject).toBe('Weekly digest')
    um.undo()
    expect(shown.at(-1)?.subject).toBe('Weekly')
    expect(um.hasUndo()).toBe(false)
    um.redo()
    expect(shown.at(-1)?.subject).toBe('Weekly digest')
    um.redo()
    expect(shown.at(-1)?.subject).toBe('Weekly digest #142')
    t.dispose()
  })

  it('takes back typing that has not paused yet', () => {
    const shown: Meta[] = []
    const t = trackMeta(editor, meta('', { preheader: 'Old' }), (m) => shown.push(m), { pause: 500 })
    t.change(meta('', { preheader: 'New preheader' }))
    // Undo is available before the pause, and takes the typing back.
    expect(editor.UndoManager.hasUndo()).toBe(true)
    editor.UndoManager.undo()
    expect(shown.at(-1)?.preheader).toBe('Old')
    // The pause that comes later doesn't bring it back.
    later(600)
    expect(shown).toHaveLength(1)
    t.dispose()
  })

  it('keeps canvas edits and subject edits in the order they were made', () => {
    const shown: Meta[] = []
    const t = trackMeta(editor, meta('One'), (m) => shown.push(m), { pause: 300 })
    const wrapper = editor.getWrapper()!
    wrapper.append('<p>Added first</p>')
    later()
    t.change(meta('Two'))
    later(400)
    wrapper.append('<p>Added last</p>')
    later()
    const um = editor.UndoManager
    um.undo()
    expect(editor.getHtml()).not.toContain('Added last')
    expect(shown).toHaveLength(0)
    um.undo()
    expect(shown.at(-1)?.subject).toBe('One')
    expect(editor.getHtml()).toContain('Added first')
    um.undo()
    expect(editor.getHtml()).not.toContain('Added first')
    t.dispose()
  })

  it('makes each A/B change a step at once, after the typing before it', () => {
    const shown: Meta[] = []
    const t = trackMeta(editor, meta('Spring sale'), (m) => shown.push(m), { pause: 500 })
    const started = ab([{ key: 'A' }, { key: 'B', subject: '' }])
    t.change(meta('Spring sale', { abTest: started }))
    later()
    // A variant's subject is typing: one step when it pauses.
    t.change(meta('Spring sale', { abTest: ab([{ key: 'A' }, { key: 'B', subject: 'Last' }]) }))
    t.change(meta('Spring sale', { abTest: ab([{ key: 'A' }, { key: 'B', subject: 'Last day' }]) }))
    // Picking a metric before the pause: the typing is its own step first.
    t.change(meta('Spring sale', { abTest: ab([{ key: 'A' }, { key: 'B', subject: 'Last day' }], { metric: 'clicks' }) }))
    later()
    const um = editor.UndoManager
    um.undo()
    expect(shown.at(-1)?.abTest?.metric).toBe('human_opens')
    expect(shown.at(-1)?.abTest?.variants[1]?.subject).toBe('Last day')
    um.undo()
    expect(shown.at(-1)?.abTest?.variants[1]?.subject).toBe('')
    um.undo()
    expect(shown.at(-1)?.abTest).toBeNull()
    t.dispose()
  })

  it('takes a reload without an undo step', () => {
    const shown: Meta[] = []
    const t = trackMeta(editor, meta('Mine'), (m) => shown.push(m))
    t.reset(meta('Theirs'))
    expect(editor.UndoManager.hasUndo()).toBe(false)
    t.change(meta('Theirs, edited'), { step: true })
    later()
    editor.UndoManager.undo()
    expect(shown.at(-1)?.subject).toBe('Theirs')
    t.dispose()
  })

  it('gives the undo manager back when it is done', () => {
    const um = editor.UndoManager
    const before = { undo: um.undo, redo: um.redo, hasUndo: um.hasUndo }
    const t = trackMeta(editor, meta('x'), () => undefined)
    t.change(meta('y'), { step: true })
    t.dispose()
    expect({ undo: um.undo, redo: um.redo, hasUndo: um.hasUndo }).toEqual(before)
  })
})

describe('typedOnly', () => {
  it('tells typing from other A/B changes', () => {
    expect(typedOnly(meta('a'), meta('b'))).toBe(true)
    expect(typedOnly(meta('a', { abTest: ab([{ key: 'A' }, { key: 'B', subject: 'x' }]) }), meta('a', { abTest: ab([{ key: 'A' }, { key: 'B', subject: 'xy' }]) }))).toBe(true)
    expect(typedOnly(meta('a'), meta('a', { abTest: ab([{ key: 'A' }]) }))).toBe(false)
    expect(typedOnly(meta('a', { abTest: ab([{ key: 'A' }]) }), meta('a', { abTest: ab([{ key: 'A' }], { testPercent: 30 }) }))).toBe(false)
  })
})
