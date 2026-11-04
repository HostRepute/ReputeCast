import { translate } from '@/i18n'
import { m } from './policy.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  costTitle: 'Se estima que una campaña de SMS o WhatsApp cuesta al menos',
  costDetail: 'Según los precios de los proveedores para sus destinatarios. En los SMS, “el permiso de envío” significa poder enviar tanto campañas como campañas de SMS.',
  costAmount: 'Costo mínimo estimado, en dólares',
  costInvalid: 'Introduce un importe mayor que cero.',
})
