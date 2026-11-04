import { translate } from '@/i18n'
import { mcard } from './card.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(mcard, {
  title: 'Monetización · últimos 30 días',
  label: 'Monetización, últimos 30 días',
  line: '{provider} · {account} · {mode, select, test {Prueba} other {Real}}',
  lineNoAccount: '{provider} · {mode, select, test {Prueba} other {Real}}',
  mrr: 'MRR',
  members: 'Suscriptores de pago',
  churn: 'Cancelaciones',
  trials: 'En prueba',
  nextSponsor: 'Próximo patrocinador',
  nextIssue: '#{issue} · {sponsor}',
  creativeDue: 'Creatividad pendiente para el {date}',
  creativeReady: 'Creatividad lista',
  open: 'Abrir Monetización',
  silentTitle: 'Webhook de {provider} sin eventos',
  silentText: 'Ningún evento de {provider} en {hours, plural, one {# hora} other {# horas}}, con {active, plural, one {# suscripción activa} other {# suscripciones activas}}. Los estados llevan como máximo una hora de retraso.',
  checkWebhook: 'Revisar el webhook',
  inviteLabel: 'Monetización',
  inviteTitle: 'Gana dinero con tu boletín',
  inviteText: 'Suscripciones de pago, regalos y patrocinios. Los lectores te pagan a través de tu propia cuenta de Stripe; ReputeCast no cobra ninguna comisión.',
  setUp: 'Configurar Monetización',
  notNow: 'Ahora no',
})
