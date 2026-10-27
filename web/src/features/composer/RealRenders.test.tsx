import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerCtx } from './context'
import { ApproxNote, ConnectRenderingDialog, RealRendersCard, RenderDialog } from './RealRenders'
import { testCampaign, testComposer } from './testing'

// CO-13: the previews say they're approximations; the workspace's own
// Litmus or Mailgun Inspect account makes real renders.

const campaign = testCampaign()
const service = '/api/v1/workspaces/acme-newsletter/rendering-service'
const renders = `/api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}/renders`
const editor = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })
const admin = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'integrations:manage', 'integrations:view'] })

const litmus = {
  connected: true,
  service: 'litmus',
  keyHint: '52d3',
  status: 'connected',
  connectedBy: 'Linh Nguyen',
  connectedAt: '2026-10-20T09:00:00Z',
  clients: [
    { id: 'OL2019', name: 'Outlook 2019', default: true },
    { id: 'GMAILNEW', name: 'Gmail', default: true },
    { id: 'OL2007', name: 'Outlook 2007', default: false },
  ],
}

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

function mount(node: ReactNode, me = editor, state = {}) {
  return renderInWorkspace(<ComposerCtx.Provider value={testComposer({ campaign, ...state })}>{node}</ComposerCtx.Provider>, { me })
}

describe('the approximation note', () => {
  it('says the previews are approximations and offers Connect to people who manage integrations', async () => {
    unmock = mockApi({ [`GET ${service}`]: () => json({ connected: false, clients: [] }) })
    const onConnect = vi.fn()
    await mount(<ApproxNote onConnect={onConnect} onRender={() => {}} />, admin)
    const note = await screen.findByRole('note')
    expect(note).toHaveTextContent('Approximated in your browser from the compiled HTML. Real renders need a rendering service (Litmus or Mailgun Inspect).')
    fireEvent.click(within(note).getByRole('button', { name: 'Connect' }))
    expect(onConnect).toHaveBeenCalled()
  })

  it('says who can connect one, to everyone else', async () => {
    unmock = mockApi({ [`GET ${service}`]: () => json({ connected: false, clients: [] }) })
    await mount(<ApproxNote onConnect={() => {}} onRender={() => {}} />)
    const note = await screen.findByRole('note')
    expect(within(note).queryByRole('button')).toBeNull()
    expect(note).toHaveTextContent('People who manage integrations can connect one.')
  })

  it('names the connected service and offers real renders', async () => {
    unmock = mockApi({ [`GET ${service}`]: () => json(litmus) })
    const onRender = vi.fn()
    await mount(<ApproxNote onConnect={() => {}} onRender={onRender} />)
    await waitFor(() => expect(screen.getByRole('note')).toHaveTextContent('Real renders come from your Litmus account.'))
    fireEvent.click(screen.getByRole('button', { name: 'Render in real clients' }))
    expect(onRender).toHaveBeenCalled()
  })
})

