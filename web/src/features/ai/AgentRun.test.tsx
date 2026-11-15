import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { useState } from 'react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { AgentReview } from './AgentReview'
import { AgentLimitsDialog, AgentSettings } from './AgentSettings'
import { AgentBriefPage } from './AgentBriefPage'
import { AgentRunPage } from './AgentRunPage'
import { AgentRunsPage } from './AgentRunsPage'
import { useAgentMutation, type AgentRun } from './agent'

const endpoint = '/api/v1/workspaces/acme-newsletter/ai/agent/runs/run-1'
const email = { subject: 'Faster exports', preheader: 'Fewer steps', mjml: '<mjml><mj-body></mj-body></mjml>' }
const run: AgentRun = {
  id: 'run-1', ownerId: 'owner', state: 'review', version: 7, planRevision: 2, acceptedRevision: 2, pauseReason: '',
  model: 'Recorded model', toolCalls: 18, tokens: 84200, costMicros: 420000, uncertain: false,
  limits: { toolCalls: 40, tokens: 300000, costMicros: 2000000 },
  createdAt: '2026-11-14T09:00:00Z', updatedAt: '2026-11-14T09:04:00Z', expiresAt: '2026-12-14T09:00:00Z',
  planDiscards: [], steps: [], conversation: [],
  plan: { summary: 'Product launch', questions: [], items: [
    { handle: 'one', kind: 'campaign', name: 'Announcement', bundle: 'launch', dependsOn: [], purpose: 'Introduce exports' },
    { handle: 'two', kind: 'campaign', name: 'Follow-up', bundle: 'launch', dependsOn: ['one'], purpose: 'Explain next steps' },
  ] },
  artifacts: [
    { id: 'one', handle: 'one', kind: 'campaign', name: 'Announcement', bundle: 'launch', revision: 3, decision: '', email, html: '<p>Faster exports</p>' },
    { id: 'two', handle: 'two', kind: 'campaign', name: 'Follow-up', bundle: 'launch', revision: 5, decision: '', email, html: '<p>Next steps</p>' },
  ],
}
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

