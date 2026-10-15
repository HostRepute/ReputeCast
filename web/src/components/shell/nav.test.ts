import { t } from '@/i18n'
import { moreIsCurrent, phoneNav, sectionOf, visibleNav } from './nav'
import { withSection } from './SidebarSectionsCard'

const labels = (items: { label: Parameters<typeof t>[0] }[]) => items.map((i) => t(i.label))

describe('shell navigation', () => {
  it('lists every section in the sidebar order of the boards', () => {
    expect(labels(visibleNav())).toEqual([
      'Overview',
      'Campaigns',
      'Automations',
      'Transactional',
      'Audience',
      'Grow',
      'Earn',
      'Forms & pages',
      'Deliverability',
      'Sending providers',
      'Settings',
    ])
  })

  it('keeps four sections in the phone tab bar and puts the rest in More', () => {
    const { tabs, more } = phoneNav()
    expect(labels(tabs)).toEqual(['Overview', 'Campaigns', 'Audience', 'Deliverability'])
    expect(labels(more)).toEqual(['Automations', 'Transactional', 'Grow', 'Earn', 'Forms & pages', 'Sending providers', 'Settings'])
  })

  it('leaves hidden sections out everywhere, and only those that can be hidden', () => {
    const hidden = ['transactional', 'earn', 'campaigns']
    expect(labels(visibleNav(hidden))).not.toContain('Transactional')
    expect(labels(visibleNav(hidden))).not.toContain('Earn')
    expect(labels(visibleNav(hidden))).toContain('Campaigns')
    expect(labels(visibleNav(hidden))).toContain('Grow')
    const { tabs, more } = phoneNav(hidden)
    expect(labels(tabs)).toEqual(['Overview', 'Campaigns', 'Audience', 'Deliverability'])
    expect(labels(more)).toEqual(['Automations', 'Grow', 'Forms & pages', 'Sending providers', 'Settings'])
  })

  it('finds the section of a screen, including the screens a section stands for', () => {
    expect(sectionOf('/w/acme-newsletter')).toBe('overview')
    expect(sectionOf('/w/acme-newsletter/')).toBe('overview')
    expect(sectionOf('/w/acme-newsletter/campaigns/abc/report')).toBe('campaigns')
    expect(sectionOf('/w/acme-newsletter/approvals/req-1')).toBe('campaigns')
    expect(sectionOf('/w/acme-newsletter/transactional')).toBe('transactional')
    expect(sectionOf('/w/acme-newsletter/grow')).toBe('grow')
    expect(sectionOf('/w/acme-newsletter/earn')).toBe('earn')
    expect(sectionOf('/w/acme-newsletter/settings/team/m-1')).toBe('settings')
    expect(sectionOf('/w/acme-newsletter/admin/users')).toBe('settings')
    expect(sectionOf('/w/acme-newsletter/account')).toBeUndefined()
    expect(sectionOf('/w/acme-newsletter/notifications')).toBeUndefined()
    expect(sectionOf('/sign-in')).toBeUndefined()
  })

  it('shows More as current while its sheet is open or on a screen that lives in it', () => {
    expect(moreIsCurrent('overview')).toBe(false)
    expect(moreIsCurrent('overview', [], true)).toBe(true)
    expect(moreIsCurrent('settings')).toBe(true)
    expect(moreIsCurrent('grow')).toBe(true)
    // A hidden section's screens still open by link, but nothing in the sheet stands for them.
    expect(moreIsCurrent('grow', ['grow'])).toBe(false)
    expect(moreIsCurrent(undefined)).toBe(false)
  })

  it('keeps the hidden sections in sidebar order when one changes', () => {
    expect(withSection([], 'earn', false)).toEqual(['earn'])
    expect(withSection(['earn'], 'transactional', false)).toEqual(['transactional', 'earn'])
    expect(withSection(['transactional', 'earn'], 'transactional', true)).toEqual(['earn'])
    expect(withSection(['grow'], 'grow', false)).toEqual(['grow'])
  })
})
