import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { WhereWhenSection } from './FormTargeting'
import { emptyTargeting, targetingSummary, type Targeting } from './targeting'

// Form targeting by segment (M62 G-32): Where and when › Visitors offers
// "In a segment" and "Not in a segment", with the workspace's segments and
// the note that it needs Website tracking on the same site.

afterEach(() => mockApi({}))

const segments = { items: [{ id: 'seg-1', name: 'Active in last 30 days', mode: 'builder', refresh: 'dynamic' }] }

describe('targeting by segment', () => {
  it('needs a segment, says it needs Website tracking, and applies the segment', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/segments': () => json(segments) })
    let applied: Targeting | undefined
    await renderInWorkspace(<WhereWhenSection hosted={false} targeting={emptyTargeting()} geoip={false} locked={false} onChange={(t) => (applied = t)} />)
    fireEvent.click(screen.getByRole('button', { name: 'Edit where and when' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('radio', { name: 'In a segment' }))
    expect(within(dialog).getByText(/Needs Website tracking on the same site\. The form shows only/)).toBeTruthy()
    const apply = within(dialog).getByRole('button', { name: 'Apply' })
    expect(apply.getAttribute('aria-disabled')).toBe('true')
    const select = within(dialog).getByRole('combobox', { name: 'Segment' })
    await waitFor(() => expect(within(select).getByRole('option', { name: 'Active in last 30 days' })).toBeTruthy())
    fireEvent.change(select, { target: { value: 'seg-1' } })
    fireEvent.click(within(dialog).getByRole('radio', { name: 'Not in a segment' }))
    expect(within(dialog).getByText(/everyone else sees it/)).toBeTruthy()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Apply' }))
    expect(applied?.visitors).toBe('not_in_segment')
    expect(applied?.segmentId).toBe('seg-1')
  })

  it('summarises the rule, and drops the segment for other visitors', () => {
    const x: Targeting = { ...emptyTargeting(), visitors: 'in_segment', segmentId: 'seg-1' }
    expect(targetingSummary(x)).toBe('visitors in a segment')
    expect(targetingSummary({ ...x, visitors: 'not_in_segment' })).toBe('visitors not in a segment')
  })
})
