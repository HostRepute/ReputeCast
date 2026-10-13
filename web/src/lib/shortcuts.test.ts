import { isTypingTarget, runShortcutHandler, shortcutFor, type KeyLike } from './shortcuts'

const key = (k: string, mods: Partial<KeyLike> = {}, target: EventTarget | null = document.body): KeyLike => ({
  key: k,
  metaKey: false,
  ctrlKey: false,
  altKey: false,
  shiftKey: false,
  target,
  ...mods,
})

describe('shortcutFor', () => {
  it('maps ⌘K, ⌘S and ⌘Enter, with Ctrl on other systems', () => {
    expect(shortcutFor(key('k', { metaKey: true }), false).action).toEqual({ type: 'search' })
    expect(shortcutFor(key('K', { ctrlKey: true }), false).action).toEqual({ type: 'search' })
    expect(shortcutFor(key('s', { metaKey: true }), false).action).toEqual({ type: 'save' })
    expect(shortcutFor(key('Enter', { ctrlKey: true }), false).action).toEqual({ type: 'readiness' })
  })

  it('keeps ⌘ shortcuts working while typing, single keys never', () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    expect(shortcutFor(key('k', { metaKey: true }, input), false).action).toEqual({ type: 'search' })
    expect(shortcutFor(key('n', {}, input), false).action).toBeUndefined()
    expect(shortcutFor(key('g', {}, input), false).pendingG).toBe(false)
    expect(shortcutFor(key('?', { shiftKey: true }, input), false).action).toBeUndefined()
    input.remove()
  })

  it('runs N, ? and the G-then sequences', () => {
    expect(shortcutFor(key('n'), false).action).toEqual({ type: 'new-campaign' })
    expect(shortcutFor(key('?', { shiftKey: true }), false).action).toEqual({ type: 'help' })
    const g = shortcutFor(key('g'), false)
    expect(g).toEqual({ pendingG: true })
    expect(shortcutFor(key('o'), true).action).toEqual({ type: 'go', to: 'overview' })
    expect(shortcutFor(key('a'), true).action).toEqual({ type: 'go', to: 'audience' })
    expect(shortcutFor(key('c'), true).action).toEqual({ type: 'go', to: 'campaigns' })
    // An unknown second key ends the sequence without doing anything.
    expect(shortcutFor(key('x'), true)).toEqual({ pendingG: false })
  })

  it('ignores keys with Alt, with ⌘ and other letters, and inside dialogs', () => {
    expect(shortcutFor(key('n', { altKey: true }), false).action).toBeUndefined()
    expect(shortcutFor(key('n', { metaKey: true }), false).action).toBeUndefined()
    const dialog = document.createElement('div')
    dialog.setAttribute('role', 'dialog')
    dialog.setAttribute('aria-modal', 'true')
    const button = document.createElement('button')
    dialog.appendChild(button)
    document.body.appendChild(dialog)
    expect(shortcutFor(key('n', {}, button), false).action).toBeUndefined()
    dialog.remove()
  })
})

describe('isTypingTarget', () => {
  it('treats text fields and editors as typing, toggles as not', () => {
    const text = document.createElement('input')
    const box = document.createElement('input')
    box.type = 'checkbox'
    const area = document.createElement('textarea')
    const editor = document.createElement('div')
    editor.setAttribute('contenteditable', 'true')
    const inside = document.createElement('span')
    editor.appendChild(inside)
    expect(isTypingTarget(text)).toBe(true)
    expect(isTypingTarget(area)).toBe(true)
    expect(isTypingTarget(box)).toBe(false)
    expect(isTypingTarget(inside)).toBe(true)
    expect(isTypingTarget(document.body)).toBe(false)
  })
})

describe('runShortcutHandler', () => {
  it('reports when no screen handles the shortcut', () => {
    expect(runShortcutHandler('save')).toBe(false)
  })
})
