import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useLayoutEffect } from 'react'
import { VoiceLearning } from '@/features/brandkit/BrandVoice'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { AgentBriefPage } from './AgentBriefPage'
import { composerMenu, itemBlock } from './composer/state'
import { m, ms } from './messages'
import { useReportAI } from './report'
import { useAssist, type Assist } from './suggest'

// Each AI entry point goes by its own feature's model (B1's per-feature
// status): the person's own model can be ready while the workspace is at
// its cap, and the other way round.

const base = '/api/v1/workspaces/acme-newsletter'
const me = workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit', 'reports:view', 'brand:edit'] })
const usage = (costMicros: number, cap: number) => ({ costMicros, share: costMicros / cap, state: costMicros >= cap ? 'reached' : 'ok', tokens: 0, inputTokens: 0, outputTokens: 0, requests: 1, reservedCostMicros: 0, reservedTokens: 0, monthStart: '2026-11-01T00:00:00Z', resetsAt: '2026-12-01T00:00:00Z' })
const workspaceModel = { personal: false, providerName: 'Anthropic', modelName: 'Workspace Claude', prices: { input: 3, output: 15, known: true }, cap: { kind: 'usd', usd: 25 }, usage: usage(25_000_000, 25_000_000) }
const ownModel = { personal: true, providerName: 'OpenAI', modelName: 'My GPT', prices: { input: 1, output: 2, known: true }, cap: { kind: 'usd', usd: 10 }, usage: usage(1_000_000, 10_000_000) }
const keys = ['subject_ideas', 'rewrite', 'translate', 'alt_text', 'accessibility_fix', 'segment_draft', 'report_summary', 'brand_voice', 'draft']
type F = Record<string, unknown>
function status(top: string, own: Record<string, F>, rest: F) {
  const features = [...keys, 'agent'].map((key) => ({ key, on: true, available: true, ...rest, ...(own[key] ?? {}) }))
  return {
    ...workspaceModel,
    ...rest,
    state: top,
    features,
    helpers: ['Linh Nguyen'],
    canManage: false,
    hostRepute: false,
    accounts: [],
    defaults: [],
    imageInput: true,
    agent: { enabled: true, limits: { toolCalls: 40, tokens: 300000, costMicros: 2000000 }, capability: { state: 'supported' } },
  }
}
/** The workspace at its cap; the person's own model ready for some features. */
const ownReady = (...f: string[]) => status('cap_reached', Object.fromEntries(f.map((k) => [k, { ...ownModel, state: 'ready' }])), { ...workspaceModel, state: 'cap_reached' })
/** The workspace ready; the person's own model at its cap for some features. */
const ownSpent = (...f: string[]) => status('ready', Object.fromEntries(f.map((k) => [k, { ...ownModel, state: 'cap_reached', usage: usage(10_000_000, 10_000_000) }])), { ...workspaceModel, state: 'ready', usage: usage(0, 25_000_000) })

const held: { a?: Assist } = {}
let unmock: (() => void) | undefined
afterEach(() => {
  unmock?.()
  held.a = undefined
})

function Probe() {
  const a = useAssist()
  useLayoutEffect(() => {
    held.a = a
  })
  return null
}
async function assistWith(s: unknown) {
  unmock = mockApi({ [`GET ${base}/ai/status`]: () => json(s) })
  await renderInWorkspace(<Probe />, { me })
  await waitFor(() => expect(held.a?.status).toBeDefined())
  return held.a!
}

describe('the composer’s Assist menu', () => {
  it('offers a feature on the person’s own model while the workspace is at its cap', async () => {
    const a = await assistWith(ownReady('subject_ideas'))
    const menu = composerMenu(a)
    expect(menu).toMatchObject({ none: false, blocked: false })
    expect(menu.lead.model).toBe('My GPT')
    expect(itemBlock(a, 'subject_ideas')).toBeUndefined()
    expect(itemBlock(a, 'rewrite')).toBe(ms.itemCap)
  })

  it('says the person’s own model is at its cap while the workspace’s answers', async () => {
    const a = await assistWith(ownSpent('subject_ideas'))
    const menu = composerMenu(a)
    expect(menu).toMatchObject({ none: false, blocked: false })
    expect(menu.lead.model).toBe('Workspace Claude')
    expect(itemBlock(a, 'subject_ideas')).toBe(ms.itemCap)
    expect(itemBlock(a, 'rewrite')).toBeUndefined()
  })

  it('is blocked only when every feature’s model is, and names a rejected own key as the person’s', async () => {
    expect(composerMenu(await assistWith(ownReady())).blocked).toBe(true)
    const rejected = status('ready', { subject_ideas: { ...ownModel, state: 'key_rejected' } }, { ...workspaceModel, state: 'ready' })
    const a = await assistWith(rejected)
    expect(itemBlock(a, 'subject_ideas')).toBe(m.keyRejectedBadge)
    expect(a.of('subject_ideas').personal).toBe(true)
  })
})

