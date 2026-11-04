import { translate } from '@/i18n'
import { m } from './notifuse.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  sourceWhat: 'Clé API · contacts, listes, modèles',
  templatesOption: 'Importer les modèles',
  templatesDetail: '{transactional, plural, =0 {aucun modèle transactionnel} one {# modèle transactionnel arrive en brouillon} other {# modèles transactionnels arrivent en brouillon}}, {saved, plural, =0 {aucun autre} one {# autre va dans les modèles enregistrés} other {# autres vont dans les modèles enregistrés}}',
  templatesSkipped: '{skipped, plural, one {Son e-mail de double opt-in ou de désabonnement n’est pas importé\u00A0: ReputeCast envoie les siens.} other {Ses # e-mails de double opt-in et de désabonnement ne sont pas importés\u00A0: ReputeCast envoie les siens.}}',
  doneTemplates: '{transactional, plural, one {# modèle transactionnel est arrivé} other {# modèles transactionnels sont arrivés}} en brouillon et {saved, plural, one {# modèle a été ajouté} other {# modèles ont été ajoutés}} aux modèles enregistrés.',
  doneTemplatesManual: '{manual, plural, one {# d’entre eux nécessite une correction manuelle avant de pouvoir être utilisé.} other {# d’entre eux nécessitent une correction manuelle avant de pouvoir être utilisés.}}',
  doneTemplatesLink: 'Ouvrir les modèles transactionnels',
})
