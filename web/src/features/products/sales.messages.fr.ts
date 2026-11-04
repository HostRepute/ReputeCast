import { translate } from '@/i18n'
import { ps } from './sales.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(ps, {
  title: 'Ventes de produits · 30 jours',
  products: 'Produits',
  orders: '{count, plural, one {commande} other {commandes}}',
  gross: 'brut',
  refunded: 'remboursé · {count, plural, one {# commande} other {# commandes}}',
  note: '{net} net pour {products, plural, one {# produit} other {# produits}}, avant les frais de Stripe. ReputeCast ne prend rien.',
})
