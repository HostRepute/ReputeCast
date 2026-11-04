import { translate } from '@/i18n'
import { m } from './settings.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Proveedores de SMS',
  senders: '{count, plural, one {# remitente} other {# remitentes}} · {kinds}',
  noSenders: 'Aún no hay remitentes',
  routes: 'Rutas',
  notRouted: 'Aún sin rutas',
  held: 'Retenido: {countries}',
  sendersLabel: 'Remitentes',
  connected: 'Conectado',
  connectedOn: '{date} · {name}',
  cost: 'Costo · {month}',
  budget: 'Presupuesto de SMS · {month}',
  spentOf: '<b>{spent}</b> de {budget}',
  alerts: 'Avisos al {alert} % y al 100 %. Al 100 %, las campañas no pueden empezar y los pasos de SMS esperan hasta que alguien que pueda gestionar proveedores lo aumente.',
  noBudget: 'Sin presupuesto mensual: {spend} hasta ahora en {month}.',
  foot: 'Los remitentes, las rutas por país, las horas de silencio, las reglas y el presupuesto están en Proveedores de envío › SMS.',
  changeBudget: 'Cambiar el presupuesto',
  open: 'Abrir proveedores de SMS',
})
