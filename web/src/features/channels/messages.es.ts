import { translate } from '@/i18n'
import { m } from './messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  switchLabel: 'Canal',
  menuButton: 'Canal: {channel}. Cambiar de canal',
  email: 'Correo',
  sms: 'SMS',
  push: 'Push',
  whatsapp: 'WhatsApp',
  smsTermsTab: 'Condiciones de los SMS',
  all: 'Todos',
  sendsOn: 'Envía por {channels}',
  noChannelCampaigns: '{channel, select, sms {Aún no hay campañas de SMS} push {Aún no hay campañas push} whatsapp {Aún no hay campañas de WhatsApp} other {Aún no hay campañas de correo}}',
})
