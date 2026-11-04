import { translate } from '@/i18n'
import { ps } from './sales.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(ps, {
  title: 'Produktverkäufe · 30 Tage',
  products: 'Produkte',
  orders: '{count, plural, one {Bestellung} other {Bestellungen}}',
  gross: 'gesamt',
  refunded: 'erstattet · {count, plural, one {# Bestellung} other {# Bestellungen}}',
  note: '{net} nach Erstattungen aus {products, plural, one {# Produkt} other {# Produkten}}, vor den Gebühren von Stripe. ReputeCast behält nichts ein.',
})
