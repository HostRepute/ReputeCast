import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { deliveryLanguage, languageSource } from './contacts'

type ContactLanguage = Schemas['ContactLanguage']
type Activity = Schemas['ContactActivityItem']

const lang = (patch: Partial<ContactLanguage>): ContactLanguage => ({
  code: 'en',
  name: 'English',
  offered: true,
  gets: 'en',
  source: 'import',
  sourceName: 'Brevo',
  browser: false,
  setAt: '2026-09-18T12:00:00Z',
  options: [],
  ...patch,
})

describe('where a contact’s language came from', () => {
  it('names each source in a sentence of its own', () => {
    expect(languageSource(lang({}), t)).toBe('Brevo import, 18 Sep 2026')
    expect(languageSource(lang({ code: 'es-MX', source: 'form', sourceName: 'Newsletter signup', browser: true, setAt: '2026-10-03T12:00:00Z' }), t)).toBe(
      'Newsletter signup form, from the browser, 3 Oct 2026',
    )
    expect(languageSource(lang({ source: 'form', sourceName: '', browser: true, setAt: '2026-10-03T12:00:00Z' }), t)).toBe('Subscribe page, from the browser, 3 Oct 2026')
    expect(languageSource(lang({ source: 'preference_center', sourceName: '', setAt: '2026-10-09T12:00:00Z' }), t)).toBe('Preference center, 9 Oct 2026')
    expect(languageSource(lang({ code: 'pt-BR', source: 'api', sourceName: '', setAt: '2026-10-12T12:00:00Z' }), t)).toBe('Set by the API, 12 Oct 2026')
    expect(languageSource(lang({ source: 'api', sourceName: 'Zapier', setAt: '2026-10-12T12:00:00Z' }), t)).toBe('Set by Zapier, 12 Oct 2026')
    expect(languageSource(lang({ source: 'team', sourceName: 'Linh Nguyen', setAt: '2026-10-15T12:00:00Z' }), t)).toBe('Changed by Linh Nguyen, 15 Oct 2026')
  })
  it('says nothing without a language or a source', () => {
    expect(languageSource(lang({ code: '' }), t)).toBe('')
    expect(languageSource(lang({ source: '' }), t)).toBe('')
  })
})

describe('the language a delivery sent', () => {
  const item = (patch: Partial<Activity>): Activity =>
    ({ id: '1', category: 'delivery', type: 'delivered', at: '2026-10-14T12:00:00Z', subject: 'Weekly Digest #142', failover: false, machine: false, ...patch }) as Activity
  it('names the version, and the fallback with why', () => {
    expect(deliveryLanguage(item({ language: 'en' }), t, 'en', 'Acme')).toEqual({ label: 'English', fallback: false })
    expect(deliveryLanguage(item({ language: 'en', languageReason: 'not_offered', contactLanguageName: 'Portuguese' }), t, 'en', 'Acme')).toEqual({
      label: 'English, the fallback',
      fallback: true,
      why: 'Portuguese isn’t one of Acme’s languages',
    })
    expect(deliveryLanguage(item({}), t, 'en', 'Acme')).toEqual({ fallback: false })
  })
})
