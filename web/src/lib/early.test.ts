import { json, mockApi } from '@/test/api'
import { takeEarlyMe, takeEarlyWorkspace, type EarlyAnswers } from './early'
import { fetchMe, fetchWorkspaceMe } from './session'

const w = window as { __rcEarly?: EarlyAnswers }

describe('early answers', () => {
  afterEach(() => {
    w.__rcEarly = undefined
  })

  it('hands each answer out once, a workspace only for its own slug', async () => {
    w.__rcEarly = { me: Promise.resolve({ user: { id: 'u1' } }), workspace: { slug: 'acme', me: Promise.resolve({ workspace: { slug: 'acme' } }) } }
    expect(await takeEarlyMe()).toEqual({ user: { id: 'u1' } })
    expect(takeEarlyMe()).toBeUndefined()
    expect(takeEarlyWorkspace('other')).toBeUndefined()
    expect(await takeEarlyWorkspace('acme')).toEqual({ workspace: { slug: 'acme' } })
    expect(takeEarlyWorkspace('acme')).toBeUndefined()
  })

  it('is nothing without the early script', () => {
    expect(takeEarlyMe()).toBeUndefined()
    expect(takeEarlyWorkspace('acme')).toBeUndefined()
  })

  it('serves the first fetch, then the API answers', async () => {
    let calls = 0
    const off = mockApi({
      'GET /api/v1/me': () => {
        calls++
        return json({ user: { id: 'from-api' } })
      },
    })
    try {
      w.__rcEarly = { me: Promise.resolve({ user: { id: 'early' } }) }
      expect((await fetchMe())?.user.id).toBe('early')
      expect(calls).toBe(0)
      expect((await fetchMe())?.user.id).toBe('from-api')
      expect(calls).toBe(1)
    } finally {
      off()
    }
  })

  it('asks the API when the early answer was not a 200', async () => {
    const off = mockApi({
      'GET /api/v1/me': () => json({ status: 401, code: 'unauthenticated' }, 401),
      'GET /api/v1/workspaces/acme/me': () => json({ workspace: { slug: 'acme', name: 'Acme' } }),
    })
    try {
      // early.ts keeps only a 200's body; anything else is undefined.
      w.__rcEarly = { me: Promise.resolve(undefined), workspace: { slug: 'acme', me: Promise.resolve(undefined) } }
      expect(await fetchMe()).toBeNull()
      expect((await fetchWorkspaceMe('acme')).workspace.name).toBe('Acme')
    } finally {
      off()
    }
  })
})
