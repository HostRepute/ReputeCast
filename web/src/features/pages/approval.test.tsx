import { screen, within } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import type { Schemas } from '@/api/client'
import { setDisplayTimeZone } from '@/lib/format'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { PageApprovalCard, PageBadge, PageDecisionNote, PageLine, pageScheduledText } from './approval'

const page = (patch: Partial<Schemas['ApprovalPageSnapshot']> = {}): Schemas['ApprovalPageSnapshot'] => ({
  pageId: 'p6',
  name: 'Sponsor the Weekly Digest',
  template: 'blank',
  slug: 'sponsor',
  path: '/sponsor',
  visibility: 'public',
  visibilityLabel: 'Public',
  when: 'now',
  published: false,
  ...patch,
})

const approver = workspaceMe({ timezone: 'UTC', permissions: ['campaigns:approve', 'website:view'], workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' } })

beforeAll(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2026-10-20T12:00:00Z'))
  setDisplayTimeZone('UTC')
})
afterAll(() => {
  vi.useRealTimers()
  setDisplayTimeZone(undefined)
})
afterEach(() => mockApi({}))

describe('Pages in the approval inbox', () => {
  it('says when a page goes live', () => {
    expect(pageScheduledText(page())).toBe('When approved')
    expect(pageScheduledText(page({ when: 'at', publishAt: '2026-10-24T13:00:00Z' }))).toBe('Oct 24, 13:00')
    expect(pageScheduledText(page({ published: true }))).toBe('Published')
  })

  it('badges the request and names its address and visibility', async () => {
    await renderInWorkspace(
      <>
        <PageBadge page={page()} />
        <PageLine page={page({ visibility: 'members', visibilityLabel: 'Members' })} />
      </>,
      { me: approver },
    )
    expect(screen.getByRole('link', { name: 'Page: open “Sponsor the Weekly Digest” in the page builder' })).toHaveAttribute('href', '/w/acme-newsletter/forms/pages/p6')
    expect(screen.getByText('Page · /sponsor · Members')).toBeInTheDocument()
  })
})

describe('A page publish request in review', () => {
  const detail = (patch: Partial<Schemas['ApprovalDetail']> = {}) =>
    ({ id: 'a1', name: 'Sponsor the Weekly Digest', subjectType: 'website_page', version: 3, page: page(), ...patch }) as Schemas['ApprovalDetail']

  it('shows what the approval publishes and the page as readers will see it', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/approvals/a1/preview': () =>
        json({ subject: 'Sponsor the Weekly Digest', preheader: '', html: '<!doctype html><html><body><h1>Sponsor the Weekly Digest</h1></body></html>', version: 3, language: 'en', languages: [] }),
    })
    await renderInWorkspace(<PageApprovalCard detail={detail()} />, { me: approver })
    const card = screen.getByRole('region', { name: 'Page' })
    expect(card).toHaveTextContent('Sponsor the Weekly Digest · Blank')
    expect(card).toHaveTextContent('/sponsor')
    expect(card).toHaveTextContent('Public')
    expect(card).toHaveTextContent('As soon as it’s approved')
    expect(card).toHaveTextContent('v3')
    expect(within(card).getByRole('link', { name: 'Open in the page builder' })).toHaveAttribute('href', '/w/acme-newsletter/forms/pages/p6')
    expect(await within(card).findByTitle('Preview of “Sponsor the Weekly Digest”, v3')).toBeInTheDocument()
  })

  it('gives a scheduled page its time in the workspace’s zone', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/approvals/a1/preview': () => json({ status: 500, title: 'x', detail: 'The preview failed.' }, 500) })
    await renderInWorkspace(<PageApprovalCard detail={detail({ page: page({ when: 'at', publishAt: '2026-10-24T13:00:00Z' }) })} />, { me: approver })
    expect(screen.getByRole('region', { name: 'Page' })).toHaveTextContent('Oct 24, 2026 · 13:00 UTC')
    expect(await screen.findByText(/The preview isn’t available/)).toBeInTheDocument()
  })

  it('tells the approver what approving does', async () => {
    await renderInWorkspace(<PageDecisionNote author="Daniel Park" page={page({ when: 'at' })} />)
    expect(
      screen.getByText(
        'Approving covers this exact version: its content, address, visibility and publish time. It goes live at its publish time. If Daniel changes it first, publishing needs a new approval.',
      ),
    ).toBeInTheDocument()
  })
})
