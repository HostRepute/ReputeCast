import { translate } from '@/i18n'
import { m } from './overview.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'SMS-Anbieter',
  sub: '{month}: {sent} SMS gesendet · {delivered} zugestellt',
  manage: 'SMS-Anbieter & Routing',
  budget: 'Ausgaben {month}',
  of: 'von {budget}',
  noBudget: 'Ausgaben {month}: {spend} · kein Monatsbudget festgelegt',
})
