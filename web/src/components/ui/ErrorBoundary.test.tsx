import { fireEvent, render, screen } from '@testing-library/react'
import { Card } from './Card'
import { StatTile } from './Data'
import { TableScroll } from './Table'

let broken = true
function Fragile() {
  if (broken) throw new Error('bad data')
  return <p>Recovered</p>
}

describe('a failing part of a page', () => {
  beforeEach(() => vi.spyOn(console, 'error').mockImplementation(() => {}))
  afterEach(() => {
    vi.restoreAllMocks()
    broken = true
  })

  it('shows a notice in its own card; the rest of the page works', () => {
    render(
      <>
        <Card aria-label="Broken">
          <Fragile />
        </Card>
        <Card aria-label="Fine">
          <p>Still here</p>
        </Card>
      </>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent("This part couldn't show")
    expect(screen.getByText('Still here')).toBeInTheDocument()
    broken = false
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(screen.getByText('Recovered')).toBeInTheDocument()
  })

  it('keeps a stat tile to itself', () => {
    render(
      <>
        <StatTile label="Revenue" value={<Fragile />} />
        <StatTile label="Members" value="1,204" />
      </>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent("This part couldn't show")
    expect(screen.getByText('1,204')).toBeInTheDocument()
  })

  it('lets a tall table scroll with its header in view', () => {
    const { container } = render(
      <TableScroll label="Jobs" maxHeight={320}>
        <table>
          <thead>
            <tr>
              <th>Job</th>
            </tr>
          </thead>
        </table>
      </TableScroll>,
    )
    const scroller = container.querySelector('.tscroll')!
    expect(scroller).toHaveClass('tall')
    expect(scroller).toHaveStyle({ maxHeight: '320px' })
  })
})
