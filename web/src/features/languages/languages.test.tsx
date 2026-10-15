import { render, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { diffWords } from './compare'
import { brandOf } from './LanguagesCard'
import { sentByLanguage } from './report'
import { ByVersion, confirmLanguages, languageRows, sentVersions, testLanguages } from './review'
import { changedLink } from './sourceEdits'

type Check = Schemas['ReadinessCheck']

/** Weekly Digest #143's "Recipients per language" (SendReadinessAccessibility). */
const checks: Check[] = [
  { key: 'content.size', group: 'content', status: 'pass', title: 'HTML size', flatTitle: 'Content: HTML size', fixes: [] },
  {
    key: 'languages.recipients',
    group: 'languages',
    status: 'pass',
    title: 'Recipients per language',
    flatTitle: 'Languages: Recipients per language',
    fixes: [],
    rows: [
      { label: 'English', count: 26890, detail: 'gets English · 23,269 English, 3,613 with no language, 8 with other languages', language: 'en', gets: 'en', getsName: 'English' },
      { label: 'Spanish', count: 2740, detail: 'gets Spanish', language: 'es', gets: 'es', getsName: 'Spanish' },
      { label: 'German', count: 1468, detail: 'gets German, the outdated version', language: 'de', gets: 'de', getsName: 'German' },
      { label: 'Vietnamese', count: 506, detail: 'gets English: there’s no Vietnamese version', language: 'vi', gets: 'en', getsName: 'English' },
    ],
  },
]

describe('languages in Review & send', () => {
  it('reads the versions a send uses from the recipient rows', () => {
    const rows = languageRows(checks)
    expect(rows).toHaveLength(4)
    expect(sentVersions(rows).map((v) => v.name)).toEqual(['English', 'Spanish', 'German'])
    expect(languageRows([])).toEqual([])
  })

  it('words the confirmation step’s languages', () => {
    expect(confirmLanguages(checks, t)).toBe('English, Spanish and German; Vietnamese readers get English')
    expect(confirmLanguages([checks[0]!], t)).toBeUndefined()
  })

  it('names a test send’s languages only for an email in several', () => {
    const versions = sentVersions(languageRows(checks))
    expect(testLanguages(['en', 'es', 'de'], versions)).toBe('English, Spanish, German')
    expect(testLanguages(['en'], versions.slice(0, 1))).toBe('')
  })

  it('draws Audience › By version, with the readers who get another version', () => {
    render(<ByVersion checks={checks} />)
    expect(screen.getByText('By version')).toBeInTheDocument()
    expect(screen.getByRole('img')).toHaveAccessibleName('English · 26,890, Spanish · 2,740, German · 1,468 and Vietnamese readers get English · 506')
    expect(screen.getByText('Vietnamese readers get English · 506')).toBeInTheDocument()
  })

  it('draws nothing for an email in one language', () => {
    const { container } = render(<ByVersion checks={[checks[0]!]} />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('the composer’s helpers', () => {
  it('shows what changed in a source block since translation', () => {
    expect(diffWords('you can now schedule flag changes ahead of time.', 'you can now schedule flag changes ahead of time, in any timezone.')).toEqual([
      { kind: 'same', text: 'you can now schedule flag changes ahead of ' },
      { kind: 'del', text: 'time.' },
      { kind: 'add', text: 'time, in any timezone.' },
    ])
    expect(diffWords('same', 'same')).toEqual([{ kind: 'same', text: 'same' }])
  })

  it('finds the one link or image a source edit changed', () => {
    const before = '<mj-button href="https://acme.io/start">Go</mj-button><mj-image src="https://cdn/a.png"></mj-image>'
    expect(changedLink(before, before.replace('acme.io/start', 'docs.acme.io/rollouts'))).toEqual({ from: 'https://acme.io/start', to: 'https://docs.acme.io/rollouts', image: false })
    expect(changedLink(before, before.replace('a.png', 'b.png'))).toEqual({ from: 'https://cdn/a.png', to: 'https://cdn/b.png', image: true })
    expect(changedLink(before, before.replace('acme.io/start', 'x').replace('a.png', 'b.png'))).toBeUndefined()
    expect(changedLink(before, before)).toBeUndefined()
  })
})

describe('languages elsewhere', () => {
  it('names the brand the way the boards do', () => {
    expect(brandOf('Acme Newsletter')).toBe('Acme')
    expect(brandOf('The Daily')).toBe('The Daily')
    expect(brandOf('Newsletter')).toBe('Newsletter')
  })

  it('splits the Sent tile by version', () => {
    expect(sentByLanguage(undefined)).toBeUndefined()
    expect(
      sentByLanguage({
        languages: 3,
        versions: 4,
        rows: [],
        clickMaps: [],
        sent: [
          { language: 'en', name: 'English', count: 27396 },
          { language: 'es', name: 'Spanish', count: 2740 },
          { language: 'de', name: 'German', count: 1468 },
        ],
      }),
    ).toBe('English 27,396 · Spanish 2,740 · German 1,468')
  })
})