describe('agent review decisions', () => {
  it('offers saved one-shot proposals from the run list after AI permission is removed', async () => {
    const base = '/api/v1/workspaces/acme-newsletter'
    unmock = mockApi({
      [`GET ${base}/ai/agent/runs`]: () => json({ items: [] }),
      [`GET ${base}/ai/drafts/campaigns`]: (r) => json({ items: new URL(r.url).searchParams.get('all') === 'true' ? [{ id: 'old-proposal', name: 'Saved export proposal', expiresAt: run.expiresAt }] : [] }),
      [`GET ${base}/ai/drafts/campaigns/old-proposal`]: () => json({ id: 'old-proposal', name: 'Saved export proposal', content: email, html: '<p>Faster exports</p>', subjectCheck: { heuristics: [] }, sent: { modelName: 'Recorded model', inputTokens: 50, outputTokens: 40, costMicros: 100 }, expiresAt: run.expiresAt, decision: '' }),
    })
    await renderInWorkspace(<AgentRunsPage />, { me: workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] }) })
    expect(screen.queryByRole('link', { name: 'New brief' })).not.toBeInTheDocument()
    fireEvent.click(await screen.findByRole('button', { name: 'Review proposal: Saved export proposal' }))
    expect(await screen.findByRole('button', { name: 'Accept and save draft' })).toBeEnabled()
    expect(screen.getByTitle('Proposed email preview')).toHaveAttribute('sandbox', '')
  })

  it('keeps discount selection local and omits it when changing to a post', async () => {
    const base = '/api/v1/workspaces/acme-newsletter'
    const submitted: Record<string, unknown>[] = []
    unmock = mockApi({
      [`GET ${base}/ai/status`]: () => json({ state: 'ready', features: [], prices: { known: true }, agent: { enabled: true, limits: run.limits }, usage: { costMicros: 0 } }),
      [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
      [`GET ${base}/coupons/pools`]: () => json({ items: [{ id: 'pool-1', name: 'Merch offer', status: 'active', free: 0 }, { id: 'old', name: 'Retired offer', status: 'archived', free: 10 }] }),
      [`POST ${base}/ai/agent/runs`]: async (r) => { submitted.push(await r.json()); return json({ detail: 'Pool changed' }, 409) },
    })
    await renderInWorkspace(<AgentBriefPage />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit', 'website:view', 'website:edit'] }) })
    const submit = screen.getByRole('button', { name: 'Create a plan' })
    await waitFor(() => expect(submit).toBeEnabled())
    fireEvent.click(screen.getByText('Include an existing discount pool'))
    const selection = await screen.findByLabelText('Selected discount pool')
    expect(selection).toHaveValue('')
    expect(screen.queryByRole('option', { name: 'Retired offer' })).not.toBeInTheDocument()
    fireEvent.change(selection, { target: { value: 'pool-1' } })
    expect(screen.getByText(/0 codes available now/)).toHaveTextContent('not a reservation')
    fireEvent.change(screen.getByLabelText('What would you like to draft?'), { target: { value: 'Draft the approved offer.' } })
    fireEvent.click(submit)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    await screen.findByText('Pool changed.')
    expect(submitted[0]?.poolId).toBe('pool-1')
    expect(submitted[0]).not.toHaveProperty('pool')
    fireEvent.change(screen.getByLabelText('Draft type'), { target: { value: 'post' } })
    expect(screen.queryByLabelText('Selected discount pool')).not.toBeInTheDocument()
    fireEvent.click(submit)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    await waitFor(() => expect(submitted).toHaveLength(2))
    expect(submitted[1]).not.toHaveProperty('poolId')
  })

  it.each([true, false])('distinguishes a build estimate from budget authority (known=%s)', async (known) => {
    unmock = mockApi({ [`GET ${endpoint}`]: () => json({ ...run, state: 'ready', acceptedRevision: 0,
      estimate: known ? { lowMicros: 110000, highMicros: 1700000, pricingAt: '2026-11-14T08:00:00Z' } : undefined,
    }) })
    await renderInWorkspace(<AgentRunPage id={run.id} selectedTab="plan" onTabChange={() => {}} />)
    await screen.findByRole('button', { name: 'Build drafts' })
    if (known) {
      expect(screen.getByText(/Remaining build estimate:/)).toHaveTextContent('$0.11–$1.70')
      // 1.70 fits the full $2 ceiling, but not $1.58 remaining after usage.
      expect(screen.getByText(/The higher scenario exceeds/)).toBeVisible()
      expect(screen.getByText(/Scenarios at prices configured/)).toHaveTextContent('actual cost can fall outside the range')
    } else {
      expect(screen.getByText(/Build cost unavailable/)).toHaveTextContent('unavailable does not mean free')
      expect(screen.queryByText(/Remaining build estimate:/)).not.toBeInTheDocument()
    }
  })

  it('shares only an explicitly selected report and preserves the brief after an unavailable snapshot', async () => {
    const base = '/api/v1/workspaces/acme-newsletter'
    const submitted: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/ai/status`]: () => json({ state: 'ready', features: [], prices: { known: true }, agent: { enabled: true, limits: run.limits }, usage: { costMicros: 0 } }),
      [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
      [`GET ${base}/campaigns`]: () => json({ items: [{ id: 'report-1', name: 'Previous launch' }], total: 1 }),
      [`POST ${base}/ai/agent/runs`]: async (r) => { submitted.push(await r.json()); return json({ detail: 'Report snapshot expired' }, 409) },
    })
    await renderInWorkspace(<AgentBriefPage />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit', 'reports:view'] }) })
    const submit = screen.getByRole('button', { name: 'Create a plan' })
    await waitFor(() => expect(submit).toBeEnabled())
    fireEvent.click(screen.getByText('Campaign totals for context'))
    const selection = await screen.findByLabelText('Report to share with the model')
    expect(selection).toHaveValue('')
    fireEvent.change(selection, { target: { value: 'report-1' } })
    fireEvent.change(screen.getByLabelText('What would you like to draft?'), { target: { value: 'Draft a useful update.' } })
    fireEvent.click(submit)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    await screen.findByText('Report snapshot expired.')
    expect(submitted).toEqual([{ sharingReviewed: true, brief: 'Draft a useful update.', kind: 'campaign', language: 'en', templateId: '', links: [], reportId: 'report-1', requestKey: expect.any(String) }])
    expect(screen.getByLabelText('What would you like to draft?')).toHaveValue('Draft a useful update.')
    fireEvent.click(screen.getByRole('button', { name: 'Do not include a report' }))
    fireEvent.click(submit)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    await waitFor(() => expect(submitted).toHaveLength(2))
    expect(submitted[1]).not.toHaveProperty('reportId')
    expect(submitted[1]).not.toHaveProperty('campaignId')
  })

  it('shows the fixed report projection without turning withheld counts into zero', async () => {
    unmock = mockApi({ [`GET ${endpoint}`]: () => json({ ...run, report: {
      status: 'rounded', rounding: 10, sent: 230, metrics: { delivered: 200, humanOpens: 70 },
      asOf: '2026-11-14T09:00:00Z', sentFrom: '2026-11-08T00:00:00Z', sentThrough: '2026-11-09T00:00:00Z',
    } }) })
    await renderInWorkspace(<AgentRunPage id={run.id} selectedTab="plan" onTabChange={() => {}} />)
    const title = await screen.findByRole('heading', { name: 'Campaign totals for context' })
    const report = title.closest('section')!
    expect(within(report).getByText('Sent').nextElementSibling).toHaveTextContent('230')
    expect(within(report).getByText('Human opens').nextElementSibling).toHaveTextContent('70')
    expect(within(report).getByText('Human clicks').nextElementSibling).toHaveTextContent('Withheld')
    expect(within(report).getAllByText('Withheld')).toHaveLength(3)
    expect(report).toHaveTextContent('The shared snapshot never refreshes')
  })

  it.each(['joined_list', 'segment_entered', 'segment_left'] as const)('names the selected audience in both sides of a %s proposal', async (kind) => {
    const name = kind === 'joined_list' ? 'Weekly digest' : 'Quiet readers'
    const reference = kind === 'joined_list' ? 'listId' : 'segmentId'
    const graph = (id: string) => ({ nodes: [{ id: 'start', type: 'trigger' as const, position: { x: 0, y: 0 }, data: { trigger: { kind, [reference]: id, reentry: 'once' as const } } }], edges: [] })
    function Review() {
      const mutations = useAgentMutation(run.id)
      return <AgentReview run={{ ...run, artifacts: [{ ...run.artifacts[0]!, kind: 'automation', email: undefined, triggerName: name,
        targetId: 'flow-1', beforeAutomation: graph('resource-1'), automation: { graph: graph('selected_audience'), emails: [] },
      }] }} mutations={mutations} permissions={['automations:edit']} blocked={false} owner={false} />
    }
    await renderInWorkspace(<Review />)
    fireEvent.click(screen.getByText('Review content'))
    for (const label of ['Before · saved content', 'Proposed · not saved']) {
      const review = await screen.findByRole('region', { name: `Steps and paths · ${label}` })
      expect(within(review).getByText(name, { exact: false })).toBeVisible()
    }
  })

  it('names missing drafts and resumes the same run without clearing cumulative usage', async () => {
    const actions: unknown[] = []
    const paused = { ...run, state: 'paused', pauseReason: 'incomplete_plan', artifacts: [run.artifacts[0]!] }
    unmock = mockApi({
      [`GET ${endpoint}`]: () => json(paused),
      'GET /api/v1/me': () => json({ user: { id: 'owner', name: 'Draft owner', email: 'owner@example.test' }, workspaces: [], instance: {} }),
      [`POST ${endpoint}/actions`]: async (r) => { actions.push(await r.json()); return new Response(null, { status: 204 }) },
    })
    await renderInWorkspace(<AgentRunPage id={run.id} selectedTab="steps" onTabChange={() => {}} />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    await screen.findByText(/The model stopped before finishing/)
    expect(screen.getByText('Not yet built').parentElement).toHaveTextContent('Follow-up')
    expect(screen.getByText('Not yet built').parentElement).not.toHaveTextContent('Announcement')
    const resume = screen.getByRole('button', { name: 'Continue saved work' })
    await waitFor(() => expect(resume).toBeEnabled())
    fireEvent.click(resume)
    await waitFor(() => expect(actions).toEqual([{ action: 'continue', version: 7, planRevision: 2 }]))
    expect(screen.getByText('18 / 40')).toBeVisible()
  })

  it('stops a building run even when a step finishes while the confirmation is open', async () => {
    const actions: unknown[] = []
    let current: AgentRun = { ...run, state: 'building' }
    unmock = mockApi({
      [`GET ${endpoint}`]: () => json(current),
      'GET /api/v1/me': () => json({ user: { id: 'owner', name: 'Draft owner', email: 'owner@example.test' }, workspaces: [], instance: {} }),
      [`POST ${endpoint}/actions`]: async (r) => {
        const body = await r.json() as { version: number }
        actions.push(body)
        if (body.version !== current.version) return json({ code: 'ai_agent_changed', detail: 'This run or plan changed' }, 409)
        current = { ...current, state: 'cancelled', version: current.version + 1 }
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<AgentRunPage id={run.id} selectedTab="steps" onTabChange={() => {}} />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    const stop = await screen.findByRole('button', { name: 'Stop run' })
    await waitFor(() => expect(stop).toBeEnabled())
    stop.focus()
    fireEvent.click(stop)
    const dialog = await screen.findByRole('dialog', { name: 'Stop run' })
    // The worker saves a step, as it does after every model response.
    current = { ...current, version: 8 }
    fireEvent.click(within(dialog).getByRole('button', { name: 'Stop run' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(actions).toEqual([{ action: 'cancel', version: 8, planRevision: 2 }])
    expect(await screen.findByText('No more steps will start. Review successful drafts below; kept drafts are not deleted.')).toBeVisible()
    // Stop is gone with the running state; focus continues from the selected tab.
    expect(screen.queryByRole('button', { name: 'Stop run' })).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('tab', { name: 'Steps' })).toHaveFocus())
  })

  it('keeps keyboard focus in the review after a decision removes its buttons', async () => {
    let current: AgentRun = run
    unmock = mockApi({
      [`GET ${endpoint}`]: () => json(current),
      'GET /api/v1/me': () => json({ user: { id: 'owner', name: 'Draft owner', email: 'owner@example.test' }, workspaces: [], instance: {} }),
      [`POST ${endpoint}/decisions`]: () => {
        current = { ...current, state: 'completed', version: 8, artifacts: current.artifacts.map((a) => ({ ...a, decision: 'kept', resultId: `kept-${a.id}` })) }
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<AgentRunPage id={run.id} selectedTab="review" onTabChange={() => {}} />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    const keep = await screen.findByRole('button', { name: 'Keep bundle' })
    await waitFor(() => expect(keep).toBeEnabled())
    keep.focus()
    fireEvent.click(keep)
    const dialog = await screen.findByRole('dialog', { name: 'Keep bundle' })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Keep bundle' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Keep bundle' })).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getAllByRole('heading', { name: 'Review', level: 2 })[0]).toHaveFocus())
  })

  it.each(['answer', 'revise', 'replan'] as const)('requires a fresh sharing confirmation for %s and retains refused input', async (action) => {
    const actions: unknown[] = []
    unmock = mockApi({
      [`GET ${endpoint}`]: () => json({ ...run, state: action === 'answer' ? 'needs_answers' : 'review' }),
      'GET /api/v1/me': () => json({ user: { id: 'owner', name: 'Draft owner', email: 'owner@example.test' }, workspaces: [], instance: {} }),
      [`POST ${endpoint}/actions`]: async (r) => { actions.push(await r.json()); return json({ detail: 'Run changed during review' }, 409) },
    })
    await renderInWorkspace(<AgentRunPage id={run.id} selectedTab="plan" onTabChange={() => {}} />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    const instruction = await screen.findByRole('textbox')
    await waitFor(() => expect(instruction).toBeEnabled())
    if (action === 'revise') fireEvent.change(screen.getByRole('combobox'), { target: { value: 'one' } })
    fireEvent.change(instruction, { target: { value: 'Shorten the export introduction.' } })
    fireEvent.submit(instruction.closest('form')!)
    const sharing = within(await screen.findByRole('dialog', { name: 'Review before sharing' }))
    expect(sharing.getByText(/Recorded model/)).toBeVisible()
    expect(actions).toEqual([])
    fireEvent.click(sharing.getByRole('button', { name: 'Cancel' }))
    expect(instruction).toHaveValue('Shorten the export introduction.')
    fireEvent.submit(instruction.closest('form')!)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    await screen.findByText('Run changed during review.')
    expect(actions).toEqual([{ sharingReviewed: true, action, version: 7, planRevision: 2, message: 'Shorten the export introduction.', ...(action === 'revise' ? { handle: 'one' } : {}) }])
    expect(instruction).toHaveValue('Shorten the export introduction.')
  })

  it.each(['campaign', 'post', 'automation'] as const)('collects the requested change before selecting a kept %s for a new proposal', async (kind) => {
    const actions: unknown[] = []
    const permissions = ['ai:use', kind === 'post' ? 'website:edit' : `${kind}s:edit`]
    let current: AgentRun = { ...run, state: 'completed', artifacts: [{ ...run.artifacts[0]!, kind, decision: 'kept', resultId: 'kept-draft' }] }
    unmock = mockApi({
      [`GET ${endpoint}`]: () => json(current),
      'GET /api/v1/me': () => json({ user: { id: 'owner', name: 'Draft owner', email: 'owner@example.test' }, workspaces: [], instance: {} }),
      [`POST ${endpoint}/actions`]: async (r) => {
        actions.push(await r.json())
        current = { ...current, state: 'ready', version: 8, planRevision: 3, acceptedRevision: 0 }
        return new Response(null, { status: 204 })
      },
    })
    function Review() {
      const [tab, setTab] = useState<'plan' | 'steps' | 'review'>('review')
      return <AgentRunPage id={run.id} selectedTab={tab} onTabChange={setTab} />
    }
    await renderInWorkspace(<Review />, { me: workspaceMe({ permissions }) })
    fireEvent.click(await screen.findByText('Review content'))
    expect(await screen.findByRole('region', { name: 'Revision 3 · Kept as drafts' })).toBeVisible()
    expect(screen.queryByText('Proposed · not saved')).not.toBeInTheDocument()
    const propose = await screen.findByRole('button', { name: 'Propose another change' })
    await waitFor(() => expect(propose).toBeEnabled())
    fireEvent.click(propose)
    const dialog = await screen.findByRole('dialog')
    const confirm = within(dialog).getByRole('button', { name: 'Propose another change' })
    expect(confirm).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(confirm)
    expect(actions).toEqual([])
    fireEvent.change(within(dialog).getByLabelText('Add an instruction'), { target: { value: 'Shorten the subject without changing the offer.' } })
    fireEvent.click(confirm)
    await waitFor(() => expect(actions).toEqual([{ sharingReviewed: true, action: 'propose', version: 7, planRevision: 2, handle: 'one', message: 'Shorten the subject without changing the offer.' }]))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByRole('tab', { name: 'Brief & plan' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('button', { name: 'Build drafts' })).toBeVisible()
  })

  it('shows post title, excerpt and markup changes with separately labelled before and after content', async () => {
    function Review() {
      const mutations = useAgentMutation(run.id)
      return <AgentReview run={{ ...run, artifacts: [{ ...run.artifacts[0]!, kind: 'post', email: undefined, targetId: 'post-1',
        before: { subject: 'Old guide', preheader: 'Long introduction', mjml: '<mjml><mj-body>Original</mj-body></mjml>' },
        post: { title: 'Export guide', excerpt: 'A short guide', mjml: '<mjml><mj-body>Updated</mj-body></mjml>' },
      }] }} mutations={mutations} permissions={['website:edit']} blocked={false} owner={false} />
    }
    await renderInWorkspace(<Review />)
    fireEvent.click(screen.getByText('Review content'))
    const before = await screen.findByRole('region', { name: 'Before · saved content' })
    const after = screen.getByRole('region', { name: 'Proposed · not saved' })
    expect(before).toHaveTextContent('Old guide')
    expect(after).toHaveTextContent('Export guide')
    expect(within(after).getByText('Excerpt · Changed')).toBeVisible()
    expect(within(after).queryByText('Subject')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Review content changes'))
    expect(screen.getByText(/Original/, { selector: 'del' })).toBeInTheDocument()
    expect(screen.getByText(/Updated/, { selector: 'ins' })).toBeInTheDocument()
  })

  it('tests model capability explicitly without submitting unsaved run limits', async () => {
    const base = '/api/v1/workspaces/acme-newsletter/ai/agent'
    const requests: string[] = []
    let state = 'untested'
    unmock = mockApi({
      [`GET ${base}/settings`]: () => json({ enabled: true, revision: 2, limits: run.limits, capability: { state } }),
      [`POST ${base}/test`]: async (r) => { requests.push(await r.text()); state = 'unsupported'; return json({ state }) },
    })
    await renderInWorkspace(<AgentSettings />, { me: workspaceMe({ permissions: ['integrations:view', 'integrations:manage'] }) })
    await screen.findByText('Tool calling not tested')
    expect(requests).toEqual([])
    fireEvent.change(screen.getByLabelText('Tool calls'), { target: { value: '60' } })
    fireEvent.click(screen.getByRole('button', { name: 'Test tool calling' }))
    await screen.findByText('Tool-calling check failed')
    expect(screen.getByLabelText('Tool calls')).toHaveValue(60)
    expect(requests).toEqual([''])
    expect(screen.getByText(/The test consumes the monthly AI budget/)).toBeVisible()
  })

  it('keeps exactly the displayed dependency revisions, even when progress refreshes during confirmation', async () => {
    const decisions: unknown[] = []
    unmock = mockApi({ [`POST ${endpoint}/decisions`]: async (r) => { decisions.push(await r.json()); return new Response(null, { status: 204 }) } })
    function Review() {
      const [value, setValue] = useState(run)
      const mutations = useAgentMutation(run.id)
      return <><button onClick={() => setValue({ ...run, version: 9, artifacts: run.artifacts.map((a) => ({ ...a, revision: a.revision + 1 })) })}>Refresh progress</button><AgentReview run={value} mutations={mutations} permissions={['campaigns:edit']} blocked={false} owner={false} /></>
    }
    await renderInWorkspace(<Review />)
    const refresh = screen.getByRole('button', { name: 'Refresh progress' })
    fireEvent.click(screen.getByRole('button', { name: 'Keep bundle' }))
    const dialog = await screen.findByRole('dialog', { name: 'Keep bundle' })
    expect(dialog).toHaveTextContent('Announcement · Revision 3')
    expect(dialog).toHaveTextContent('Follow-up · Revision 5')
    expect(decisions).toEqual([])
    fireEvent.click(refresh)
    fireEvent.click(within(dialog).getByRole('button', { name: 'Keep bundle' }))
    await waitFor(() => expect(decisions).toEqual([{ version: 7, decision: 'kept', artifacts: [{ id: 'one', revision: 3 }, { id: 'two', revision: 5 }] }]))
    expect(screen.queryByRole('button', { name: /send|publish/i })).not.toBeInTheDocument()
  })

  it('preserves a conflicting existing-draft proposal and can reject it without AI permission', async () => {
    const decisions: { decision: string }[] = []
    unmock = mockApi({ [`POST ${endpoint}/decisions`]: async (r) => {
      const body = await r.json()
      decisions.push(body)
      return body.decision === 'accepted' ? json({ code: 'ai_draft_changed', detail: 'Changed since this proposal.' }, 409) : new Response(null, { status: 204 })
    } })
    function Review() {
      const mutations = useAgentMutation(run.id)
      return <AgentReview run={{ ...run, artifacts: [{ ...run.artifacts[0]!, targetId: 'existing-campaign', before: { ...email, subject: 'Old subject' } }] }} mutations={mutations} permissions={['campaigns:edit']} blocked={false} owner={false} />
    }
    await renderInWorkspace(<Review />, { me: workspaceMe({ permissions: ['campaigns:edit'] }) })
    fireEvent.click(screen.getByRole('button', { name: 'Accept change' }))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Accept change' }))
    await waitFor(() => expect(within(screen.getByRole('dialog')).getByRole('alert')).toHaveTextContent('Changed since this proposal'))
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }))
    expect(await screen.findByRole('button', { name: 'Accept change' })).toBeDisabled()
    expect(screen.getByText(/Nothing was overwritten/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reject proposal' }))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Reject proposal' }))
    await waitFor(() => expect(decisions.map((d) => d.decision)).toEqual(['accepted', 'rejected']))
  })

  it('blocks keeping an incomplete bundle but allows discarding only its available staging drafts', async () => {
    const decisions: unknown[] = []
    unmock = mockApi({ [`POST ${endpoint}/decisions`]: async (r) => { decisions.push(await r.json()); return new Response(null, { status: 204 }) } })
    function Review() {
      const mutations = useAgentMutation(run.id)
      return <AgentReview run={{ ...run, state: 'cancelled', artifacts: [run.artifacts[0]!] }} mutations={mutations} permissions={['campaigns:edit']} blocked={false} owner={false} />
    }
    await renderInWorkspace(<Review />)
    expect(screen.getByRole('button', { name: 'Keep bundle' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Discard bundle' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Announcement · Revision 3')
    expect(dialog).not.toHaveTextContent('Follow-up')
    fireEvent.click(within(dialog).getByRole('button', { name: 'Discard bundle' }))
    await waitFor(() => expect(decisions).toEqual([{ version: 7, decision: 'discarded', artifacts: [{ id: 'one', revision: 3 }] }]))
  })

  it('shows readable audience logic on expansion and preserves the preview across a catalog retry', async () => {
    let available = false
    unmock = mockApi({
      'GET /api/v1/workspaces/acme-newsletter/segments/catalog': () => available ? json({ fields: [
        { key: 'list', label: 'On a list', kind: 'list', ops: [{ key: 'includes', label: 'Includes' }], options: [{ value: 'list-1', label: 'Weekly digest' }] },
        { key: 'opened', label: 'Opened an email', kind: 'period', ops: [{ key: 'not_in_last', label: 'Not in the last' }], options: [] },
        { key: 'custom:score', label: 'Reader score', kind: 'number', ops: [{ key: 'between', label: 'Between' }], options: [] },
        { key: 'last_open', label: 'Last opened', kind: 'period', ops: [{ key: 'never', label: 'Never' }], options: [] },
      ] }) : json({ detail: 'Audience labels unavailable' }, 503),
    })
    function Review() {
      const mutations = useAgentMutation(run.id)
      const item = { handle: 'audience', kind: 'segment' as const, name: 'Quiet readers', bundle: 'audience', dependsOn: [], purpose: 'Review the selected audience' }
      return <AgentReview run={{ ...run, plan: { summary: 'Audience', questions: [], items: [item] }, artifacts: [{
        ...item, id: 'audience-1', revision: 1, decision: '', segment: { groups: [
          { match: 'all', conditions: [{ field: 'list', op: 'includes', value: 'list-1' }, { field: 'opened', op: 'not_in_last', amount: 7, unit: 'weeks' }] },
          { match: 'any', conditions: [{ field: 'custom:score', op: 'between', value: '12', to: '47' }, { field: 'last_open', op: 'never' }] },
        ] },
      }] }} mutations={mutations} permissions={['segments:edit']} blocked={false} owner={false} />
    }
    await renderInWorkspace(<Review />)
    expect(screen.queryByRole('region', { name: 'Group 1' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Review content'))
    await screen.findByText('Audience labels unavailable.')
    available = true
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    const all = await screen.findByRole('region', { name: 'Group 1' })
    const any = screen.getByRole('region', { name: 'Group 2' })
    expect(within(all).getByRole('heading')).toHaveTextContent('Match ALL of these')
    expect(within(any).getByRole('heading')).toHaveTextContent('Match ANY of these')
    expect(within(all).getAllByRole('listitem').map((item) => item.textContent)).toEqual(['On a listIncludesWeekly digest', 'Opened an emailNot in the last7 weeks'])
    expect(within(any).getAllByRole('listitem').map((item) => item.textContent)).toEqual(['Reader scoreBetween12 – 47', 'Last openedNever'])
    expect(screen.queryByText('list-1')).not.toBeInTheDocument()
  })

  it('raises ceilings separately from continuation without resetting the visible usage', async () => {
    const limits: unknown[] = []
    let available = false
    unmock = mockApi({
      'GET /api/v1/workspaces/acme-newsletter/ai': () => available ? json({ connection: { cap: { kind: 'usd', usd: 10 } }, usage: { costMicros: 3_000_000, reservedCostMicros: 2_000_000, resetsAt: '2026-12-01T00:00:00Z' } }) : json({ detail: 'Monthly usage unavailable' }, 503),
      [`PUT ${endpoint}/limits`]: async (r) => { limits.push(await r.json()); return new Response(null, { status: 204 }) },
    })
    function Limits() {
      const mutations = useAgentMutation(run.id)
      return <AgentLimitsDialog run={run} onClose={() => {}} onSave={mutations.limits} />
    }
    await renderInWorkspace(<Limits />)
    expect(screen.getByRole('dialog')).toHaveTextContent('18 / 40')
    await screen.findByText('Monthly usage unavailable.')
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(limits).toEqual([])
    available = true
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(screen.getByRole('dialog')).toHaveTextContent('Monthly headroom: $5.00'))
    fireEvent.change(screen.getByLabelText('Tool calls'), { target: { value: '60' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(limits).toEqual([{ version: 7, limits: { toolCalls: 60, tokens: 300000, costMicros: 2000000 } }]))
    expect(screen.getByRole('dialog')).toHaveTextContent('18 / 40')
    expect(screen.queryByRole('button', { name: 'Continue saved work' })).not.toBeInTheDocument()
  })

  it('validates the brief and link fields before a paid plan, preserving input after server refusal', async () => {
    const base = '/api/v1/workspaces/acme-newsletter'
    const submitted: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/ai/status`]: () => json({ state: 'ready', features: [], prices: { known: true }, agent: { enabled: true, limits: run.limits }, usage: { costMicros: 7_880_000 } }),
      [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
      [`POST ${base}/ai/agent/runs`]: async (r) => { submitted.push(await r.json()); return json({ detail: 'No execution slot available' }, 409) },
    })
    await renderInWorkspace(<AgentBriefPage />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    expect(screen.queryByText('Campaign totals for context')).not.toBeInTheDocument()
    const submit = screen.getByRole('button', { name: 'Create a plan' })
    await waitFor(() => expect(submit).toBeEnabled())
    const brief = screen.getByLabelText('What would you like to draft?')
    const links = screen.getByLabelText('Links to preserve')
    fireEvent.change(brief, { target: { value: '   ' } })
    fireEvent.change(links, { target: { value: 'javascript:alert(1)' } })
    fireEvent.click(submit)
    await waitFor(() => expect(brief).toHaveAttribute('aria-invalid', 'true'))
    expect(links).toHaveAttribute('aria-invalid', 'true')
    expect(submitted).toEqual([])
    expect(screen.getByText(/Workspace usage this month/)).toHaveTextContent('This run: up to 40 tool calls, 300,000 tokens and $2.00')
    fireEvent.change(brief, { target: { value: 'Explain faster exports.' } })
    fireEvent.change(links, { target: { value: 'https://example.test/exports' } })
    fireEvent.click(submit)
    const sharing = within(await screen.findByRole('dialog', { name: 'Review before sharing' }))
    expect(submitted).toEqual([])
    fireEvent.click(sharing.getByRole('button', { name: 'Cancel' }))
    expect(submitted).toEqual([])
    fireEvent.click(submit)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    await screen.findByText('No execution slot available.')
    expect(submitted).toEqual([{ sharingReviewed: true, brief: 'Explain faster exports.', kind: 'campaign', language: 'en', templateId: '', links: ['https://example.test/exports'], requestKey: expect.any(String) }])
    expect(brief).toHaveValue('Explain faster exports.')
    expect(links).toHaveValue('https://example.test/exports')
  })

  it('resends a brief after a lost response with the same request key, and a changed brief with a new one', async () => {
    const base = '/api/v1/workspaces/acme-newsletter'
    const submitted: { brief: string; requestKey?: string }[] = []
    unmock = mockApi({
      [`GET ${base}/ai/status`]: () => json({ state: 'ready', features: [], prices: { known: true }, agent: { enabled: true, limits: run.limits }, usage: { costMicros: 0 } }),
      [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
      [`POST ${base}/ai/agent/runs`]: async (r) => { submitted.push(await r.json()); throw new TypeError('Failed to fetch') },
    })
    await renderInWorkspace(<AgentBriefPage />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit'] }) })
    const submit = screen.getByRole('button', { name: 'Create a plan' })
    const brief = screen.getByLabelText('What would you like to draft?')
    for (const [i, text] of ['Explain faster exports.', 'Explain faster exports.', 'Explain faster imports.'].entries()) {
      await waitFor(() => expect(submit).toBeEnabled())
      fireEvent.change(brief, { target: { value: text } })
      fireEvent.click(submit)
      fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
      await waitFor(() => expect(submitted).toHaveLength(i + 1))
      expect(await screen.findByRole('alert')).toBeVisible()
    }
    expect(submitted[0]?.requestKey).toMatch(/^[A-Za-z0-9_-]{1,80}$/)
    expect(submitted[1]?.requestKey).toBe(submitted[0]?.requestKey)
    expect(submitted[2]?.requestKey).not.toBe(submitted[0]?.requestKey)
  })
})