function Report() {
  const { button, panel } = useReportAI('c1', true)
  return (
    <>
      <Probe />
      {button}
      {panel}
    </>
  )
}

describe('Summarize results', () => {
  it('asks the person’s own model while the workspace has none', async () => {
    const asked = vi.fn(() => json({ status: 409, title: 'Conflict', code: 'ai_cap_reached', detail: 'stop here' }, 409))
    const none = status('not_configured', { report_summary: { ...ownModel, state: 'ready' } }, { state: 'not_configured' })
    unmock = mockApi({ [`GET ${base}/ai/status`]: () => json(none), [`GET ${base}/campaigns/c1/ai-summary`]: () => json({ kept: false }), [`POST ${base}/ai/report-summary`]: asked })
    await renderInWorkspace(<Report />, { me })
    await waitFor(() => expect(held.a?.status).toBeDefined())
    fireEvent.click(await screen.findByRole('button', { name: 'Summarize results' }))
    await waitFor(() => expect(asked).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: 'Connect a model' })).toBeNull()
  })

  it('invites to connect a model when the feature has none, though the workspace’s answers', async () => {
    const asked = vi.fn(() => json({}))
    const missing = status('ready', { report_summary: { state: 'not_configured' } }, { ...workspaceModel, state: 'ready' })
    unmock = mockApi({ [`GET ${base}/ai/status`]: () => json(missing), [`GET ${base}/campaigns/c1/ai-summary`]: () => json({ kept: false }), [`POST ${base}/ai/report-summary`]: asked })
    await renderInWorkspace(<Report />, { me })
    await waitFor(() => expect(held.a?.status).toBeDefined())
    fireEvent.click(await screen.findByRole('button', { name: 'Summarize results' }))
    expect(await screen.findByText(/Write a plain-language summary/)).toBeInTheDocument()
    expect(asked).not.toHaveBeenCalled()
  })
})

describe('Learn the brand voice', () => {
  const reads = { [`GET ${base}/campaigns`]: () => json({ items: [], total: 0, page: 1, pageSize: 10 }) }
  it('learns on the person’s own model while the workspace is at its cap', async () => {
    unmock = mockApi({ ...reads, [`GET ${base}/ai/status`]: () => json(ownReady('brand_voice')) })
    await renderInWorkspace(<VoiceLearning blocked={false} onBusy={vi.fn()} onSuggestion={vi.fn()} />, { me })
    expect(await screen.findByText('OpenAI · My GPT')).toBeInTheDocument()
    expect(screen.queryByText("Assist · This month's cap is reached")).toBeNull()
  })

  it('says the person’s own model is at its cap while the workspace’s answers', async () => {
    unmock = mockApi({ ...reads, [`GET ${base}/ai/status`]: () => json(ownSpent('brand_voice')) })
    await renderInWorkspace(<VoiceLearning blocked={false} onBusy={vi.fn()} onSuggestion={vi.fn()} />, { me })
    const notes = await screen.findAllByText("Assist · This month's cap is reached")
    expect(notes.some((n) => n.matches('p.note'))).toBe(true)
  })
})

describe('the agent brief', () => {
  const reads = { [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }) }
  it('starts on the person’s own agent model while the workspace is at its cap', async () => {
    unmock = mockApi({ ...reads, [`GET ${base}/ai/status`]: () => json(ownReady('agent')) })
    await renderInWorkspace(<AgentBriefPage initialBrief="Draft a spring series." />, { me })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Create a plan' })).toBeEnabled())
    expect(screen.getByText('OpenAI · My GPT')).toBeInTheDocument()
  })

  it('doesn’t start when the person’s own agent model is at its cap, though the workspace’s answers', async () => {
    unmock = mockApi({ ...reads, [`GET ${base}/ai/status`]: () => json(ownSpent('agent')) })
    await renderInWorkspace(<AgentBriefPage initialBrief="Draft a spring series." />, { me })
    expect(await screen.findByText(/To plan a task, connect a working model/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create a plan' })).toBeDisabled()
  })
})
