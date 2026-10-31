import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { MarketingCapCard } from '@/features/sending-rules/MarketingCapCard'
import { emptyRefs, type FlowGraph, type TriggerConfig } from './graph'
import { DeleteFlowDialog, ResumeDialog } from './manage'
import { ContactAutomationsCard, StepContactsButton } from './people'
import { SequenceStats } from './sequence'
import { AlsoTriggers } from './triggers'

const ws = '/api/v1/workspaces/acme-newsletter'
const owner = workspaceMe({
  permissions: ['automations:view', 'automations:edit', 'automations:publish', 'subscribers:view', 'workspace:manage'],
  workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' },
})
const noContent = () => new Response(null, { status: 204 })

describe('Automations, complete (review 2026-11 M44)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('AT-1: deleting a live flow says who is in it, asks for its name and offers Pause instead', async () => {
    const calls: string[] = []
    unmock = mockApi({
      [`DELETE ${ws}/automations/a1`]: () => {
        calls.push('delete')
        return noContent()
      },
    })
    let done = false
    await renderInWorkspace(<DeleteFlowDialog flow={{ id: 'a1', name: 'Welcome series', status: 'live', inFlow: 1284 }} onClose={() => {}} onDeleted={() => (done = true)} />, { me: owner })
    const dialog = screen.getByRole('dialog', { name: 'Delete Welcome series?' })
    expect(dialog).toHaveTextContent('1,284 contacts are in this flow. They leave it at once and get none of its remaining emails')
    expect(within(dialog).getByRole('button', { name: 'Pause instead' })).toBeInTheDocument()
    const del = within(dialog).getByRole('button', { name: 'Delete flow' })
    expect(del).toHaveAttribute('aria-disabled', 'true')
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Type Welcome series to confirm' }), { target: { value: 'Welcome series' } })
    fireEvent.click(del)
    await waitFor(() => expect(calls).toEqual(['delete']))
    await waitFor(() => expect(done).toBe(true))
  })

  it('AT-4: resuming asks what happens to contacts whose wait ended', async () => {
    const bodies: unknown[] = []
    unmock = mockApi({
      [`POST ${ws}/automations/a1/resume`]: async (r) => {
        bodies.push(await r.json())
        return json({})
      },
    })
    let resumed = false
    await renderInWorkspace(<ResumeDialog id="a1" name="Welcome series" overdue={412} onClose={() => {}} onResumed={() => (resumed = true)} />, { me: owner })
    const dialog = screen.getByRole('dialog', { name: 'Resume Welcome series?' })
    expect(dialog).toHaveTextContent('412 contacts had a wait end while the flow was paused')
    fireEvent.click(within(dialog).getByRole('radio', { name: /Spread over/ }))
    fireEvent.change(within(dialog).getByRole('spinbutton', { name: 'Hours' }), { target: { value: '8' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Resume' }))
    await waitFor(() => expect(bodies).toEqual([{ overdue: 'spread', spreadHours: 8 }]))
    expect(resumed).toBe(true)
  })

  it('AT-2: lists the contacts at a step a page at a time and takes one out with a reason', async () => {
    const removed: unknown[] = []
    unmock = mockApi({
      [`GET ${ws}/automations/a1/steps/d/contacts`]: (r) => {
        const cursor = new URL(r.url).searchParams.get('cursor')
        const item = (n: number) => ({ runId: `r${n}`, contactId: `c${n}`, email: `reader${n}@example.com`, name: `Reader ${n}`, since: '2026-10-30T09:12:00Z', until: '2026-11-01T09:12:00Z', wait: 'delay', enteredAt: '2026-10-30T09:00:00Z' })
        return cursor ? json({ total: 3, items: [item(3)] }) : json({ total: 3, items: [item(1), item(2)], next: 'p2' })
      },
      [`POST ${ws}/automations/a1/runs/r1/remove`]: async (r) => {
        removed.push(await r.json())
        return noContent()
      },
    })
    await renderInWorkspace(<StepContactsButton automationId="a1" automationName="Welcome series" nodeId="d" step="Wait 2 days" here={3} />, { me: owner })
    fireEvent.click(screen.getByRole('button', { name: 'See the 3 contacts here' }))
    const dialog = await screen.findByRole('dialog', { name: 'Wait 2 days · 3 contacts' })
    expect(await within(dialog).findByText('Reader 1')).toBeInTheDocument()
    expect(within(dialog).getByText('2 of 3')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Show more' }))
    expect(await within(dialog).findByText('Reader 3')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove Reader 1 from the flow' }))
    const confirm = await screen.findByRole('dialog', { name: 'Remove Reader 1 from Welcome series?' })
    fireEvent.click(within(confirm).getByRole('button', { name: 'Remove from flow' }))
    expect(await within(confirm).findByText('Say why, for the audit log.')).toBeInTheDocument()
    fireEvent.change(within(confirm).getByRole('textbox', { name: 'Why (kept in the audit log)' }), { target: { value: 'Asked to stop' } })
    fireEvent.click(within(confirm).getByRole('button', { name: 'Remove from flow' }))
    await waitFor(() => expect(removed).toEqual([{ reason: 'Asked to stop' }]))
  })

  it("AT-2: the profile's Automations card says where the contact is", async () => {
    unmock = mockApi({
      [`GET ${ws}/contacts/c1/automations`]: () =>
        json({
          items: [
            { runId: 'r1', automationId: 'a1', name: 'Welcome series', status: 'active', step: 'Wait 2 days', nodeId: 'd', enteredAt: '2026-10-30T09:00:00Z', stepSince: '2026-10-31T09:12:00Z' },
            { runId: 'r0', automationId: 'a2', name: 'Re-engagement', status: 'goal', step: 'Clicked', nodeId: 'g', enteredAt: '2026-09-01T09:00:00Z', stepSince: '2026-09-12T09:00:00Z', finishedAt: '2026-09-12T09:00:00Z' },
          ],
        }),
    })
    await renderInWorkspace(<ContactAutomationsCard contactId="c1" name="Maya Patel" />, { me: owner })
    expect(await screen.findByRole('heading', { name: 'Automations · 2' })).toBeInTheDocument()
    expect(screen.getByText(/At "Wait 2 days" since/)).toBeInTheDocument()
    expect(screen.getByText('In progress')).toBeInTheDocument()
    expect(screen.getAllByText('Reached the goal').length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: 'Remove Maya Patel from the flow' })).toHaveLength(1)
  })

  it('AT-11: adds, changes and removes further triggers', async () => {
    unmock = mockApi({})
    let last: TriggerConfig | undefined
    function Harness() {
      const [t, setT] = useState<TriggerConfig>({ kind: 'joined_list', listId: 'l1' })
      last = t
      return <AlsoTriggers t={t} set={(p) => setT({ ...t, ...p })} disabled={false} options={undefined} />
    }
    await renderInWorkspace(<Harness />, { me: owner })
    fireEvent.click(screen.getByRole('button', { name: 'Add another trigger' }))
    fireEvent.change(screen.getByRole('combobox', { name: 'Tag' }), { target: { value: 'vip' } })
    expect(last?.also).toEqual([{ kind: 'tag_added', tag: 'vip' }])
    fireEvent.click(screen.getByRole('button', { name: 'Add another trigger' }))
    expect(screen.queryByRole('button', { name: 'Add another trigger' })).not.toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button', { name: 'Remove trigger' })[1]!)
    fireEvent.click(screen.getByRole('button', { name: 'Remove trigger' }))
    expect(last?.also).toBeUndefined()
  })

  it('AT-9: sets the marketing cap', async () => {
    const bodies: unknown[] = []
    unmock = mockApi({
      [`PUT ${ws}/marketing-cap`]: async (r) => {
        bodies.push(await r.json())
        return json({})
      },
    })
    await renderInWorkspace(<MarketingCapCard cap={{ count: 0, days: 7 }} read={false} onSaved={async () => {}} />, { me: owner })
    fireEvent.click(screen.getByRole('switch', { name: 'Marketing cap' }))
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Emails' }), { target: { value: '60' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save cap' }))
    expect(await screen.findByText('Enter the emails from 1 to 50 and the days from 1 to 90.')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Emails' }), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save cap' }))
    await waitFor(() => expect(bodies).toEqual([{ count: 3, days: 7 }]))
  })

  it('AT-13: the sequence shows how many go on to the next email', async () => {
    unmock = mockApi({})
    const graph: FlowGraph = {
      nodes: [
        { id: 'm1', type: 'email', position: { x: 0, y: 0 }, data: { label: 'Step 1' } },
        { id: 'm2', type: 'email', position: { x: 0, y: 0 }, data: { label: 'Step 2' } },
      ],
      edges: [],
    }
    await renderInWorkspace(
      <SequenceStats
        graph={graph}
        refs={emptyRefs}
        sequence={[
          { nodeId: 'm1', sent: 100, delivered: 98, openedHuman: 60, clicked: 14, unsubscribed: 1, next: 92 },
          { nodeId: 'm2', sent: 92, delivered: 90, openedHuman: 44, clicked: 9, unsubscribed: 0 },
        ]}
      />,
      { me: owner },
    )
    expect(screen.getByRole('list', { name: 'The sequence' })).toBeInTheDocument()
    expect(screen.getByText('92% went on to the next email')).toBeInTheDocument()
    expect(screen.getByText('Last email')).toBeInTheDocument()
  })
})
