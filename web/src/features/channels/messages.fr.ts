import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  switchLabel: 'Canal',
  menuButton: 'Canal\u00A0: {channel}. Changer de canal',
  email: 'E-mail',
  sms: 'SMS',
  push: 'Push',
  whatsapp: 'WhatsApp',
  smsTermsTab: 'Conditions des SMS',
  all: 'Tous',
  sendsOn: 'Envoi\u00A0: {channels}',
  noChannelCampaigns: '{channel, select, sms {Aucune campagne SMS pour l’instant} push {Aucune campagne push pour l’instant} whatsapp {Aucune campagne WhatsApp pour l’instant} other {Aucune campagne e-mail pour l’instant}}',
})
