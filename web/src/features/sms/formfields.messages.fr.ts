import { translate } from '@/i18n'
import { m } from './formfields.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  phone: 'Numéro de mobile',
  phoneOptional: 'Numéro de mobile · facultatif',
  phoneRequired: 'Numéro de mobile · obligatoire',
  textsTitle: 'SMS',
  smsConsent: 'Case de consentement SMS · décochée par défaut',
  needsPhone: 'Nécessite le champ Numéro de mobile.',
  consentText: 'Texte de consentement SMS',
  evidence: 'Le numéro, les listes, ce texte exact et sa version, la page, l’adresse IP et l’heure sont conservés comme preuve de consentement SMS.',
  newVersion: 'L’enregistrement crée la version {version}\u202F; les inscriptions précédentes conservent le texte accepté à l’époque.',
  separate: 'Le consentement e-mail n’inscrit jamais personne aux SMS\u00A0: cette case est indépendante.',
  confirmation: 'Une fois la case cochée, la personne reçoit le SMS de confirmation d’inscription défini dans Routage et règles.',
  termsWords: 'conditions des SMS',
})
