import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { dismissToast } from '@/components/ui/Toast'
import { resetLocalJobs, trackJob, updateJob } from '@/lib/jobs'
import { renderInWorkspace } from '@/test/app'
import { JobsTray } from './JobsTray'

afterEach(() => {
  act(() => {
    resetLocalJobs()
    dismissToast()
  })
})

describe('JobsTray', () => {
  it('stays hidden while nothing runs', async () => {
    await renderInWorkspace(<JobsTray />)
    expect(screen.queryByRole('region', { name: 'Background jobs' })).not.toBeInTheDocument()
  })

  it('shows running jobs, collapses, and says when each one ends', async () => {
    trackJob('acme-newsletter', { id: 'job-1', kind: 'import', title: 'Importing 12,480 contacts from Mailchimp', status: 'running', progress: 68, eta: '1 min left' })
    trackJob('acme-newsletter', { id: 'job-2', kind: 'export', title: 'Export · contacts with activity', status: 'queued' })
    const { router } = await renderInWorkspace(<JobsTray />)

    const tray = await screen.findByRole('region', { name: 'Background jobs' })
    expect(tray).toHaveTextContent('Background jobs · 2')
    expect(tray).toHaveTextContent('68% · 1 min left')
    expect(tray).toHaveTextContent('queued')
    expect(tray).toHaveTextContent('Jobs keep running if you close the tab.')
    expect(screen.getByRole('progressbar', { name: 'Importing 12,480 contacts from Mailchimp' })).toHaveAttribute('aria-valuenow', '68')
    expect(screen.getByRole('link', { name: 'History' })).toHaveAttribute('href', '/w/acme-newsletter/settings/audit')

    const toggle = screen.getByRole('button', { name: /Background jobs · 2/ })
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()

    act(() => updateJob('acme-newsletter', 'job-1', { status: 'done', detail: '12,480 contacts imported', href: '/w/acme-newsletter/audience' }))
    const done = await screen.findByRole('status')
    expect(done).toHaveTextContent('Import finished')
    expect(done).toHaveTextContent('12,480 contacts imported')
    await waitFor(() => expect(tray).toHaveTextContent('Background jobs · 1'))
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Open' })))
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/audience')

    act(() => updateJob('acme-newsletter', 'job-2', { status: 'failed', detail: 'the export query timed out' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Export · contacts with activity failed')
    await waitFor(() => expect(screen.queryByRole('region', { name: 'Background jobs' })).not.toBeInTheDocument())
  })
})
