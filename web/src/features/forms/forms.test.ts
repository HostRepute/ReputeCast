import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { luminance, onAccentFor } from './BrandingPage'
import { embedSnippet, fieldLabel, hostPath, inputOf, sameInput, type SignupForm } from './common'
import { doubleOptInNote } from './FormBuilderPage'

const form = (over: Partial<SignupForm> = {}): SignupForm =>
  ({
    id: 'f1',
    name: 'Newsletter signup',
    kind: 'embedded',
    status: 'live',
    publicId: 'frm_8c2e1a',
    slug: 'newsletter-signup',
    hostedPage: true,
    hostedUrl: 'https://news.acme.io/subscribe',
    fields: [
      { key: 'email', label: '', required: true },
      { key: 'first_name', label: '', required: false },
    ],
    listIds: ['digest', 'updates'],
    defaultListIds: ['digest'],
    listChoice: true,
    doubleOptIn: true,
    consentText: 'I agree to receive emails from Acme.',
    consentVersion: 3,
    spam: { honeypot: true, minSeconds: 3, turnstile: true, blockDisposable: true, rateLimitPerIp: 5 },
    afterSubmit: 'message',
    redirectUrl: '',
    allowedOrigins: ['acme.io', 'docs.acme.io'],
    content: {
      heading: 'The Acme newsletter',
      description: '',
      button: 'Subscribe',
      footnote: '',
      listsLabel: 'What would you like?',
      emailPlaceholder: 'you@company.com',
      namePlaceholder: 'Sam',
      popupDelay: 8,
      popupScroll: 0,
      popupRepeatDays: 14,
    },
    ...over,
  }) as SignupForm

describe('forms builder helpers', () => {
  it('shows addresses without the scheme, as the board does', () => {
    expect(hostPath('https://news.acme.io/subscribe')).toBe('news.acme.io/subscribe')
    expect(hostPath('http://localhost:5213/p/acme-newsletter/subscribe/')).toBe('localhost:5213/p/acme-newsletter/subscribe')
  })

  it('lays the embed code out on four lines like FormBuilder', () => {
    expect(embedSnippet('frm_8c2e1a', 'https://news.acme.io/p/acme-newsletter/forms/frm_8c2e1a.js')).toBe(
      '<div data-reputecast-form="frm_8c2e1a"></div>\n<script async\n  src="https://news.acme.io/p/acme-newsletter/forms/frm_8c2e1a.js">\n</script>',
    )
  })

  it('edits a copy of the saved form and notices changes', () => {
    const f = form()
    const a = inputOf(f)
    expect(a.publish).toBe(true)
    expect(sameInput(a, inputOf(f))).toBe(true)
    a.allowedOrigins.push('blog.acme.io')
    a.content.heading = 'Changed'
    expect(f.allowedOrigins).toEqual(['acme.io', 'docs.acme.io'])
    expect(f.content.heading).toBe('The Acme newsletter')
    expect(sameInput(a, inputOf(f))).toBe(false)
  })

  it("names fields: the form's label, else the built-in or custom field's", () => {
    const customs = [{ key: 'plan', label: 'Plan', type: 'select', options: ['Free', 'Pro'] }] as Schemas['SignupFormCustomField'][]
    expect(fieldLabel({ key: 'email', label: '', required: true }, customs)).toBe('Email')
    expect(fieldLabel({ key: 'first_name', label: 'Your name', required: false }, customs)).toBe('Your name')
    expect(fieldLabel({ key: 'plan', label: '', required: false }, customs)).toBe('Plan')
    expect(fieldLabel({ key: 'gone', label: '', required: false }, customs)).toBe('gone')
  })

  it('says how confirmations go out, or that they wait', () => {
    expect(doubleOptInNote(true, 'Postmark')).toBe('confirmation sent on the System stream via Postmark; contacts stay "pending" until they click')
    expect(doubleOptInNote(true, '')).toBe('confirmations wait until System mail can be sent; contacts stay "pending" until they click')
    expect(doubleOptInNote(false, 'Postmark')).toBe('off: contacts are subscribed as soon as they submit')
  })
})

describe('branding preview colours', () => {
  const saved = { accent: '#4338CA', onAccent: '#FFFFFF' } as Schemas['PublicBranding']

  it('computes WCAG relative luminance', () => {
    expect(luminance('#FFFFFF')).toBeCloseTo(1, 5)
    expect(luminance('#000000')).toBe(0)
    expect(luminance('not a colour')).toBeUndefined()
  })

  it('keeps 4.5:1 text on the accent: white on dark accents, ink on light ones', () => {
    expect(onAccentFor('#4338ca', saved)).toBe('#FFFFFF')
    expect(onAccentFor('#1D4ED8', saved)).toBe('#FFFFFF')
    expect(onAccentFor('#FDE047', saved)).toBe('#0B1220')
    expect(onAccentFor('#43zz', saved)).toBe('#FFFFFF')
  })
})
