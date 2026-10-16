import { fireEvent, render, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { compareLanguage, diffWords, excerpt } from './compare'
import { LanguageSwitcher, type Langs } from './composer'
import { withoutVersion } from './composerParts'
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

type Version = Schemas['EmailLanguage']

const version = (patch: Partial<Version> & Pick<Version, 'language' | 'name' | 'status'>): Version => ({
  native: '',
  offered: true,
  held: false,
  attention: 0,
  changed: 0,
  blocks: 9,
  upToDate: 9,
  translated: 9,
  still: 0,
  subjectChanged: false,
  preheaderChanged: false,
  noLanguage: 0,
  notOffered: 0,
  gets: patch.language,
  translatedBy: '',
  translatedFrom: 0,
  version: 1,
  updatedBy: '',
  subject: '',
  preheader: '',
  changedBlocks: [],
  stillIndexes: [],
  changedIndexes: [],
  ...patch,
})

/** Weekly Digest #143's versions (ComposerLanguages). */
const versions: Langs = {
  source: 'en',
  sourceVersion: 9,
  canChangeSource: false,
  default: 'en',
  fallback: 'en',
  blocks: 9,
  canEdit: true,
  editHelpers: [],
  canManageWorkspace: true,
  workspaceHelpers: [],
  versions: [
    version({ language: 'en', name: 'English', status: 'source', readers: 26890 }),
    version({ language: 'es', name: 'Spanish', status: 'up_to_date', readers: 2740, translatedBy: 'Daniel Park', translatedFrom: 9 }),
    version({ language: 'de', name: 'German', status: 'outdated', attention: 2, readers: 1468, translatedBy: 'Sofia Rossi', translatedFrom: 6 }),
    version({ language: 'vi', name: 'Vietnamese', status: 'missing', gets: 'en', readers: 506 }),
  ],
}

describe('the composer’s language switcher', () => {
  const noop = () => {}

  it('counts the languages with a version on the source, as on Weekly Digest #142', () => {
    render(<LanguageSwitcher campaignName="Weekly Digest #143" data={versions} language="en" onPick={noop} onCompare={noop} onAdd={noop} />)
    const button = screen.getByRole('button', { name: 'Language version: English, Source · 3 of 4 languages have a version. Change language' })
    expect(button).toHaveTextContent('English Source · 3 of 4 languages')
  })

  it('names a version and its status, and lists every language with its readers', () => {
    const pick = vi.fn()
    render(<LanguageSwitcher campaignName="Weekly Digest #143" data={versions} language="de" onPick={pick} onCompare={noop} onAdd={noop} />)
    fireEvent.click(screen.getByRole('button', { name: 'Language version: German, outdated. Change language' }))
    const list = screen.getByRole('listbox', { name: 'Language versions of Weekly Digest #143' })
    expect(list).toHaveTextContent('GermanOutdated · 2 blocks1,468')
    expect(list).toHaveTextContent('VietnameseMissing · gets English506')
    fireEvent.click(screen.getByRole('option', { name: /Spanish/ }))
    expect(pick).toHaveBeenCalledWith('es')
  })
})

describe('the composer’s helpers', () => {
  it('keeps the words around a change in the inspector', () => {
    const before = 'Stream every workspace event to S3 or your SIEM, with 13 months of history. See the changelog'
    const after = 'Stream every workspace event to S3 or your SIEM, with 13 months of history and a search. See the changelog'
    const text = excerpt(diffWords(before, after), 4)
      .map((p) => (p.kind === 'same' ? p.text : `[${p.kind} ${p.text.trim()}]`))
      .join('')
    expect(text).toBe('…with 13 months of [del history.][add history] [add and a search.]See the changelog')
    expect(excerpt([{ kind: 'same', text: 'all the same' }])).toEqual([{ kind: 'same', text: 'all the same' }])
  })

  it('leaves the save state’s version number to the source', () => {
    expect(withoutVersion('Draft · autosaved 11:20 · v14', true)).toBe('Draft · autosaved 11:20')
    expect(withoutVersion('Draft · autosaved 11:20 · v14', false)).toBe('Draft · autosaved 11:20 · v14')
  })

  it('compares the version asked for, or the first outdated one', () => {
    expect(compareLanguage(versions, 'es')).toBe('es')
    expect(compareLanguage(versions, '')).toBe('de')
    expect(compareLanguage(versions, 'vi')).toBe('de')
  })

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
