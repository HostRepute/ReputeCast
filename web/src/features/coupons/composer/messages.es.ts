import { translate } from '@/i18n'
import { mc } from './messages'

/** Spanish for the composer's Discount code block (M59). */
export default translate(mc, {
  block: 'Código de descuento',
  name: 'Código de descuento',
  defaultLabel: 'Tu código',
  defaultButton: 'Comprar ahora',
  canvasPick: 'Elige un grupo de códigos de descuento en los ajustes del bloque.',
  canvasUnknown: 'No hay ningún grupo de códigos de descuento llamado {slug}. Elige otro en los ajustes del bloque.',
  oneCode: 'Un código por lector, asignado cuando se envía el email.',
  pool: 'Grupo',
  poolOption: '{name} · {free} libres',
  poolNone: 'Elige un grupo',
  poolStore: '{platform} · {store}',
  poolUpload: 'Códigos subidos',
  poolOnePer: 'un código por contacto',
  poolLiquid: 'en Liquid <mono>{token}</mono>',
  noPools: 'Aún no hay grupos. <a>Crea uno</a> en Campañas › Códigos de descuento.',
  label: 'Etiqueta',
  button: 'Botón',
  buttonOn: 'Mostrar el botón',
  buttonOpens: 'Abre {store} con el código aplicado: <mono>{token}</mono>',
  buttonApply: 'Abre la dirección para aplicarlo con el código: <mono>{token}</mono>',
  buttonNoUrl: '{name} no tiene dirección para aplicarlo, así que el bloque no muestra botón. Añade una en la página del grupo.',
  manage: 'Gestionar los códigos de descuento',
  manageNote: 'Los grupos, sus cifras y dónde se usa cada uno, en Campañas › Códigos de descuento.',
  pass: 'Correcto',
  passNote: 'Revisar y enviar comprueba que haya un código para cada destinatario.',
  warn: 'Aviso',
  archivedNote: '{name} está archivado: los lectores no reciben código. Elige otro grupo.',
  tokensGroup: 'Códigos de descuento',
})
