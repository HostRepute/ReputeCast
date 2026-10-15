import { parse } from '@formatjs/icu-messageformat-parser'
import { m as roles } from '@/features/roles/messages'
import { t } from '@/i18n'
import { m } from './messages'

describe('shell and roles messages', () => {
  it('are valid ICU messages', () => {
    for (const d of [...Object.values(m), ...Object.values(roles)]) {
      expect(() => parse(d.message), d.id).not.toThrow()
    }
  })

  it('keep their English wording when formatted', () => {
    expect(t(m.moreTabLabel, { sections: 'Automations and Settings' })).toBe('More: Automations and Settings')
    expect(t(m.switcherWorkspaceLine, { role: 'Owner', count: 48213 })).toBe('Owner · 48,213 subscribers')
    expect(t(m.notificationsMarked, { count: 1 })).toBe('1 notification marked as read')
    expect(t(m.bannerMaintenanceBody, { version: 'v1.6.0', by: 'Linh Nguyen', when: 'started 09:00' })).toBe(
      'ReputeCast v1.6.0 is being installed by Linh Nguyen (started 09:00). Reading works; edits and sends resume when it finishes.',
    )
    expect(t(m.bannerMaintenanceBody, { version: 'none', by: 'none', when: 'none' })).toBe('ReputeCast is being upgraded. Reading works; edits and sends resume when it finishes.')
    expect(t(m.bannerMaintenanceEta, { minutes: 1 })).toBe('about 1 minute')
    expect(t(m.stateErrorBodySaved, { saved: 'Your draft from 09:38 is saved.' })).toBe(
      "The page couldn't load. Your draft from 09:38 is saved. Retry now, or wait a minute; the error is already in the server log.",
    )
    expect(t(m.userMenuVersionUpdate, { name: 'Acme ReputeCast', version: 'v1.0.0', latest: 'v1.5.0' })).toBe('Acme ReputeCast v1.0.0 · v1.5.0 available')
  })
})
