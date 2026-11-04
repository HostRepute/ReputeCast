import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  label: 'Suggestion de série de bienvenue',
  title: 'Les nouveaux abonnés de la liste {list} ne reçoivent pas encore d’e-mail de bienvenue',
  body: 'Le modèle de départ Série de bienvenue envoie trois e-mails au cours de la première semaine.',
  covered: '{count, plural, one {La liste {lists} en a déjà une.} other {Les listes {lists} en ont déjà une.}}',
  start: 'Lancer la série de bienvenue',
  notNow: 'Plus tard',
  dismissed: 'Nous ne suggérerons plus de série de bienvenue pour la liste {list}',
  dismissedDetail: 'Créez-en une à tout moment depuis Automatisations › Modèles de départ.',
  dismissFailed: 'Impossible d’ignorer la suggestion',
})
