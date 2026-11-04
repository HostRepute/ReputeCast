import { translate } from '@/i18n'
import { m } from './messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  switchLabel: 'Kanal',
  menuButton: 'Kanal: {channel}. Kanal wechseln',
  email: 'E-Mail',
  sms: 'SMS',
  push: 'Push',
  whatsapp: 'WhatsApp',
  smsTermsTab: 'SMS-Bedingungen',
  all: 'Alle',
  sendsOn: 'Sendet {channels}',
  noChannelCampaigns: '{channel, select, sms {Noch keine SMS-Kampagnen} push {Noch keine Push-Kampagnen} whatsapp {Noch keine WhatsApp-Kampagnen} other {Noch keine E-Mail-Kampagnen}}',
})
