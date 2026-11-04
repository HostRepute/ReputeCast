import { translate } from '@/i18n'
import { mcard } from './card.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mcard, {
  title: 'Monétisation · 30 derniers jours',
  label: 'Monétisation, 30 derniers jours',
  line: '{provider} · {account} · {mode, select, test {Test} other {Production}}',
  lineNoAccount: '{provider} · {mode, select, test {Test} other {Production}}',
  mrr: 'MRR',
  members: 'Abonnés payants',
  churn: 'Attrition',
  trials: 'Essais',
  nextSponsor: 'Prochain sponsor',
  nextIssue: 'n°\u00A0{issue} · {sponsor}',
  creativeDue: 'Création attendue le {date}',
  creativeReady: 'Création prête',
  open: 'Ouvrir Monétisation',
  silentTitle: 'Webhook {provider} silencieux',
  silentText: 'Aucun événement {provider} depuis {hours, plural, one {# heure} other {# heures}}, alors que {active, plural, one {# abonnement est actif} other {# abonnements sont actifs}}. Les statuts ont au plus une heure de retard.',
  checkWebhook: 'Vérifier le webhook',
  inviteLabel: 'Monétisation',
  inviteTitle: 'Monétisez votre newsletter',
  inviteText: 'Abonnements payants, cadeaux et sponsoring. Les lecteurs vous paient via votre propre compte Stripe\u202F; ReputeCast ne prend aucune commission.',
  setUp: 'Configurer Monétisation',
  notNow: 'Plus tard',
})
