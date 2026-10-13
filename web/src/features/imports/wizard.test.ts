import { setDisplayTimeZone, suppressedWhen } from '@/lib/format'
import { currentStep, sourceHint, type Detail } from './ImportFrame'

describe('import wizard', () => {
  it('opens an import on the step it is at', () => {
    expect(currentStep({ status: 'draft', step: 'source' })).toBe('source')
    expect(currentStep({ status: 'staging', step: 'fields' })).toBe('fields')
    expect(currentStep({ status: 'paused', step: 'fields' })).toBe('fields')
    expect(currentStep({ status: 'draft', step: 'preview' })).toBe('preview')
    expect(currentStep({ status: 'running', step: 'import' })).toBe('progress')
    expect(currentStep({ status: 'paused', step: 'import' })).toBe('progress')
    expect(currentStep({ status: 'done', step: 'done' })).toBe('report')
    expect(currentStep({ status: 'cancelled', step: 'import' })).toBe('report')
    expect(currentStep({ status: 'failed', step: 'import' })).toBe('report')
  })

  it('names the source and the chosen lists in the rail', () => {
    const d = {
      source: 'mailchimp',
      sourceName: 'Mailchimp',
      fileName: '',
      options: { lists: [{ sourceId: '3f9a2c', target: 'x' }] },
      connection: { lists: [{ id: '3f9a2c', name: 'Acme Customers' }, { id: 'other', name: 'Old list' }] },
    } as unknown as Detail
    expect(sourceHint(d)).toBe('Mailchimp · Acme Customers')
    expect(sourceHint({ ...d, source: 'csv', fileName: 'contacts-export.csv' } as Detail)).toBe('contacts-export.csv')
  })
})

describe('suppression times', () => {
  beforeAll(() => setDisplayTimeZone('UTC'))
  const now = new Date('2026-10-12T10:00:00Z')
  it('today shows the time', () => expect(suppressedWhen('2026-10-12T09:41:00Z', now)).toBe('Today 09:41'))
  it('this week shows the weekday', () => expect(suppressedWhen('2026-10-07T18:00:00Z', now)).toBe('Wed 7 Oct'))
  it('older shows the date', () => expect(suppressedWhen('2026-09-18T12:00:00Z', now)).toBe('18 Sep'))
  it('other years show the year', () => expect(suppressedWhen('2025-12-30T12:00:00Z', now)).toBe('30 Dec 2025'))
})
