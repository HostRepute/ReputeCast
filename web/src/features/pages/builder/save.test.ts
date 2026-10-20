import { isOffline, isUnsaved, saveReducer, theirsFrom, type SaveEvent, type SaveStatus } from './save'

const AT = '2026-10-19T10:41:00Z'
const loaded: SaveStatus = { kind: 'saved', version: 9, at: AT, by: 'Linh Nguyen', auto: true }

function run(s: SaveStatus, ...events: SaveEvent[]): SaveStatus {
  return events.reduce(saveReducer, s)
}

describe('saving the draft', () => {
  it('marks edits after a save as unsaved, keeping when it last saved', () => {
    expect(run(loaded, { type: 'edit' })).toEqual({ kind: 'dirty', version: 9, at: AT })
  })

  it('saves as the next version; Save and ⌘S are not autosaves', () => {
    const saving = run(loaded, { type: 'edit' }, { type: 'start' })
    expect(saving).toEqual({ kind: 'saving', version: 9, at: AT })
    expect(saveReducer(saving, { type: 'saved', version: 10, at: '2026-10-19T10:42:00Z', by: 'Linh Nguyen', manual: true, dirtyAgain: false })).toEqual({
      kind: 'saved',
      version: 10,
      at: '2026-10-19T10:42:00Z',
      by: 'Linh Nguyen',
      auto: false,
    })
    expect(saveReducer(saving, { type: 'saved', version: 10, at: '2026-10-19T10:42:00Z', by: 'Linh Nguyen', manual: false, dirtyAgain: false })).toMatchObject({ kind: 'saved', auto: true })
  })

  it('stays unsaved when edits came in while it saved, on top of the new version', () => {
    const s = run(loaded, { type: 'edit' }, { type: 'start' }, { type: 'edit' })
    expect(s.kind).toBe('saving')
    expect(saveReducer(s, { type: 'saved', version: 10, at: '2026-10-19T10:42:00Z', by: 'Linh Nguyen', manual: false, dirtyAgain: true })).toEqual({
      kind: 'dirty',
      version: 10,
      at: '2026-10-19T10:42:00Z',
    })
  })

  it('holds a conflict until it’s resolved: Keep mine saves on top as v11, Load theirs reloads v10', () => {
    const theirs = { version: 10, by: 'Marcus Chen', at: '2026-10-19T10:40:00Z' }
    const conflict = run(loaded, { type: 'edit' }, { type: 'start' }, { type: 'conflict', theirs })
    expect(conflict).toEqual({ kind: 'conflict', version: 9, theirs })
    expect(saveReducer(conflict, { type: 'edit' })).toBe(conflict)
    expect(saveReducer(conflict, { type: 'saved', version: 11, at: '2026-10-19T10:43:00Z', by: 'Linh Nguyen', manual: true, dirtyAgain: false })).toMatchObject({ kind: 'saved', version: 11 })
    expect(saveReducer(conflict, { type: 'loaded', version: 10, at: theirs.at, by: 'Marcus Chen', auto: false })).toEqual({ kind: 'saved', version: 10, at: theirs.at, by: 'Marcus Chen', auto: false })
  })

  it('stays offline while it retries, keeping when it began, until a save gets through', () => {
    const off = run(loaded, { type: 'edit' }, { type: 'start' }, { type: 'offline', at: '2026-10-19T10:42:00Z' })
    expect(off).toEqual({ kind: 'offline', version: 9, since: '2026-10-19T10:42:00Z' })
    expect(isUnsaved(off)).toBe(true)
    const retried = run(off, { type: 'edit' }, { type: 'start' })
    expect(retried).toBe(off)
    expect(saveReducer(retried, { type: 'offline', at: '2026-10-19T10:43:00Z' })).toEqual(off)
    expect(saveReducer(retried, { type: 'saved', version: 10, at: '2026-10-19T10:44:00Z', by: 'Linh Nguyen', manual: false, dirtyAgain: false })).toMatchObject({ kind: 'saved', version: 10 })
  })

  it('reports a failed save, and a new edit tries again', () => {
    const failed = run(loaded, { type: 'edit' }, { type: 'start' }, { type: 'failed', message: 'The page is too large.' })
    expect(failed).toEqual({ kind: 'error', version: 9, message: 'The page is too large.' })
    expect(saveReducer(failed, { type: 'edit' }).kind).toBe('dirty')
  })

  it('saves the next edit on top of a version the page’s settings made', () => {
    expect(saveReducer(loaded, { type: 'version', version: 12 })).toMatchObject({ kind: 'saved', version: 12 })
    expect(saveReducer(loaded, { type: 'version', version: 8 })).toBe(loaded)
  })

  it('knows which states would lose edits on leaving', () => {
    expect(isUnsaved(loaded)).toBe(false)
    for (const kind of ['dirty', 'saving', 'conflict', 'offline', 'error'] as const) {
      const s = { kind, version: 9 } as unknown as SaveStatus
      expect(isUnsaved(s), kind).toBe(true)
    }
  })
})

describe('a save’s answers', () => {
  it('reads the other person’s version from a version_conflict problem', () => {
    expect(theirsFrom({ version: 10, savedBy: 'Marcus Chen', savedAt: '2026-10-19T10:40:00Z' })).toEqual({ version: 10, by: 'Marcus Chen', at: '2026-10-19T10:40:00Z' })
    expect(theirsFrom(undefined)).toEqual({ version: 0, by: '', at: '' })
    expect(theirsFrom({ version: 'ten', savedBy: 7 })).toEqual({ version: 0, by: '', at: '' })
  })

  it('treats network failures and gateways as offline, not as errors', () => {
    expect(isOffline(new TypeError('Failed to fetch'))).toBe(true)
    expect(isOffline({ status: 503 })).toBe(true)
    expect(isOffline({ status: 502 })).toBe(true)
    expect(isOffline({ status: 409 })).toBe(false)
    expect(isOffline({ status: 422 })).toBe(false)
    expect(isOffline(null)).toBe(false)
  })
})
