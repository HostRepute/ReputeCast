import { translate } from '@/i18n'
import { mpi } from './entries.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mpi, {
  stripe: 'Stripe',
  paddle: 'Paddle',
  lemonsqueezy: 'Lemon Squeezy',
  byOwnStripe: 'Votre propre compte Stripe',
  byMerchant: 'Marchand officiel',
  stripeLine: 'Abonnements payants et paiements uniques, avec un portail client pour vos lecteurs.',
  merchantLine: 'Un marchand officiel\u00A0: il vend à vos lecteurs pour vous et gère la taxe sur les ventes et la TVA.',
  capSubscriptions: 'Abonnements',
  capOneTime: 'Paiements uniques',
  capPortal: 'Portail client',
  capTestMode: 'Mode test',
  capTax: 'Calcul des taxes',
  factRestricted: 'Une clé restreinte, jamais une clé secrète',
  factConnected: '{account} · {mode, select, live {Production} other {Test}}',
  factNotInRelease: 'Absent de cette version',
  notAvailable: 'Pas encore disponible',
  connect: 'Connecter',
  manage: 'Gérer',
  view: 'Voir',
  importHint: 'Vous transférez des abonnés payants depuis Substack, Ghost, beehiiv ou Kit\u202F? <a>Importez-les dans Monétisation</a>, associés à vos abonnements Stripe.',
})
