import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { forgetJob, resetLocalJobs, setJobsSource, stubJobsSource, trackJob, updateJob, useBackgroundJobs, type BackgroundJob } from './jobs'

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>
)

const importing: BackgroundJob = { id: 'job-1', kind: 'import', title: 'Importing 12,480 contacts from Mailchimp', status: 'running', progress: 68, eta: '1 min left' }

describe('background jobs', () => {
  afterEach(() => {
    resetLocalJobs()
    setJobsSource(stubJobsSource)
  })

  it('shows jobs this page started until the server knows them', async () => {
    trackJob('acme-newsletter', importing)
    const { result } = renderHook(() => useBackgroundJobs('acme-newsletter'), { wrapper })
    expect(result.current.active.map((j) => j.id)).toEqual(['job-1'])
    updateJob('acme-newsletter', 'job-1', { status: 'done', detail: '12,480 contacts imported' })
    await waitFor(() => expect(result.current.active).toEqual([]))
    forgetJob('acme-newsletter', 'job-1')
    await waitFor(() => expect(result.current.jobs).toEqual([]))
  })

  it("merges the server's list, which wins for the same job", async () => {
    trackJob('acme-newsletter', { ...importing, progress: 10 })
    setJobsSource({
      list: async () => [
        { ...importing, progress: 70 },
        { id: 'job-2', kind: 'export', title: 'Export · contacts with activity', status: 'queued' },
      ],
    })
    const { result } = renderHook(() => useBackgroundJobs('acme-newsletter'), { wrapper })
    await waitFor(() => expect(result.current.jobs).toHaveLength(2))
    expect(result.current.jobs.find((j) => j.id === 'job-1')?.progress).toBe(70)
    expect(result.current.active).toHaveLength(2)
  })

  it('keeps workspaces apart', () => {
    trackJob('acme-newsletter', importing)
    const { result } = renderHook(() => useBackgroundJobs('acme-engineering-blog'), { wrapper })
    expect(result.current.jobs).toEqual([])
  })
})
