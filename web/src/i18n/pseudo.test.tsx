import { render, screen } from '@testing-library/react'
import { act } from 'react'
import { T, defineMessages, formatList, setLocale, setRegion, t } from '.'
import { accent, pseudoSource, pseudoWrap, PSEUDO_LOCALE } from './pseudo'

const m = defineMessages('pseudoTest', {
  schedule: 'Schedule send to {count} recipients',
  sent: '{n, plural, one {# message} other {# messages}} sent',
  home: 'Home of ReputeCast',
  sentTo: 'Sent to <b>{name}</b>',
  edit: 'Edit',
})

describe('pseudo-locale', () => {
  beforeEach(async () => {
    await act(() => setLocale(PSEUDO_LOCALE))
  })
  afterEach(async () => {
    await act(() => setLocale('en'))
    setRegion('en-US')
  })

  it("matches the brief's example: values keep their spelling", () => {
    expect(t(m.schedule, { count: '31,604' })).toBe('⟦Šçĥéðûļé šéñð ţö 31,604 ŕéçîþîéñţš ·············⟧')
  })

  it('pads by about 40%, as on the stress-test boards', () => {
    expect(t(m.edit)).toBe('⟦Éðîţ ·⟧')
    expect(t(m.home)).toBe('⟦Ĥömé öƒ ReputeCast ······⟧')
    expect(pseudoWrap('Campaigns')).toEqual(['⟦', ' ···⟧'])
    expect(pseudoWrap('')).toEqual(['', ''])
  })

  it('accents plural options but not the number', () => {
    expect(t(m.sent, { n: 48213 })).toBe('⟦48,213 méššáĝéš šéñţ ·······⟧')
  })

  it("accents a list's own words, not its items", () => {
    expect(formatList(['Linh', 'Marcus', 'Aisha'])).toBe('Linh, Marcus áñð Aisha')
    expect(formatList(['Linh', 'Marcus'], 'disjunction')).toBe('Linh öŕ Marcus')
  })

  it('keeps names and acronyms', () => {
    expect(accent('Draft · blocked A/B SES ReputeCast')).toBe('Ðŕáƒţ · bļöçķéð A/B SES ReputeCast')
    expect(pseudoSource("It''s {n, plural, one {# day} other {# days}}")).toBe("Îţ'š {n,plural,one{# ðáý} other{# ðáýš}}")
  })

  it('brackets rich messages around their markup', () => {
    render(<T msg={m.sentTo} values={{ name: 'Sam Rivera', b: (c) => <strong>{c}</strong> }} />)
    expect(document.body.textContent).toBe('⟦Šéñţ ţö Sam Rivera ······⟧')
    expect(screen.getByText('Sam Rivera').tagName).toBe('STRONG')
  })
})
