import { APP_LANGUAGES, browserLanguage, defaultRegionFor, resolvePreferences, resolveVisitor } from './languages'

describe('app languages', () => {
  it('list English first, then the others by their own names', () => {
    expect(APP_LANGUAGES.map((l) => l.code)).toEqual(['en', 'de', 'es', 'fr', 'vi'])
  })

  it('follow their own country’s formats by default', () => {
    expect(defaultRegionFor('en')).toBe('standard')
    expect(defaultRegionFor('de')).toBe('de-DE')
    expect(defaultRegionFor('es')).toBe('es-ES')
    expect(defaultRegionFor('fr')).toBe('fr-FR')
    expect(defaultRegionFor('vi')).toBe('vi-VN')
  })

  it('match the browser by base language', () => {
    expect(browserLanguage(APP_LANGUAGES, ['de-AT', 'en'])).toBe('de')
    expect(browserLanguage(APP_LANGUAGES, ['es-MX'])).toBe('es')
    expect(browserLanguage(APP_LANGUAGES, ['fr-CA', 'fr'])).toBe('fr')
    expect(browserLanguage(APP_LANGUAGES, ['it-IT', 'vi'])).toBe('vi')
    expect(browserLanguage(APP_LANGUAGES, ['it-IT'])).toBeUndefined()
  })

  it('resolve a member’s and a visitor’s formats from the language', () => {
    expect(resolvePreferences({ language: 'fr', regionFormat: 'auto', hourCycle: 'auto', weekStart: 'auto' }, undefined)).toEqual({
      locale: 'fr',
      regionFormat: 'fr-FR',
      hourCycle: 'h23',
      weekStart: 'monday',
    })
    expect(resolveVisitor({ available: APP_LANGUAGES, browser: ['es-AR'] }).regionFormat).toBe('es-ES')
  })
})
