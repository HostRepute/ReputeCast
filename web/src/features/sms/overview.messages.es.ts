import { translate } from '@/i18n'
import { m } from './overview.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Proveedores de SMS',
  sub: '{month}: {sent} SMS enviados · {delivered} entregados',
  manage: 'Proveedores y enrutamiento de SMS',
  budget: 'Gasto de {month}',
  of: 'de {budget}',
  noBudget: 'Gasto de {month}: {spend} · sin presupuesto mensual',
})
