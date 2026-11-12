import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { QueryClient } from '@tanstack/react-query'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { meQueryKey } from '@/lib/session'
import { SystemPage, waited } from './SystemPage'

type Job = Schemas['SystemJob']

const view: Schemas['SystemView'] = {
  jobs: { waiting: 128, scheduled: 1277, running: 9, retrying: 1, givenUp: 1, lagSeconds: 4 },
  queues: [
    { name: 'sending', waiting: 96, scheduled: 1204, running: 6, lagSeconds: 4 },
    { name: 'events', waiting: 31, scheduled: 12, running: 2, lagSeconds: 1 },
    { name: 'system', waiting: 0, scheduled: 3, running: 0 },
  ],
  database: {
    version: '16.4 (Debian 16.4-1)',
    bytes: 4.2 * 1024 ** 3,
    connections: 18,
    maxConnections: 100,
    schemaVersion: 3384,
    schemaCurrent: true,
    tables: [{ name: 'delivery_events', rows: 18_400_000, bytes: 2.1 * 1024 ** 3, indexBytes: 812 * 1024 ** 2 }],
  },
  files: { backend: 'local', dir: '/var/lib/reputecast/files', files: 2418, bytes: 1.3 * 1024 ** 3, countedAt: '2026-11-01T09:40:00Z' },
}
const job = (o: Partial<Job>): Job => ({ id: 48213, kind: 'webhooks.deliver', queue: 'events', state: 'retrying', attempt: 4, maxAttempts: 12, error: 'dial tcp 203.0.113.40:443: i/o timeout', failedAt: '2026-11-01T09:38:00Z', nextAt: '2026-11-01T09:42:00Z', workspace: { id: 'ws-1', name: 'Acme Newsletter', slug: 'acme-newsletter' }, ...o })
const jobs = [job({}), job({ id: 46102, kind: 'digests.build_issue', queue: 'default', state: 'given_up', attempt: 12, error: 'feed https://acme.io/blog/rss: 404 Not Found', nextAt: undefined, workspace: undefined })]

/** Linh, an instance admin. */
const admin = (qc: QueryClient) => {
  qc.setQueryData(meQueryKey, {
    user: { id: 'u-1', name: 'Linh Nguyen', email: 'linh@acme.io', isInstanceAdmin: true },
    workspaces: [{ id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleName: 'Owner' }],
    instance: { name: 'Acme ReputeCast', version: 'v1.4.2' },
  } as never)
  qc.setQueryData(['meta'], { instanceName: 'Acme ReputeCast', publicUrl: 'https://reputecast.acme.io', version: 'v1.4.2' } as never)
}

