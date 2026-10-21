import { defineMessages } from '@/i18n'

/** Settings › Approvals: the rule for what texts cost (brief §3.7). */
export const m = defineMessages('smsPolicy', {
  costTitle: 'An SMS or WhatsApp campaign is estimated to cost at least',
  costDetail: 'From the providers’ prices for its recipients. For texts, “the send permission” is sms:send on top of campaigns:send.',
  costAmount: 'Minimum estimated cost, in dollars',
  costInvalid: 'Enter an amount above $0.',
})
