import { describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { m as approval } from './approval.messages'
import { m as step } from './automation.messages'
import { m as composer } from './composer.messages'
import { m as setup } from './messages'
import { m as waStep } from '@/features/whatsapp/automation.messages'

// The push and WhatsApp messages with ICU selects, plurals and quoted braces.
describe('push and WhatsApp messages', () => {
  it('writes the template variables with their braces', () => {
    expect(t(approval.varFieldFallback, { key: '1', field: 'first_name', fallback: 'there' })).toBe('{{1}} from first_name, or “there”')
    expect(t(approval.varText, { key: '2', text: '“8:30”' })).toBe('{{2}} “8:30”')
  })

  it('words the previews as the board does', () => {
    expect(t(composer.capWindows, { msg: 'full', n: 22, max: 40, lines: 3, hidden: '', image: 'yes', buttons: 2 })).toBe(
      'Title in full (22 of about 40 characters) · message in full · image and both buttons.',
    )
    expect(t(composer.capChromeMac, { msg: 'cut', lines: 2, hidden: 'and switch a flag off in seconds.', buttons: 2 })).toBe(
      'Message cut after 2 lines: “and switch a flag off in seconds.” is hidden · the buttons show under Options on hover · no image.',
    )
    expect(t(composer.capIos, { msg: 'full', lines: 4, hidden: '' })).toBe('Title and message in full, up to 4 lines · no image or buttons on iPhone.')
  })

  it('counts sites and subscriptions', () => {
    expect(t(setup.checkedAll, { count: 2, date: '15 Oct, 08:00' })).toBe('Both sites checked 15 Oct, 08:00')
    expect(t(setup.rotateConfirm, { count: 4264 })).toBe('Rotate keys and end 4,264 subscriptions')
  })

  it("labels the WhatsApp step's variables with their braces", () => {
    expect(t(waStep.varSourceFor, { key: '1' })).toBe('What fills {{1}}')
    expect(t(waStep.fallbackFor, { key: '2' })).toBe('Fallback for {{2}}')
    expect(t(waStep.costLine, { amount: '$0.86', n: 214 })).toBe('$0.86 for 214 messages')
  })

  it("words the push step's reach", () => {
    expect(t(step.reach, { linked: 612, contacts: 874 })).toBe('612 of the 874 contacts in this flow have a linked subscription.')
    expect(t(step.reach, { linked: 1, contacts: 1240 })).toBe('1 of the 1,240 contacts in this flow has a linked subscription.')
    expect(t(step.nodeLinked, { n: 612 })).toBe('612 linked subscriptions')
  })
})
