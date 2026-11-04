import { translate } from '@/i18n'
import { m } from './importconsent.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Consentement SMS',
  withEvidence: 'Importé avec sa preuve',
  withEvidenceText: 'Les lignes qui indiquent oui, avec une date, une source et le texte de consentement, passent à Abonné aux SMS sur la liste cible, consignées comme importées avec cette preuve. Personne ne reçoit de SMS.',
  missing: 'Le consentement SMS a besoin de sa preuve\u00A0: associez {count, plural, one {la colonne} other {les colonnes}} {missing} dans Associer les champs. D’ici là, les numéros sont importés et le consentement SMS reste à Non abonné.',
  numbersOnly: 'Les numéros sont importés sans consentement SMS\u00A0: ils restent à Non abonné pour les SMS. Pour importer aussi le consentement SMS, associez une colonne de consentement avec sa date, sa source et son texte dans Associer les champs.',
  date: 'date',
  source: 'source',
  text: 'texte',
  mapFields: 'Associer les champs',
})
