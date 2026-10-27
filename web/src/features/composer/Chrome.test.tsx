import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Component } from 'grapesjs'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { meQueryKey, type Me } from '@/lib/session'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { DismissControls, reasonsFor } from './Checks'
import { ComposerCtx, type Checks, type Issue } from './context'
import { headlessMjmlEditor } from './editor/testEditor'
import { ModeTabs, PermissionBanner, StatusChip, useReadiness } from './Header'
import { InboxStrip } from './InboxStrip'
import { StyleFields } from './Inspector'
import { QuickReviewBar } from './QuickReview'
import { Thumb } from './Thumb'
import { testCampaign, testComposer } from './testing'

// CO-14: the composer's chrome per the review boards.

const base = '/api/v1/workspaces/acme-newsletter/campaigns/01970000-0000-7000-8000-0000000000c1'
const editor = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'campaigns:send'] })

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

const issue = (patch: Partial<Issue>): Issue => ({ id: 'content.alt_text', severity: 'warn', group: 'Content', title: 'Hero image has no alt text', path: [1, 0, 0], section: 1, fixes: [], ...patch }) as Issue

const checks = (issues: Issue[], dismissed: Checks['dismissed'] = []): Checks =>
  ({
    verdict: 'warnings',
    blocks: issues.filter((i) => i.severity === 'block').length,
    warnings: issues.filter((i) => i.severity === 'warn').length,
    sizeBytes: 65_000,
    clipBytes: 102_400,
    images: 7,
    imagesMissingAlt: 1,
    links: 14,
    linksBlocklisted: 0,
    linksBroken: 0,
    subjectHeuristics: [],
    issues,
    sections: [],
    plainText: '',
    checkedAt: '2026-10-27T09:38:00Z',
    imageShare: 0.38,
    words: 300,
    linkList: [],
    lint: [],
    variants: [],
    issueKeys: issues.map((_, i) => `k${i}`),
    dismissed,
  }) as unknown as Checks

function Readiness() {
  const r = useReadiness()
  return <span data-testid="ready">{`${r.blocks}/${r.warnings}`}</span>
}

describe('top bar', () => {
  it('switches Write, Design and Code with clicks and arrow keys', async () => {
    const setView = vi.fn()
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ setView })}>
        <ModeTabs />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    const list = screen.getByRole('tablist', { name: 'Editing mode' })
    const tabs = within(list).getAllByRole('tab')
    expect(tabs.map((x) => x.textContent)).toEqual(['Write', 'Design', 'Code'])
    expect(within(list).getByRole('tab', { name: 'Design' })).toHaveAttribute('aria-selected', 'true')
    fireEvent.click(within(list).getByRole('tab', { name: 'Code' }))
    expect(setView).toHaveBeenLastCalledWith('mjml')
    fireEvent.keyDown(within(list).getByRole('tab', { name: 'Design' }), { key: 'ArrowLeft' })
    expect(setView).toHaveBeenLastCalledWith('write')
  })

  it('says the save state like the board, and counts warnings that are not dismissed', async () => {
    const two = checks([issue({}), issue({ id: 'content.size', title: '98 KB of Gmail’s 102 KB clipping limit' }), issue({ id: 'links.blocklist', severity: 'block', title: 'Blocklisted link' })], [
      { index: 1, dismissal: { id: 'd1', key: 'k1', checkId: 'content.size', reason: 'size', reasonText: 'we’ll stay under 102 KB', note: '', by: 'Linh Nguyen', at: '2026-10-27T09:40:00Z' } },
    ])
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ status: { kind: 'saved', at: '2026-10-27T09:38:00Z', version: 14 }, checks: two })}>
        <StatusChip word="draft" draft />
        <Readiness />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    expect(screen.getByRole('status')).toHaveTextContent('Saved 09:38 · v14')
    expect(screen.getByTestId('ready')).toHaveTextContent('1/1')
  })
})

describe('inbox strip', () => {
  it('sums up the subject, audience and route, and opens them with Edit', async () => {
    const campaign = testCampaign({ route: { source: 'campaign', shares: [{ providerId: 'p1', adapter: 'ses', short: 'SES', percent: 90 }, { providerId: 'p2', adapter: 'postal', short: 'Postal', percent: 10 }] } as never })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign })}>
        <InboxStrip />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    const strip = screen.getByRole('region', { name: 'Inbox' })
    expect(strip).toHaveTextContent('Subject: Three launches, one deprecation, and the roadmap for Q4')
    expect(strip).toHaveTextContent('To Weekly Digest · 31,192')
    expect(strip).toHaveTextContent('via SES 90% / Postal 10%')
    fireEvent.click(within(strip).getByRole('button', { name: 'Edit' }))
    const panel = screen.getByRole('region', { name: 'Subject, preheader and recipients' })
    expect(within(panel).getByText(/^55 characters · Gmail shows about 70 on desktop, 35 on iPhone/)).toBeInTheDocument()
    expect(within(panel).getByText('77 characters · shown after the subject in most inboxes')).toBeInTheDocument()
    expect(within(panel).getByRole('group', { name: 'Sending to' })).toBeInTheDocument()
    expect(within(strip).getByRole('button', { name: 'Done' })).toHaveAttribute('aria-expanded', 'true')
  })

  it('starts a subject test from the subject, with B starting as A (CO-6)', async () => {
    const setABTest = vi.fn()
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ setABTest })}>
        <InboxStrip phoneSheet />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    fireEvent.click(screen.getByRole('button', { name: 'A/B test the subject' }))
    expect(setABTest).toHaveBeenCalledWith(expect.objectContaining({ kind: 'subject', variants: [{ key: 'A' }, { key: 'B', subject: 'Three launches, one deprecation, and the roadmap for Q4' }] }))
  })
})

