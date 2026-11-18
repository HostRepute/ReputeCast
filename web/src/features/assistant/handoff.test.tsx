import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { AgentBriefPage, handoffBrief } from '@/features/ai/AgentBriefPage'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { Conversation, type Item } from './Conversation'

// "Start an agent run" (review of B4, finding 6): the drafted brief goes to
// the agent's page in the history state, for this workspace only, and the
// form starts from it; starting is still the person's, after the sharing review.

const brief = 'Draft three spring emails for the Engineering list.'
const items: Item[] = [
  {
    kind: 'answer',
    key: 'a1',
    turnId: 't1',
    texts: [],
    tools: [{ callId: 'c1', name: 'draft_agent_task', ok: true, ms: 40, label: 'Spring series', card: { kind: 'agent_task', title: 'Spring series', path: '/w/acme-newsletter/agent/new', brief } }],
  },
]
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

describe('the agent-task handoff', () => {
  it('carries the brief in the history state, for its workspace only', async () => {
    const { router } = await renderInWorkspace(<Conversation items={items} />)
    fireEvent.click(screen.getByRole('link', { name: 'Start an agent run' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/w/acme-newsletter/agent/new'))
    expect(router.state.location.search).toEqual({})
    expect(handoffBrief(router.state.location.state, 'acme-newsletter')).toBe(brief)
    expect(handoffBrief(router.state.location.state, 'another-workspace')).toBe('')
    expect(handoffBrief({ agentBrief: { workspace: 'acme-newsletter', text: 'x'.repeat(5000) } }, 'acme-newsletter')).toHaveLength(4000)
  })

  it('starts the brief form from it, and still asks before sharing', async () => {
    const base = '/api/v1/workspaces/acme-newsletter'
    const posted: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/ai/status`]: () => json({ state: 'ready', features: [], prices: { known: true }, agent: { enabled: true, limits: { toolCalls: 40, tokens: 300000, costMicros: 2000000 } }, usage: { costMicros: 0 } }),
      [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
      [`POST ${base}/ai/agent/runs`]: async (r) => {
        posted.push(await r.json())
        return json({ detail: 'stop here' }, 409)
      },
    })
    await renderInWorkspace(<AgentBriefPage initialBrief={brief} />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    expect(screen.getByRole('textbox', { name: /What would you like to draft/ })).toHaveValue(brief)
    const submit = screen.getByRole('button', { name: 'Create a plan' })
    await waitFor(() => expect(submit).toBeEnabled())
    fireEvent.click(submit)
    expect(await screen.findByRole('button', { name: 'Share reviewed content' })).toBeInTheDocument()
    expect(posted).toHaveLength(0)
    fireEvent.click(screen.getByRole('button', { name: 'Share reviewed content' }))
    await waitFor(() => expect(posted).toHaveLength(1))
    expect((posted[0] as { brief: string; sharingReviewed: boolean }).brief).toBe(brief)
  })
})
