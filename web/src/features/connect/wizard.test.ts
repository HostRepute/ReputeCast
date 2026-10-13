import { numberIn } from '@/features/senders/ProvidersSettingsPage'
import type { Setup } from './api'
import { viewStep } from './SetupWizard'
import { stepLine } from './Wizard'

const ses = {
  summaries: { connection: 'us-east-1 · production', domain: 'news.acme.io · 7 of 7', domainShort: 'news.acme.io', test: 'delivered in 2.4 s', streams: 'Broadcasts, Automations, System' },
  info: { step1: 'Amazon SES', connectionHint: 'AWS access, region, provisioning', domainHint: 'DKIM, MAIL FROM, SPF, DMARC', short: 'SES' },
  domain: { name: 'news.acme.io' },
} as unknown as Setup

const lines = (current: number, setup?: Setup) => [1, 2, 3, 4, 5, 6].map((n) => stepLine(n, current, setup))

describe('setup stepper', () => {
  it('describes every step in the catalog', () => {
    expect(lines(1)).toEqual([
      'Pick from the catalog',
      'Keys, region, webhooks',
      'DKIM, return-path, SPF, DMARC',
      'Watch the event arrive',
      'Broadcasts, Automations, System',
      'Import or start empty',
    ])
  })

  it('says what each step set up, as the setup boards do', () => {
    // SetupSES, SetupDomain, SetupTestSend, SetupStreams, SetupContacts
    expect(lines(2, ses)).toEqual(['Amazon SES', 'AWS access, region, provisioning', 'DKIM, MAIL FROM, SPF, DMARC', '', '', ''])
    expect(lines(3, ses)).toEqual(['Amazon SES', 'us-east-1 · production', 'news.acme.io', '', '', ''])
    expect(lines(4, ses)).toEqual(['Amazon SES', 'us-east-1 · production', 'news.acme.io · 7 of 7', 'Watch the event arrive', '', ''])
    expect(lines(5, ses)).toEqual(['Amazon SES', 'us-east-1 · production', 'news.acme.io', 'delivered in 2.4 s', 'What SES will carry', ''])
    expect(lines(6, ses)).toEqual(['Amazon SES', '', '', '', 'Broadcasts, Automations, System', 'Import or start empty'])
  })

  it("doesn't preview the domain step for API providers (SetupError)", () => {
    const mailgun = { ...ses, info: { step1: 'Mailgun', connectionHint: 'API key, region, webhooks', domainHint: '', short: 'Mailgun' } } as unknown as Setup
    expect(lines(2, mailgun)).toEqual(['Mailgun', 'API key, region, webhooks', '', '', '', ''])
  })
})

describe('setup steps', () => {
  it('opens the step in the URL, or where the setup is', () => {
    expect(viewStep({ step: 'domain', status: 'open', providerId: 'p1' })).toBe('domain')
    expect(viewStep({ step: 'domain', status: 'open', providerId: 'p1' }, 'test')).toBe('test')
    expect(viewStep({ step: 'done', status: 'finished', providerId: 'p1' })).toBe('done')
    expect(viewStep({ step: 'done', status: 'finished', providerId: 'p1' }, 'streams')).toBe('streams')
  })

  it('needs a connected provider after the connection and a finished setup for the summary', () => {
    expect(viewStep({ step: 'connection', status: 'open', providerId: '' }, 'test')).toBe('connection')
    expect(viewStep({ step: 'connection', status: 'open', providerId: '' }, 'done')).toBe('connection')
    expect(viewStep({ step: 'contacts', status: 'open', providerId: 'p1' }, 'done')).toBe('contacts')
  })
})

describe('sending defaults', () => {
  it('reads the number in a field', () => {
    expect(numberIn('2.0 %')).toBe(2)
    expect(numberIn('0.10 %')).toBe(0.1)
    expect(numberIn('5,000 recipients · type the count to confirm')).toBe(5000)
    expect(numberIn('5 per second')).toBe(5)
    expect(numberIn('no limit')).toBe(0)
    expect(numberIn('')).toBe(0)
    expect(numberIn('soon')).toBeUndefined()
  })
})