describe('Instance admin › System', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('shows the jobs, queues, largest tables and file storage', async () => {
    unmock = mockApi({
      'GET /api/v1/admin/system': () => json(view),
      'GET /api/v1/admin/system/jobs': () => json({ items: jobs }),
    })
    await renderInWorkspace(<SystemPage />, { path: '/w/acme-newsletter/admin/system', seed: admin })
    expect(await screen.findByText('oldest due job waited 4 s')).toBeInTheDocument()
    expect(screen.getByText('4.2 GB')).toBeInTheDocument()
    expect(screen.getByText('PostgreSQL 16.4 · 18 of 100 connections')).toBeInTheDocument()
    expect(screen.getByText('1 retrying, 1 given up')).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Failed jobs · 2' })).toBeInTheDocument()
    const failed = screen.getByRole('table', { name: 'Failed jobs' })
    const rows = within(failed).getAllByRole('row')
    expect(within(rows[1]!).getByText('webhooks.deliver')).toBeInTheDocument()
    expect(within(rows[1]!).getByText('Acme Newsletter')).toBeInTheDocument()
    expect(within(rows[1]!).getByText('4 of 12')).toBeInTheDocument()
    expect(within(rows[2]!).getByText('Instance')).toBeInTheDocument()
    expect(within(rows[2]!).getByText('Given up')).toBeInTheDocument()
    const queues = screen.getByRole('table', { name: 'Queues' })
    expect(within(queues).getByText('Campaign and automation sends')).toBeInTheDocument()
    expect(within(queues).getAllByRole('row')[3]).toHaveTextContent('—')
    expect(screen.getByText('4.2 GB in all · schema 3384, current')).toBeInTheDocument()
    expect(screen.getByText('2,418 files · 1.3 GB')).toBeInTheDocument()
    const files = screen.getByRole('region', { name: 'File storage' })
    expect(within(files).getByText('Local folder')).toBeInTheDocument()
    expect(within(files).getByText('/var/lib/reputecast/files')).toBeInTheDocument()
    expect(within(files).getByText('Every web and worker process needs the same folder.')).toBeInTheDocument()
  })

  it('shows the bucket when files are in object storage, never its keys', async () => {
    const bucket = { ...view, files: { backend: 's3' as const, bucket: 'acme-reputecast', prefix: 'prod/', endpoint: 'https://s3.eu-central-003.backblazeb2.com', files: 2418, bytes: 1.3 * 1024 ** 3, countedAt: '2026-11-01T09:40:00Z' } }
    unmock = mockApi({
      'GET /api/v1/admin/system': () => json(bucket),
      'GET /api/v1/admin/system/jobs': () => json({ items: [] }),
    })
    await renderInWorkspace(<SystemPage />, { path: '/w/acme-newsletter/admin/system', seed: admin })
    const files = await screen.findByRole('region', { name: 'File storage' })
    expect(within(files).getByText('Uploads live in this bucket, not in the database: back the bucket up with it')).toBeInTheDocument()
    expect(within(files).getByText('S3-compatible bucket')).toBeInTheDocument()
    expect(within(files).getByText('acme-reputecast')).toBeInTheDocument()
    expect(within(files).getByText('prod/')).toBeInTheDocument()
    expect(within(files).getByText('https://s3.eu-central-003.backblazeb2.com')).toBeInTheDocument()
    expect(within(files).getByText('Every web and worker process reads the same bucket.')).toBeInTheDocument()
    expect(within(files).queryByText('Folder')).not.toBeInTheDocument()
  })

  it('names AWS when the bucket has no endpoint of its own', async () => {
    const aws = { ...view, files: { backend: 's3' as const, bucket: 'acme-files', region: 'eu-west-1', files: 0, bytes: 0, countedAt: '2026-11-01T09:40:00Z' } }
    unmock = mockApi({
      'GET /api/v1/admin/system': () => json(aws),
      'GET /api/v1/admin/system/jobs': () => json({ items: [] }),
    })
    await renderInWorkspace(<SystemPage />, { path: '/w/acme-newsletter/admin/system', seed: admin })
    const files = await screen.findByRole('region', { name: 'File storage' })
    expect(within(files).getByText('AWS S3 · eu-west-1')).toBeInTheDocument()
    expect(within(files).queryByText('Prefix')).not.toBeInTheDocument()
    expect(screen.getByText('nothing uploaded yet')).toBeInTheDocument()
  })

  it('retries a job, and cancels or deletes one after asking', async () => {
    const posted: string[] = []
    unmock = mockApi({
      'GET /api/v1/admin/system': () => json(view),
      'GET /api/v1/admin/system/jobs': () => json({ items: jobs }),
      'POST /api/v1/admin/system/jobs/48213/retry': () => {
        posted.push('retry 48213')
        return json({ id: 48213, state: 'available' })
      },
      'POST /api/v1/admin/system/jobs/46102/discard': () => {
        posted.push('discard 46102')
        return json({ id: 46102, state: 'deleted' })
      },
    })
    await renderInWorkspace(<SystemPage />, { path: '/w/acme-newsletter/admin/system', seed: admin })
    fireEvent.click(await screen.findByRole('button', { name: 'Retry webhooks.deliver #48213' }))
    await waitFor(() => expect(posted).toEqual(['retry 48213']))
    expect(await screen.findByText('webhooks.deliver #48213 runs again')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Delete digests.build_issue #46102' }))
    const dialog = await screen.findByRole('dialog', { name: 'Delete digests.build_issue #46102?' })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete job' }))
    await waitFor(() => expect(posted).toEqual(['retry 48213', 'discard 46102']))
  })

  it('says when nothing failed', async () => {
    unmock = mockApi({
      'GET /api/v1/admin/system': () => json({ ...view, jobs: { ...view.jobs, retrying: 0, givenUp: 0 } }),
      'GET /api/v1/admin/system/jobs': () => json({ items: [] }),
    })
    await renderInWorkspace(<SystemPage />, { path: '/w/acme-newsletter/admin/system', seed: admin })
    expect(await screen.findByText('No failed jobs. Jobs that fail show here with their error.')).toBeInTheDocument()
  })

  it('says how long a job waited', () => {
    expect(waited(4.4)).toBe('4 s')
    expect(waited(125)).toBe('2 min 05 s')
    expect(waited(4320)).toBe('1 h 12 min')
  })
})
