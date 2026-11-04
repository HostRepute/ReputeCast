import { translate } from '@/i18n'
import { m } from './messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Audiencia',
  sections: 'Secciones de Audiencia',
  tabContacts: 'Contactos',
  tabContactsCount: 'Contactos · {count}',
  tabSegments: 'Segmentos',
  tabSegmentsCount: 'Segmentos · {count}',
  tabHealth: 'Salud de la lista',
  tabSuppressions: 'Lista de supresión',
  tabSuppressionsCount: 'Lista de supresión · {count}',
  tierActive: 'Activo',
  tierCooling: 'Enfriándose',
  tierAtRisk: 'En riesgo',
  tierSunset: 'En retirada',
  windowActive: 'abrió en los últimos 30 días',
  windowCooling: 'de 31 a 90 días',
  windowAtRisk: 'de 91 a 180 días',
  windowSunset: 'sin aperturas en 180 días',
  consentDouble: 'Doble confirmación',
  consentSingle: 'Suscripción simple',
  consentPending: 'Importado · pendiente',
  consentPendingFilter: 'Importado, pendiente',
  mailboxOther: 'Otro',
})
