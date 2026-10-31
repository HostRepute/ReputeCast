import type { Schemas } from '@/api/client'
import { afterSave, changedStreams, discardStreams, keepEditsOnLatest, rebase, startEditing, toSaveInput, valueNames, type RoutingConfig } from './routing-model'

const ses = 'ses-0000-0000-0000-000000000001'
const postmark = 'pm-00-0000-0000-0000-000000000002'
const postal = 'po-00-0000-0000-0000-000000000003'

type StreamView = Schemas['RoutingStreamView']

function config(broadcast: Partial<StreamView> = {}, automation: Partial<StreamView> = {}): RoutingConfig {
  return {
    streams: [
      { stream: 'broadcast', version: 'b1', splits: [{ providerId: ses, weight: 90 }, { providerId: postal, weight: 10 }], fallback: '', failover: 'pause_and_ask', conditions: [], ...broadcast },
      { stream: 'automation', version: 'a1', splits: [{ providerId: ses, weight: 100 }], fallback: postmark, failover: 'automatic', conditions: [], ...automation },
      { stream: 'system', version: 's1', splits: [{ providerId: postmark, weight: 100 }], fallback: ses, failover: 'automatic', conditions: [] },
      { stream: 'transactional', version: '0', splits: [], fallback: '', failover: 'automatic', conditions: [] },
    ],
    providers: [],
    lists: [],
    senderDomains: [],
    categories: [],
    templates: [],
    savedBy: '',
    issues: [],
    shared: { suppressions: 0, pushed: 0 },
  }
}

describe('routing being edited (PR-2)', () => {
  it('saves only the changed streams, each with the version its edits started from', () => {
    const e = startEditing(config())
    e.draft = { ...e.draft, broadcast: { ...e.draft.broadcast, splits: [{ providerId: ses, weight: 85 }, { providerId: postal, weight: 15 }] } }
    const body = toSaveInput(e)
    expect(body.streams).toHaveLength(1)
    expect(body.streams[0]).toMatchObject({ stream: 'broadcast', baseVersion: 'b1', splits: [{ providerId: ses, weight: 85 }, { providerId: postal, weight: 15 }] })
  })

  it('keeps unsaved edits when the saved routing changes, and follows it elsewhere', () => {
    const e = startEditing(config())
    e.draft = { ...e.draft, broadcast: { ...e.draft.broadcast, splits: [{ providerId: ses, weight: 85 }, { providerId: postal, weight: 15 }] } }
    // Someone changed Automations' fallback, and the planner raised Postal.
    const next = config({ version: 'b2', splits: [{ providerId: ses, weight: 88 }, { providerId: postal, weight: 12 }] }, { version: 'a2', fallback: '' })
    const r = rebase(e, next)
    expect(r.draft.broadcast.splits[1]!.weight).toBe(15)
    expect(r.versions.broadcast).toBe('b1')
    expect(r.draft.automation.fallback).toBe('')
    expect(r.versions.automation).toBe('a2')
    expect(changedStreams(r.draft, r.base)).toEqual(['broadcast'])
  })

  it('after a refused save, shows the latest routing or keeps the edits on top of it', () => {
    const e = startEditing(config())
    e.draft = { ...e.draft, broadcast: { ...e.draft.broadcast, failover: 'automatic' } }
    const latest = config({ version: 'b2', splits: [{ providerId: ses, weight: 88 }, { providerId: postal, weight: 12 }] })
    const shown = discardStreams(e, latest, ['broadcast'])
    expect(shown.draft.broadcast.splits[1]!.weight).toBe(12)
    expect(changedStreams(shown.draft, shown.base)).toEqual([])
    const kept = keepEditsOnLatest(e, latest, ['broadcast'])
    expect(kept.draft.broadcast.failover).toBe('automatic')
    expect(toSaveInput(kept).streams[0]!.baseVersion).toBe('b2')
  })

  it('after a save, the saved streams are clean', () => {
    const e = startEditing(config())
    e.draft = { ...e.draft, system: { ...e.draft.system, fallback: '' } }
    const answer = config()
    answer.streams[2] = { ...answer.streams[2]!, version: 's2', fallback: '' }
    const done = afterSave(e, answer, ['system'])
    expect(changedStreams(done.draft, done.base)).toEqual([])
    expect(done.versions.system).toBe('s2')
  })
})

describe("a condition's values as people read them", () => {
  const cfg = {
    ...config(),
    lists: [{ id: 'l-beta', name: 'Beta Program' }],
    categories: [{ id: 'c-billing', name: 'Billing' }],
  }

  it('names lists, categories and mailbox providers, and keeps sender domains as they are', () => {
    expect(valueNames({ field: 'list', values: ['l-beta'] }, cfg)).toEqual(['Beta Program'])
    expect(valueNames({ field: 'category', values: ['c-billing'] }, cfg)).toEqual(['Billing'])
    expect(valueNames({ field: 'mailbox_provider', values: ['gmail', 'icloud'] }, cfg)).toEqual(['Gmail', 'iCloud'])
    expect(valueNames({ field: 'sender_domain', values: ['updates.acme.io'] }, cfg)).toEqual(['updates.acme.io'])
  })

  it("takes the server's labels first, and keeps an id it can't name", () => {
    expect(valueNames({ field: 'list', values: ['l-beta'], valueLabels: ['Beta testers'] }, cfg)).toEqual(['Beta testers'])
    expect(valueNames({ field: 'list', values: ['l-gone'] }, cfg)).toEqual(['l-gone'])
  })
})