describe('checks', () => {
  it('offers the reasons that fit the warning', () => {
    expect(reasonsFor(issue({}))).toEqual(['decorative', 'wrong', 'other'])
    expect(reasonsFor(issue({ id: 'content.size' }))).toEqual(['size', 'wrong', 'other'])
    expect(reasonsFor(issue({ id: 'subject.spam', group: 'Subject' }))).toEqual(['subject', 'wrong', 'other'])
  })

  it('dismisses a warning with a reason', async () => {
    const posts: unknown[] = []
    unmock = mockApi({
      [`POST ${base}/dismissals`]: async (r) => {
        posts.push(await r.json())
        return json({}, 201)
      },
    })
    const retryChecks = vi.fn()
    const alt = issue({})
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ checks: checks([alt]), retryChecks })}>
        <DismissControls issue={alt} index={0} />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss…' }))
    const group = screen.getByRole('group', { name: 'Why dismiss this warning?' })
    expect(within(group).getAllByRole('radio').map((r) => r.closest('label')?.textContent)).toEqual(['The image is decorative', 'The check is wrong here', 'Other'])
    expect(group).toHaveTextContent('It stays listed in the Checks with your name and reason.')
    // "Other" needs a note.
    fireEvent.click(within(group).getByRole('radio', { name: 'Other' }))
    expect(within(group).getByRole('button', { name: 'Dismiss warning' })).toBeDisabled()
    fireEvent.click(within(group).getByRole('radio', { name: 'The image is decorative' }))
    fireEvent.click(within(group).getByRole('button', { name: 'Dismiss warning' }))
    await waitFor(() => expect(posts).toEqual([{ key: 'k0', checkId: 'content.alt_text', title: 'Hero image has no alt text', reason: 'decorative' }]))
    expect(retryChecks).toHaveBeenCalled()
  })

  it('shows who dismissed a warning, with Undo', async () => {
    let deleted = 0
    unmock = mockApi({
      [`DELETE ${base}/dismissals/d1`]: () => {
        deleted++
        return new Response(null, { status: 204 })
      },
    })
    const alt = issue({})
    const d = { index: 0, dismissal: { id: 'd1', key: 'k0', checkId: 'content.alt_text', reason: 'decorative' as const, reasonText: 'the image is decorative', note: '', by: 'Linh Nguyen', at: '2026-10-27T09:40:00Z' } }
    const retryChecks = vi.fn()
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ checks: checks([alt], [d]), retryChecks })}>
        <DismissControls issue={alt} index={0} dismissed={d} />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    expect(screen.getByText('Dismissed by Linh Nguyen · the image is decorative')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Undo' }))
    await waitFor(() => expect(deleted).toBe(1))
    expect(retryChecks).toHaveBeenCalled()
  })

  it('says why a block can’t be dismissed', async () => {
    const block = issue({ id: 'compliance.footer', severity: 'block', title: 'No unsubscribe link or postal address in the body' })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ checks: checks([block]) })}>
        <DismissControls issue={block} index={0} />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    expect(screen.getByText('Blocks can’t be dismissed: Gmail, Yahoo and CAN-SPAM require both.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Dismiss…' })).toBeNull()
  })
})

