import { translate } from '@/i18n'
import { ps } from './sales.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(ps, {
  title: 'Ventas de productos · 30 días',
  products: 'Productos',
  orders: '{count, plural, one {pedido} other {pedidos}}',
  gross: 'bruto',
  refunded: 'reembolsado · {count, plural, one {# pedido} other {# pedidos}}',
  note: '{net} netos de {products, plural, one {# producto} other {# productos}}, antes de las comisiones de Stripe. ReputeCast no se queda nada.',
})
