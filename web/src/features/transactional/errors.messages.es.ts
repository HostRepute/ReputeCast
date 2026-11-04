import { translate } from '@/i18n'
import { me } from './errors.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(me, {
  template_not_found_when: 'El ID no existe en este espacio de trabajo',
  template_not_found_do: 'Comprueba el ID en Plantillas',
  template_not_published_when: 'La plantilla tiene un borrador pero ninguna versión publicada',
  template_not_published_do: 'Publícala primero',
  missing_variables_when: 'Falta una variable obligatoria en {data}',
  missing_variables_do: 'La respuesta indica cuáles',
  invalid_recipient_when: 'La dirección no es una dirección de correo válida',
  invalid_recipient_do: 'Valídala en tu aplicación',
  recipient_suppressed_when: 'La dirección rebotó, generó una queja o es una trampa de spam',
  recipient_suppressed_do: 'La respuesta indica el motivo y la fecha',
  sender_not_verified_when: '{from} no es un remitente transaccional verificado',
  sender_not_verified_do: 'Verifícalo en Configuración › Remitentes',
  attachments_too_large_when: 'Supera el límite de la ruta',
  attachments_too_large_do: '{size} en {provider}',
  rate_limited_when: 'Supera el límite de velocidad de la clave',
  rate_limited_do: 'Espera lo que indica {header}',
  no_healthy_route_when: 'Todos los proveedores de la ruta están caídos',
  no_healthy_route_do: 'Reintenta con la misma clave de idempotencia',
  idempotency_key_reused_when: 'La misma clave llegó con una solicitud distinta en menos de 24 horas',
  idempotency_key_reused_do: 'Usa una clave nueva para un mensaje nuevo',
  template_not_allowed_when: 'La clave no puede enviar esta plantilla o categoría',
  template_not_allowed_do: 'Usa una clave que pueda, o amplía el ámbito de la clave',
})
