import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { uploadFile } from '@/features/products/upload'
import type { SignupForm } from './common'
import { dayOptions, FormDownloadSection, formUploads } from './LeadMagnet'
import { FormPagesSection, OwnOriginsNote } from './UsedOnPages'

const base = '/api/v1/workspaces/acme-newsletter/forms/f1'
const form = { id: 'f1', name: 'DNS checklist' } as SignupForm

/** The DNS records checklist the sample data offers (a PDF, 2 pages, 380 KB). */
const offered = (patch: Partial<Schemas['FormDownload']> = {}): Schemas['FormDownload'] => ({
  file: {
    id: 'file1',
    name: 'dns-records-checklist.pdf',
    kind: 'pdf',
    kindLabel: 'PDF',
    contentType: 'application/pdf',
    pages: 2,
    sizeBytes: 389120,
    uploadedAt: '2026-09-26T13:20:00Z',
    uploadedBy: 'Sofia Rossi',
  },
  days: 7,
  links: 1032,
  downloaded: 1019,
  ...patch,
})

const editor = workspaceMe({ permissions: ['forms:edit', 'subscribers:view'], workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' } })

afterEach(() => mockApi({}))

describe('After confirmation: offer a download', () => {
  it('shows the file, how long links work and the numbers', async () => {
    mockApi({ [`GET ${base}/download`]: () => json(offered()) })
    await renderInWorkspace(<FormDownloadSection form={form} readOnly={false} doubleOptIn />, { me: editor })
    expect(await screen.findByText('dns-records-checklist.pdf')).toBeInTheDocument()
    expect(screen.getByText('PDF · 2 pages · 380 KB')).toBeInTheDocument()
    expect(screen.getByText('1,032 links · 1,019 downloaded')).toBeInTheDocument()
    expect(screen.getByLabelText('Links work for')).toHaveValue('7')
    expect(screen.getByRole('button', { name: 'Replace dns-records-checklist.pdf' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Stop offering dns-records-checklist.pdf' })).toBeInTheDocument()
    expect(screen.queryByText(/needs double opt-in/)).not.toBeInTheDocument()
  })

  it('is read-only without forms:edit', async () => {
    mockApi({ [`GET ${base}/download`]: () => json(offered()) })
    await renderInWorkspace(<FormDownloadSection form={form} readOnly doubleOptIn />)
    expect(await screen.findByText('Links work for 7 days.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Replace/ })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Choose the file to offer')).not.toBeInTheDocument()
  })

  it('offers an upload when there is no file, and says double opt-in is needed for one', async () => {
    mockApi({ [`GET ${base}/download`]: () => json({ days: 7, links: 0, downloaded: 0 }) })
    await renderInWorkspace(<FormDownloadSection form={form} readOnly={false} doubleOptIn={false} />, { me: editor })
    expect(await screen.findByRole('button', { name: 'Upload a file' })).toBeInTheDocument()
    expect(screen.queryByText(/needs double opt-in/)).not.toBeInTheDocument()
  })

  it('changes how long links work', async () => {
    let sent: unknown
    mockApi({
      [`GET ${base}/download`]: () => json(offered()),
      [`PUT ${base}/download`]: async (r) => {
        sent = await r.json()
        return json(offered({ days: 14 }))
      },
    })
    await renderInWorkspace(<FormDownloadSection form={form} readOnly={false} doubleOptIn />, { me: editor })
    fireEvent.change(await screen.findByLabelText('Links work for'), { target: { value: '14' } })
    await waitFor(() => expect(screen.getByLabelText('Links work for')).toHaveValue('14'))
    expect(sent).toEqual({ days: 14 })
  })

  it('removes the file after asking, and reads the download again', async () => {
    let file = true
    mockApi({
      [`GET ${base}/download`]: () => json(file ? offered() : offered({ file: undefined, links: 1032 })),
      [`DELETE ${base}/download`]: () => {
        file = false
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<FormDownloadSection form={form} readOnly={false} doubleOptIn />, { me: editor })
    fireEvent.click(await screen.findByRole('button', { name: 'Stop offering dns-records-checklist.pdf' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('People who confirm no longer get a download, and the 1,032 links already sent stop working.')
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove the file' }))
    expect(await screen.findByRole('button', { name: 'Upload a file' })).toBeInTheDocument()
  })

  it('lists the days a link may work, keeping an unusual current value', () => {
    expect(dayOptions(7)).toEqual([1, 2, 3, 5, 7, 10, 14, 21, 30])
    expect(dayOptions(4)).toEqual([1, 2, 3, 4, 5, 7, 10, 14, 21, 30])
  })
})

describe('Uploading the file in chunks', () => {
  it('starts, sends raw bytes at each offset and finishes', async () => {
    const calls: string[] = []
    mockApi({
      [`POST ${base}/download/uploads`]: async (r) => {
        calls.push(`start ${JSON.stringify(await r.json())}`)
        return json({ id: 'u1', received: 0, size: 6, chunkBytes: 4 }, 201)
      },
      [`PUT ${base}/download/uploads/u1`]: async (r) => {
        const body = new Uint8Array(await r.arrayBuffer())
        const offset = Number(new URL(r.url).searchParams.get('offset'))
        calls.push(`chunk ${offset} ${body.length} ${r.headers.get('content-type')}`)
        return json({ id: 'u1', received: offset + body.length, size: 6, chunkBytes: 4 })
      },
      [`POST ${base}/download/uploads/u1/complete`]: () => {
        calls.push('finish')
        return json(offered())
      },
    })
    const file = Object.assign(new Blob([new Uint8Array(6)]), { name: 'guide.pdf' })
    const out = await uploadFile(formUploads('acme-newsletter', 'f1'), file, { wait: async () => {} })
    expect(calls).toEqual(['start {"name":"guide.pdf","size":6}', 'chunk 0 4 application/octet-stream', 'chunk 4 2 application/octet-stream', 'finish'])
    expect(out.file?.name).toBe('dns-records-checklist.pdf')
  })
})

describe('Used on pages and the site’s own pages', () => {
  it('lists the pages that use the form, each opening in the page builder', async () => {
    mockApi({
      [`GET ${base}/pages`]: () =>
        json({
          items: [
            { id: 'p2', name: 'DNS records checklist', path: '/dns-checklist', url: 'https://news.acme.io/dns-checklist', status: 'published', live: true },
            { id: 'p9', name: 'Launch notes', path: '/launch-notes', url: 'https://news.acme.io/launch-notes', status: 'published', live: false },
          ],
        }),
    })
    await renderInWorkspace(<FormPagesSection form={form} />, { me: editor })
    expect(await screen.findByRole('link', { name: 'Open DNS records checklist in the page builder' })).toHaveAttribute('href', '/w/acme-newsletter/forms/pages/p2')
    expect(screen.getByText('/dns-checklist')).toBeInTheDocument()
    expect(screen.getByText('/launch-notes · in its unpublished changes')).toBeInTheDocument()
  })

  it('says when no page uses the form', async () => {
    mockApi({ [`GET ${base}/pages`]: () => json({ items: [] }) })
    await renderInWorkspace(<FormPagesSection form={form} />, { me: editor })
    expect(await screen.findByText(/No page uses this form yet/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Pages' })).toHaveAttribute('href', '/w/acme-newsletter/forms/pages')
  })

  it('names the hosts that are always allowed', async () => {
    await renderInWorkspace(<OwnOriginsNote origins={['news.acme.io', 'reputecast.acme.io']} />)
    expect(screen.getByText('Your own pages on news.acme.io and reputecast.acme.io are always allowed.')).toBeInTheDocument()
  })
})