describe('style panel', () => {
  it('sets spacing, background and columns with presets, and keeps the attributes in Advanced', async () => {
    const { editor: ed } = await headlessMjmlEditor(
      '<mjml><mj-body><mj-section padding="16px 24px"><mj-column><mj-text>A</mj-text></mj-column><mj-column><mj-text>B</mj-text></mj-column></mj-section></mj-body></mjml>',
    )
    const section = ed.getWrapper()!.findType('mj-section')[0] as Component
    unmock = mockApi({ [`GET ${base}/languages`]: () => json({ source: 'en', versions: [] }) })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ editor: ed, selected: section })}>
        <StyleFields c={section} />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    const spacing = screen.getByRole('group', { name: 'Spacing' })
    expect(within(spacing).getByRole('button', { name: 'M' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByText('16 px above and below, 24 px at the sides')).toBeInTheDocument()
    fireEvent.click(within(spacing).getByRole('button', { name: 'L' }))
    expect(section.getAttributes().padding).toBe('32px 24px')
    fireEvent.click(within(screen.getByRole('group', { name: 'Background' })).getByRole('button', { name: 'Grey' }))
    expect(section.getAttributes()['background-color']).toBe('#F3F4F6')
    fireEvent.click(within(screen.getByRole('group', { name: 'Columns' })).getByRole('button', { name: '60/40' }))
    const cols = section.components().filter((x: Component) => x.get('tagName') === 'mj-column')
    expect(cols.map((x: Component) => x.getAttributes().width)).toEqual(['60%', '40%'])
    expect(screen.getByText('On phones the columns stack, the left one first.')).toBeInTheDocument()
    expect(screen.getByText('Advanced (MJML attributes)')).toBeInTheDocument()
  })
})

describe('quick review on phones', () => {
  it('offers Edit text, Inbox, Checks, Comments and Send test', async () => {
    unmock = mockApi({ [`GET ${base}/comments`]: () => json({ threads: [], open: 2, resolved: 0 }) })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ checks: checks([issue({}), issue({ id: 'content.size', title: 'Size' })]) })}>
        <QuickReviewBar />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    const bar = screen.getByRole('group', { name: 'Quick review' })
    await within(bar).findByRole('button', { name: 'Comments, 2 open' })
    expect(within(bar).getAllByRole('button').map((b) => b.getAttribute('aria-label') ?? b.textContent)).toEqual(['Edit text', 'Inbox', 'Checks, 2 warnings', 'Comments, 2 open', 'Send test'])
    fireEvent.click(within(bar).getByRole('button', { name: 'Edit text' }))
    expect(screen.getByText(/Tap any outlined text on the email to change it/)).toBeInTheDocument()
    fireEvent.click(within(bar).getByRole('button', { name: 'Checks, 2 warnings' }))
    expect(await screen.findByRole('dialog', { name: 'Checks, 2 warnings' })).toBeInTheDocument()
  })
})

describe('permission banner', () => {
  it('says who can send, not which permission', async () => {
    const writer = workspaceMe({
      permissions: ['campaigns:view', 'campaigns:edit'],
      workspace: { ...workspaceMe().workspace, roleKey: 'content-writer', roleName: 'Content Writer' },
    })
    const me = { user: { id: 'u-4', email: 'daniel@acme.io', name: 'Daniel Park', avatarColor: 'warn' }, workspaces: [] } as unknown as Me
    const campaign = testCampaign()
    campaign.access = { ...campaign.access, canSend: false, sendHelpers: ['Linh Nguyen', 'Marcus Chen'], canManageRouting: false }
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign })}>
        <PermissionBanner />
      </ComposerCtx.Provider>,
      { me: writer, seed: (qc) => qc.setQueryData(meQueryKey, me) },
    )
    const banner = await screen.findByRole('status')
    expect(banner).toHaveTextContent('Only people who can send campaigns can send it: ask Linh Nguyen or Marcus Chen.')
    expect(banner.textContent).not.toMatch(/campaigns:send|providers:manage/)
  })
})

describe('template thumbnails', () => {
  it('draw a miniature email, and read older thumbnails by their sizes', async () => {
    const { container } = await renderInWorkspace(
      <>
        <Thumb items={[{ kind: 'header', h: 10, w: 40 }, { kind: 'image', h: 30 }, { kind: 'heading', h: 8, w: 80 }, { row: [{ kind: 'image', h: 18 }, { kind: 'image', h: 18 }] }, { kind: 'button', h: 14, w: 35, accent: true }, { kind: 'footer', h: 8 }]} />
        <Thumb items={[{ h: 10, w: 40 }, { h: 34 }, { h: 8, w: 70 }, { h: 14, w: 35, accent: true }, { h: 8 }]} />
        <Thumb items={[]} blank />
      </>,
    )
    const [mini, legacy, blank] = Array.from(container.querySelectorAll('.thumb.mini'))
    expect(mini!.querySelector('.memail > .mhead .mlogo')).not.toBeNull()
    expect(mini!.querySelectorAll('.mbody > .mimg')).toHaveLength(1)
    expect(mini!.querySelectorAll('.mrow .mcol')).toHaveLength(2)
    expect(mini!.querySelectorAll('.mbody .mbt')).toHaveLength(3)
    expect(mini!.querySelector(':scope > .mfoot')).not.toBeNull()
    expect(legacy!.querySelector('.mhead')).not.toBeNull()
    expect(legacy!.querySelector('.mimg')).not.toBeNull()
    expect(legacy!.querySelector('.mbt')).not.toBeNull()
    expect(legacy!.querySelector(':scope > .mfoot')).not.toBeNull()
    expect(blank).toHaveTextContent('Empty')
  })
})
