import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { PushSetupPage } from '@/features/push/SetupPage'
import { SmsTabs } from '@/features/sms/common'
import { WhatsAppFrame } from '@/features/whatsapp/AccountPage'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'

/** Email and SMS set up; push and WhatsApp not yet. */
const channels = (smsSetUp = true): Schemas['SmsChannels'] => ({
  switchShown: smsSetUp,
  channels: [
    { channel: 'email', available: true, setUp: true },
    { channel: 'sms', available: true, setUp: smsSetUp },
    { channel: 'push', available: true, setUp: false },
    { channel: 'whatsapp', available: true, setUp: false },
  ],
})

const api = (smsSetUp = true) =>
  mockApi({
    'GET /api/v1/workspaces/acme-newsletter/channels': () => json(channels(smsSetUp)),
    'GET /api/v1/workspaces/acme-newsletter/push/setup': () => json({ title: 'Internal Server Error', status: 500, detail: 'The database is restarting.' }, 500),
  })

// Review R3.U5: one channel button with a listbox heads every channel's
// section of Sending providers, so each header has a single row of tabs.
describe('the channel menu', () => {
  afterEach(() => mockApi({}))

  it.each([
    ['SMS', <SmsTabs key="sms" />, '/w/acme-newsletter/providers/sms', 'SMS sections'],
    ['Push', <PushSetupPage key="push" />, '/w/acme-newsletter/providers/push', 'Push sections'],
    ['WhatsApp', (
      <WhatsAppFrame key="whatsapp" doc="WhatsApp">
        {null}
      </WhatsAppFrame>
    ), '/w/acme-newsletter/providers/whatsapp', 'WhatsApp sections'],
  ])('heads the %s section instead of a row of channel tabs', async (name, ui, path, sections) => {
    api()
    await renderInWorkspace(ui, { path })
    expect(await screen.findByRole('button', { name: `Channel: ${name}. Change channel` })).toHaveAttribute('aria-expanded', 'false')
    // One row of tabs: the section's own, and no channel tabs above them.
    expect(screen.getByRole('navigation', { name: sections })).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Channel' })).not.toBeInTheDocument()
  })

  it('lists every channel with the current one selected, and Escape closes it', async () => {
    api()
    await renderInWorkspace(<SmsTabs />, { path: '/w/acme-newsletter/providers/sms' })
    const button = await screen.findByRole('button', { name: 'Channel: SMS. Change channel' })
    fireEvent.click(button)
    const list = screen.getByRole('listbox', { name: 'Channel' })
    expect(list).not.toHaveClass('ch-menu-end')
    const options = within(list).getAllByRole('option')
    expect(options.map((o) => o.textContent)).toEqual(['Email', 'SMS', 'Push', 'WhatsApp'])
    expect(within(list).getByRole('option', { name: 'SMS' })).toHaveAttribute('aria-selected', 'true')
    expect(within(list).getByRole('option', { name: 'Email' })).toHaveAttribute('href', '/w/acme-newsletter/providers')
    expect(within(list).getByRole('option', { name: 'Push' })).toHaveAttribute('href', '/w/acme-newsletter/providers/push')
    await waitFor(() => expect(within(list).getByRole('option', { name: 'SMS' })).toHaveFocus())
    fireEvent.keyDown(list, { key: 'ArrowDown' })
    expect(within(list).getByRole('option', { name: 'Push' })).toHaveFocus()
    fireEvent.keyDown(list, { key: 'Escape' })
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(button).toHaveFocus()
  })

  it('opens leftward from the button when the right has no room, as on a phone', async () => {
    api()
    await renderInWorkspace(<SmsTabs />, { path: '/w/acme-newsletter/providers/sms' })
    const button = await screen.findByRole('button', { name: 'Channel: SMS. Change channel' })
    // A 390 px phone: the button sits after the title, 228 px from the left.
    const width = window.innerWidth
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    button.getBoundingClientRect = () => ({ left: 228, right: 315, top: 12, bottom: 56, width: 87, height: 44, x: 228, y: 12 }) as DOMRect
    try {
      fireEvent.click(button)
      expect(screen.getByRole('listbox', { name: 'Channel' })).toHaveClass('ch-menu-end')
    } finally {
      Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
    }
  })

  it('leads to a channel’s setup until it is set up', async () => {
    api(false)
    await renderInWorkspace(<WhatsAppFrame doc="WhatsApp">{null}</WhatsAppFrame>, { path: '/w/acme-newsletter/providers/whatsapp' })
    fireEvent.click(await screen.findByRole('button', { name: 'Channel: WhatsApp. Change channel' }))
    await waitFor(() => expect(screen.getByRole('option', { name: 'SMS' })).toHaveAttribute('href', '/w/acme-newsletter/providers/connect/sms'))
  })
})
