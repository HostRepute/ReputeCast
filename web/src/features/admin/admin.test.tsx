import { renderToStaticMarkup } from 'react-dom/server'
import { dayPartWords, instanceAuditTime, setDisplayTimeZone } from '@/lib/format'
import { richHint, shortURL } from './sso/hints'

describe('instance audit time', () => {
  beforeAll(() => setDisplayTimeZone('UTC'))
  const now = new Date('2026-10-12T09:46:00Z')
  it('shows seconds today', () => expect(instanceAuditTime('2026-10-12T09:12:03Z', now)).toBe('09:12:03'))
  it('shows the weekday this week', () => expect(instanceAuditTime('2026-10-11T14:02:00Z', now)).toBe('Sun 11 Oct 14:02'))
  it('shows month and day before that', () => expect(instanceAuditTime('2026-09-30T17:10:00Z', now)).toBe('Sep 30 17:10'))
})

describe('day part words', () => {
  afterAll(() => setDisplayTimeZone(undefined))
  const now = new Date('2026-10-12T21:46:00Z')
  it('names the part of today in the workspace timezone', () => {
    setDisplayTimeZone('America/New_York')
    expect(dayPartWords('2026-10-12T13:12:03Z', now)).toBe('this morning')
    expect(dayPartWords('2026-10-12T19:00:00Z', now)).toBe('this afternoon')
    expect(dayPartWords('2026-10-12T23:30:00Z', now)).toBe('this evening')
  })
  it('says yesterday, then the date', () => {
    setDisplayTimeZone('America/New_York')
    expect(dayPartWords('2026-10-11T21:00:00Z', now)).toBe('yesterday')
    expect(dayPartWords('2026-10-09T13:00:00Z', now)).toBe('on Oct 9')
  })
})

describe('SSO preset hints', () => {
  it('puts the IdP menu path in bold', () => {
    const html = renderToStaticMarkup(
      <>
        {richHint(
          'In the Entra app registration open Token configuration → Add groups claim and include security groups in the ID token.',
        )}
      </>,
    )
    expect(html).toBe('In the Entra app registration open <b>Token configuration → Add groups claim</b> and include security groups in the ID token.')
  })
  it('puts the IdP menu path in bold in Vietnamese', () => {
    const html = renderToStaticMarkup(
      <>{richHint('Trong Okta, hãy mở Security → API → authorization server của bạn → Claims và thêm claim groups vào ID token, rồi yêu cầu scope groups.')}</>,
    )
    expect(html).toBe(
      'Trong Okta, hãy mở <b>Security → API → authorization server của bạn → Claims</b> và thêm claim groups vào ID token, rồi yêu cầu scope groups.',
    )
  })
  it('puts URLs in mono without the sentence dot', () => {
    const html = renderToStaticMarkup(<>{richHint('Use the tenant, for example https://login.microsoftonline.com/7c1a2b3d-…/v2.0. Find it under Overview.')}</>)
    expect(html).toBe('Use the tenant, for example <span class="mono">https://login.microsoftonline.com/7c1a2b3d-…/v2.0</span>. Find it under Overview.')
  })
  it('leaves plain hints alone', () => expect(renderToStaticMarkup(<>{richHint('Configure your IdP to send groups.')}</>)).toBe('Configure your IdP to send groups.'))
  it('shortens discovery URLs to their path', () => {
    expect(shortURL('https://login.microsoftonline.com/acme/v2.0/.well-known/openid-configuration')).toBe('…/acme/v2.0/.well-known/openid-configuration')
    expect(shortURL('not a url')).toBe('not a url')
  })
})
