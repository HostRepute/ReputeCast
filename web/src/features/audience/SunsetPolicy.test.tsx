import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { components } from '@/api/schema'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { PolicyCard } from './HealthPage'

type Health = components['schemas']['ListHealth']
type Policy = components['schemas']['SunsetPolicy']

const ws = '/api/v1/workspaces/acme-newsletter'
const manager = workspaceMe({ permissions: ['subscribers:view', 'lists:manage'] })
const policy: Policy = {
  enabled: true,
  enabledAt: '2026-09-01',
  activeDays: 30,
  atRiskDays: 91,
  sunsetDays: 181,
  countMachineOpens: false,
  reengage: true,
  reengageAutomation: 'Re-engagement',
  stopBroadcasts: true,
  stopAfterDays: 30,
  deleteAfter: false,
  deleteAfterMonths: 12,
}
const health = (p: Policy) => ({ policy: p, thisWeek: { entering: 0, stopping: 0 } }) as unknown as Health

describe('Sunset policy (review 2026-11 AU-12)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('shows the dry run before erasing starts and sends the typed word', async () => {
    const saved: Policy[] = []
    const dryRuns: string[] = []
    unmock = mockApi({
      [`GET ${ws}/audience/sunset-policy/dry-run`]: (r) => {
        dryRuns.push(new URL(r.url).searchParams.get('months') ?? '')
        return json({ erase: 1284, kept: 6, from: '2026-12-01' })
      },
      [`PUT ${ws}/audience/sunset-policy`]: async (r) => {
        const body = (await r.json()) as Policy
        saved.push(body)
        return json(body)
      },
      [`GET ${ws}/audience/health`]: () => json({}),
    })
    await renderInWorkspace(<PolicyCard h={health(policy)} />, { path: '/w/acme-newsletter/audience/health', me: manager })

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Months in Sunset' }), { target: { value: '18' } })
    fireEvent.click(screen.getByRole('checkbox', { name: /Erase the contact after/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Save policy' }))

    const dialog = await screen.findByRole('dialog', { name: 'Turn on erasing?' })
    expect(await within(dialog).findByText('1,284 contacts')).toBeInTheDocument()
    expect(within(dialog).getByText(/have been in Sunset for 18 months or more and would be erased on 1 December/)).toBeInTheDocument()
    expect(within(dialog).getByText(/The 6 paid members are kept until their subscription ends/)).toBeInTheDocument()
    expect(dryRuns).toEqual(['18'])
    expect(saved).toHaveLength(0)

    const confirm = within(dialog).getByRole('button', { name: 'Turn on erasing' })
    expect(confirm).toHaveAttribute('aria-disabled', 'true')
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Type ERASE to confirm' }), { target: { value: 'ERASE' } })
    fireEvent.click(confirm)
    await waitFor(() => expect(saved).toHaveLength(1))
    expect(saved[0]).toMatchObject({ deleteAfter: true, deleteAfterMonths: 18, confirmErase: 'ERASE', enabled: true })
  })

  it('switches the policy off and edits the days without a dry run', async () => {
    const saved: Policy[] = []
    unmock = mockApi({
      [`PUT ${ws}/audience/sunset-policy`]: async (r) => {
        const body = (await r.json()) as Policy
        saved.push(body)
        return json(body)
      },
      [`GET ${ws}/audience/health`]: () => json({}),
    })
    await renderInWorkspace(<PolicyCard h={health({ ...policy, deleteAfter: true })} />, { path: '/w/acme-newsletter/audience/health', me: manager })
    fireEvent.click(screen.getByRole('switch', { name: 'Sunset policy' }))
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Days after entering Sunset' }), { target: { value: '45' } })
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Months in Sunset' }), { target: { value: '24' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save policy' }))
    await waitFor(() => expect(saved).toHaveLength(1))
    expect(saved[0]).toMatchObject({ enabled: false, stopAfterDays: 45, deleteAfterMonths: 24 })
    expect(saved[0]!.confirmErase).toBeUndefined()
  })

  it('asks for a number of days it can use', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<PolicyCard h={health(policy)} />, { path: '/w/acme-newsletter/audience/health', me: manager })
    fireEvent.change(screen.getByRole('spinbutton', { name: 'Days after entering Sunset' }), { target: { value: '0' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save policy' }))
    expect(await screen.findByText('Enter the days before Broadcasts stop as a number from 1 to 365.')).toBeInTheDocument()
  })
})
