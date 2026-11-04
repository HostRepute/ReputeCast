import { translate } from '@/i18n'
import { m } from './overview.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Fournisseurs SMS',
  sub: 'En {month}\u00A0: {sent} SMS envoyés · {delivered} délivrés',
  manage: 'Fournisseurs SMS et routage',
  budget: 'Dépenses en {month}',
  of: 'sur {budget}',
  noBudget: 'Dépenses en {month}\u00A0: {spend} · aucun budget mensuel défini',
})
