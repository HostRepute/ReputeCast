import { fireEvent, screen, within } from '@testing-library/react'
import { useState } from 'react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { GrowthSourceField } from './ContactsScore'

function Harness({ initial }: { initial?: string }) {
  const [value, setValue] = useState(initial)
  return (
    <>
      <GrowthSourceField value={value} onChange={setValue} open />
      <output>{value ?? 'any'}</output>
    </>
  )
}

describe('GrowthSourceField', () => {
  afterEach(() => mockApi({})())

  // Review 2026-11 EN-7: the Source filter offers this workspace's own
  // sources, never the demo's.
  it("lists the workspace's own sources with how many joined", async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/growth/sources': () =>
        json({
          items: [
            { contacts: 1204, source: { key: 'person:', kind: 'person', name: '', hosts: [], imported: false } },
            { contacts: 87, source: { key: 'subscribe_page:', kind: 'subscribe_page', name: '', hosts: [], imported: false } },
          ],
        }),
    })
    await renderInWorkspace(<Harness />)
    const select = screen.getByLabelText('Source')
    await within(select).findByRole('option', { name: 'Added by a person · 1,204' })
    const names = within(select)
      .getAllByRole('option')
      .map((o) => o.textContent)
    expect(names).toHaveLength(3)
    expect(names.join()).not.toMatch(/Brevo|Mailchimp|docs/)
    fireEvent.change(select, { target: { value: 'subscribe_page:' } })
    expect(screen.getByRole('status')).toHaveTextContent('subscribe_page:')
  })

  it('keeps a filter from a link before the list loads', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/growth/sources': () => new Promise<Response>(() => {}) })
    await renderInWorkspace(<Harness initial="form:0192" />)
    expect(screen.getByLabelText('Source')).toHaveValue('form:0192')
  })
})
