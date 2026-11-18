import { act, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { useLayoutEffect, useState } from 'react'
import type { Schemas } from '@/api/client'
import { WorkspaceProvider } from '@/lib/session'
import { json } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { setTestFetch } from '@/test/fetch'
import { setAssistantThread, threadOf, useAssistant } from './store'
import { useThreadController, type ThreadController } from './ThreadView'

// The thread controller's requests come back after the person moved on
// (review of B4, findings 1 and 4): an answer for another thread or
// workspace changes nothing here, drafts stay with their thread, and Retry
// sends the refused message itself.

const at = '2026-11-18T10:00:00Z'
const usage = { costMicros: 0, requests: 0, tokens: 0, toolCalls: 0, uncertainRequests: 0, uncertainTokens: 0, uncertainCostMicros: 0 }
const model = { id: 'm1', accountId: 'acc', model: 'claude', modelName: 'Claude', provider: 'anthropic', providerName: 'Anthropic', personal: false, usable: true, imageInput: true, tools: { state: 'supported' }, prices: {} }
const own = { ...model, id: 'pm', accountId: 'pacc', modelName: 'My Claude', personal: true }
function thread(id: string, text?: string, extra: Partial<Schemas['AIThread']> = {}): Schemas['AIThread'] {
  return {
    id,
    title: `Thread ${id}`,
    state: 'open',
    screen: 'campaigns',
    eventSeq: 3,
    createdAt: at,
    lastActivityAt: at,
    expiresAt: at,
    limits: { toolCalls: 40, tokens: 300000, costMicros: 2000000 },
    usage,
    contexts: [],
    mcpServerIds: [],
    messages: text ? [{ id: `u-${id}`, role: 'user', text, createdAt: at }] : [],
    model: model as Schemas['AIThread']['model'],
    ...extra,
  }
}

type Route = (req: Request) => Response | Promise<Response>
function serve(routes: Record<string, Route>) {
  const sent: { url: string; body: unknown }[] = []
  setTestFetch(async (input, init) => {
    const req = input instanceof Request ? input : new Request(input, init)
    const path = new URL(req.url).pathname.replace(/^\/api\/v1\/workspaces\/[^/]+/, '')
    if (req.method !== 'GET') sent.push({ url: `${req.method} ${path}`, body: await req.clone().json().catch(() => undefined) })
    const route = routes[`${req.method} ${path}`]
    return route ? route(req) : json({ status: 404, title: 'Not Found' }, 404)
  })
  return sent
}

// What the probes hold after each render.
const probe: { c?: ThreadController; toWorkspace?: (slug: string) => void } = {}
function Probing() {
  const [slug, setSlug] = useState('acme-newsletter')
  useLayoutEffect(() => {
    probe.toWorkspace = setSlug
  }, [])
  const me = workspaceMe({ permissions: ['ai:use', 'campaigns:view'], workspace: { ...workspaceMe().workspace, slug } })
  return (
    <WorkspaceProvider slug={slug} me={me}>
      <ProbeIn slug={slug} />
    </WorkspaceProvider>
  )
}
function ProbeIn({ slug }: { slug: string }) {
  useAssistant()
  // As the panel does: the thread the store says for the workspace.
  const ctl = useThreadController(threadOf(slug))
  useLayoutEffect(() => {
    probe.c = ctl
  })
  return null
}

afterEach(() => {
  setTestFetch(undefined)
  setAssistantThread('acme-newsletter', undefined)
  setAssistantThread('other', undefined)
  localStorage.clear()
})

describe('the thread controller', () => {
  it('drops a sent message’s answer that comes back after a switch to another thread', async () => {
    let answer: (r: Response) => void = () => {}
    serve({
      'GET /ai/assistant/models': () => json({ models: [model], defaultId: 'm1' }),
      'GET /ai/assistant/threads/A': () => json(thread('A', 'Earlier in A')),
      'GET /ai/assistant/threads/B': () => json(thread('B', 'Earlier in B')),
      'POST /ai/assistant/threads/A/messages': () => new Promise<Response>((done) => (answer = done)),
    })
    setAssistantThread('acme-newsletter', 'A')
    await renderInWorkspace(<Probing />)
    await waitFor(() => expect(probe.c!.state.thread?.id).toBe('A'))
    act(() => void probe.c!.send('Hello A'))
    act(() => setAssistantThread('acme-newsletter', 'B'))
    await waitFor(() => expect(probe.c!.state.thread?.id).toBe('B'))
    await act(async () => answer(json(thread('A', 'Hello A', { eventSeq: 9 }))))
    expect(probe.c!.threadId).toBe('B')
    expect(probe.c!.state.thread?.id).toBe('B')
    expect(probe.c!.state.messages.map((x) => x.text)).toEqual(['Earlier in B'])
  })

  it('keeps each workspace’s draft to itself', async () => {
    serve({ 'GET /ai/assistant/models': () => json({ models: [model], defaultId: 'm1' }) })
    await renderInWorkspace(<Probing />)
    act(() => probe.c!.setDraft('For Acme'))
    act(() => probe.toWorkspace!('other'))
    expect(probe.c!.draft.get()).toBe('')
    act(() => probe.toWorkspace!('acme-newsletter'))
    expect(probe.c!.draft.get()).toBe('For Acme')
  })

  it('retries a refused first question as it was, over new text and after switching to a personal model', async () => {
    const sent = serve({
      'GET /ai/assistant/models': () => json({ models: [model, own], defaultId: 'm1' }),
      'POST /ai/assistant/threads': () => json(thread('N')),
      'GET /ai/assistant/threads/N': () => json(thread('N')),
      'PATCH /ai/assistant/threads/N': () => json(thread('N', undefined, { model: own as Schemas['AIThread']['model'] })),
      'POST /ai/assistant/threads/N/messages': () => json({ status: 409, title: 'Conflict', code: 'ai_cap_reached', detail: 'The monthly cap is reached.' }, 409),
    })
    await renderInWorkspace(<Probing />)
    await waitFor(() => expect(probe.c!.list.length).toBe(2))
    await act(async () => probe.c!.send('First question'))
    await waitFor(() => expect(probe.c!.sendProblem?.problem).toBe('monthly'))
    expect(probe.c!.draft.get()).toBe('First question')
    // The person types something else; the card's personal-model button still retries the refused question.
    act(() => probe.c!.setDraft('Something else'))
    const card = probe.c!
    await act(async () => card.pickModel('pm').then(card.retry))
    await waitFor(() => expect(sent.filter((x) => x.url === 'POST /ai/assistant/threads/N/messages')).toHaveLength(2))
    expect(sent.filter((x) => x.url === 'POST /ai/assistant/threads/N/messages').map((x) => (x.body as { text: string }).text)).toEqual(['First question', 'First question'])
    expect(probe.c!.draft.get()).toBe('Something else')
  })
})

