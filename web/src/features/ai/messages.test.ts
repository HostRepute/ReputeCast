import { describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { m, ms } from './messages'
import vi from './messages.vi'

// X-1: the assistant's settings and composer copy say who can act (people
// who can manage integrations, everyone who can use the assistant), never
// a permission's key.
const key = /\b(?:ai|integrations|campaigns|providers|transactional):(?:use|manage|send|edit|view|publish)\b/

describe('AI copy', () => {
  it('names people, not permission keys, in English and Vietnamese', () => {
    for (const d of [...Object.values(m), ...Object.values(ms)]) {
      expect(d.message, d.id).not.toMatch(key)
    }
    for (const [id, text] of Object.entries(vi)) {
      expect(text, id).not.toMatch(key)
    }
  })

  it('says who can change the settings', () => {
    expect(t(m.featuresCaptionReadOnly)).toBe('Only people who can manage integrations can turn features on or off.')
    expect(t(m.aiUseRoles, { roles: 'Owner, Admin and Editor' })).toBe('Who can use the assistant: Owner, Admin and Editor.')
    expect(t(m.featureToastDetail)).toBe('For everyone who can use the assistant in this workspace.')
    expect(t(m.connectedDetail, { model: 'Claude Sonnet 4.5' })).toBe('Claude Sonnet 4.5 · the features switched on are ready for everyone who can use the assistant')
    expect(t(m.connectWhoAsk, { helpers: 'Linh Nguyen or Marcus Chen' })).toBe('Only people who can manage integrations can connect a model. Ask Linh Nguyen or Marcus Chen.')
  })
})
