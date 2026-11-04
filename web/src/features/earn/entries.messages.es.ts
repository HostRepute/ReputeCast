import { translate } from '@/i18n'
import { mpi } from './entries.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(mpi, {
  stripe: 'Stripe',
  paddle: 'Paddle',
  lemonsqueezy: 'Lemon Squeezy',
  byOwnStripe: 'Tu propia cuenta de Stripe',
  byMerchant: 'Comerciante registrado',
  stripeLine: 'Suscripciones de pago y pagos únicos, con un portal del cliente para tus lectores.',
  merchantLine: 'Un comerciante registrado (merchant of record): vende a tus lectores en tu nombre y gestiona el impuesto sobre las ventas y el IVA.',
  capSubscriptions: 'Suscripciones',
  capOneTime: 'Pagos únicos',
  capPortal: 'Portal del cliente',
  capTestMode: 'Modo de prueba',
  capTax: 'Cálculo de impuestos',
  factRestricted: 'Una clave restringida, nunca una clave secreta',
  factConnected: '{account} · {mode, select, live {Real} other {Prueba}}',
  factNotInRelease: 'No incluido en esta versión',
  notAvailable: 'Aún no disponible',
  connect: 'Conectar',
  manage: 'Gestionar',
  view: 'Ver',
  importHint: '¿Traes suscriptores de pago de Substack, Ghost, beehiiv o Kit? <a>Impórtalos en Monetización</a>, vinculados con tus suscripciones de Stripe.',
})
