import { screen, within } from '@testing-library/react'
import { mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { UsesList } from './UsesList'

describe('What uses a field or segment (review 2026-11 AU-18, AU-19)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('opens a saved view at its own address and a transactional template in its editor', async () => {
    unmock = mockApi({})
    await renderInWorkspace(
      <UsesList
        uses={[
          { kind: 'view', id: '0192f000-0000-7000-8000-000000000001', name: 'Team accounts' },
          { kind: 'transactional', id: 'tpl-1', name: 'Plan receipt' },
        ]}
      />,
    )
    const [view, template] = screen.getAllByRole('listitem') as [HTMLElement, HTMLElement]
    expect(within(view).getByText('Saved view')).toBeInTheDocument()
    expect(within(view).getByRole('link', { name: 'Open' })).toHaveAttribute('href', '/w/acme-newsletter/audience?view=0192f000-0000-7000-8000-000000000001')
    expect(within(template).getByText('Transactional template')).toBeInTheDocument()
    expect(within(template).getByRole('link', { name: 'Open' })).toHaveAttribute('href', '/w/acme-newsletter/transactional/templates/tpl-1')
  })
})