describe('connecting a service', () => {
  it("shows the service's refusal and keeps the dialog open; a good key connects", async () => {
    const bodies: unknown[] = []
    let connected = false
    unmock = mockApi({
      [`GET ${service}`]: () => json(connected ? litmus : { connected: false, clients: [] }),
      [`PUT ${service}`]: async (r) => {
        const body = (await r.json()) as { apiKey: string }
        bodies.push(body)
        if (body.apiKey !== 'lit_live_7c1e9a4b52d3') {
          return json({ title: 'Unprocessable Entity', status: 422, detail: 'Litmus refused the API key. Check it in your Litmus account and paste it again.', errors: [{ location: 'body.apiKey', message: 'Litmus refused the API key. Check it in your Litmus account and paste it again.' }] }, 422)
        }
        connected = true
        return json(litmus)
      },
    })
    await mount(<ConnectRenderingDialog open onClose={() => {}} />, admin)
    const dialog = await screen.findByRole('dialog', { name: 'Connect a rendering service' })
    const key = within(dialog).getByLabelText('API key')
    expect(key).toHaveAttribute('type', 'password')
    fireEvent.change(key, { target: { value: 'lit_live_wrong000' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Test and connect' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent('Litmus refused the API key.')
    fireEvent.change(key, { target: { value: 'lit_live_7c1e9a4b52d3' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Test and connect' }))
    expect(await within(dialog).findByText('Litmus · key ending 52d3 · connected by Linh Nguyen, 20 Oct')).toBeInTheDocument()
    expect(within(dialog).getByText('3 email clients available')).toBeInTheDocument()
    expect(bodies[1]).toEqual({ service: 'litmus', apiKey: 'lit_live_7c1e9a4b52d3' })
  })

  it('asks for the region of a Mailgun account', async () => {
    const bodies: unknown[] = []
    unmock = mockApi({
      [`GET ${service}`]: () => json({ connected: false, clients: [] }),
      [`PUT ${service}`]: async (r) => {
        bodies.push(await r.json())
        return json({ ...litmus, service: 'mailgun_inspect', region: 'eu' })
      },
    })
    await mount(<ConnectRenderingDialog open onClose={() => {}} />, admin)
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByLabelText('Mailgun Inspect'))
    fireEvent.change(within(dialog).getByLabelText('Region'), { target: { value: 'eu' } })
    fireEvent.change(within(dialog).getByLabelText('API key'), { target: { value: 'key-3f9a1c2e8b7d' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Test and connect' }))
    await waitFor(() => expect(bodies).toEqual([{ service: 'mailgun_inspect', apiKey: 'key-3f9a1c2e8b7d', region: 'eu' }]))
  })
})

describe('rendering', () => {
  it('starts with the usual clients picked, saving unsaved edits first', async () => {
    const posted: unknown[] = []
    unmock = mockApi({
      [`GET ${service}`]: () => json(litmus),
      [`POST ${renders}`]: async (r) => {
        posted.push(await r.json())
        return json({ id: 'r1', version: 16, variant: 'A', service: 'litmus', status: 'pending', results: [], requestedBy: 'Linh Nguyen', requestedAt: '2026-10-25T09:41:00Z' }, 201)
      },
    })
    const save = vi.fn(async () => true)
    const onClose = vi.fn()
    await mount(<RenderDialog open onClose={onClose} />, editor, { save, hasUnsaved: () => true })
    const dialog = await screen.findByRole('dialog', { name: 'Render in real clients' })
    await within(dialog).findByLabelText('Outlook 2019')
    expect(dialog).toHaveTextContent('personalized for the sample contact Sam Rivera. Each client is one preview on your Litmus plan.')
    expect(within(dialog).getByLabelText('Outlook 2019')).toBeChecked()
    expect(within(dialog).getByLabelText('Outlook 2007')).not.toBeChecked()
    fireEvent.click(within(dialog).getByLabelText('Outlook 2007'))
    fireEvent.click(within(dialog).getByRole('button', { name: 'Render in 3 clients' }))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(save).toHaveBeenCalled()
    expect(posted).toEqual([{ clients: ['OL2019', 'GMAILNEW', 'OL2007'] }])
  })

  it("sends nothing when the edits can't be saved", async () => {
    const post = vi.fn(() => json({}, 201))
    unmock = mockApi({ [`GET ${service}`]: () => json(litmus), [`POST ${renders}`]: post })
    await mount(<RenderDialog open onClose={() => {}} />, editor, { save: async () => false, hasUnsaved: () => true })
    const dialog = await screen.findByRole('dialog')
    await waitFor(() => expect(within(dialog).getByLabelText('Gmail')).toBeChecked())
    fireEvent.click(within(dialog).getByRole('button', { name: 'Render in 2 clients' }))
    expect(await within(dialog).findByRole('alert')).toHaveTextContent("Your edits couldn't be saved, so nothing was sent to Litmus.")
    expect(post).not.toHaveBeenCalled()
  })

  it('shows each client as it arrives, and says when the email changed since', async () => {
    unmock = mockApi({
      [`GET ${renders}`]: () =>
        json({
          items: [
            {
              id: 'r1',
              version: 15,
              variant: 'A',
              service: 'litmus',
              status: 'pending',
              requestedBy: 'Linh Nguyen',
              requestedAt: '2026-10-25T09:41:00Z',
              results: [
                { client: 'OL2019', name: 'Outlook 2019', status: 'done', imageUrl: 'https://OL2019.capture.litmuscdn.com/full.png', thumbUrl: 'https://OL2019.capture.litmuscdn.com/thumb.png' },
                { client: 'GMAILNEW', name: 'Gmail', status: 'pending' },
                { client: 'OL2007', name: 'Outlook 2007', status: 'failed' },
              ],
            },
          ],
        }),
    })
    await mount(<RealRendersCard />, editor, { currentVersion: () => 16 })
    const card = await screen.findByRole('region', { name: 'Real renders · Litmus' })
    expect(card).toHaveTextContent('2 of 3 captured…')
    expect(card).toHaveTextContent('From v15; the email has changed since.')
    const link = within(card).getByRole('link', { name: 'Open Outlook 2019 full size' })
    expect(link).toHaveAttribute('href', 'https://OL2019.capture.litmuscdn.com/full.png')
    expect(link).toHaveAttribute('target', '_blank')
    expect(within(card).getByText('Capturing…')).toBeInTheDocument()
    expect(within(card).getByText("Couldn't capture")).toBeInTheDocument()
  })
})
