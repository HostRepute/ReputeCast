import { translate } from '@/i18n'
import { mf } from './followup.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(mf, {
  menu: 'Seguimiento',
  nonOpeners: 'Reenviar a quienes no lo abrieron…',
  clickers: 'Enviar a quienes hicieron clic…',
  titleNonOpeners: 'Reenviar a quienes no lo abrieron',
  titleClickers: 'Enviar a quienes hicieron clic',
  bodyNonOpeners: '{count, plural, one {<b># contacto</b> recibió “{name}” y no lo ha abierto.} other {<b># contactos</b> recibieron “{name}” y no lo han abierto.}}',
  machineOnly: '{count, plural, one {# lo abrió solo a través de un proxy de privacidad, como Mail Privacy Protection de Apple: cuenta como desconocido y se excluye.} other {# lo abrieron solo a través de un proxy de privacidad, como Mail Privacy Protection de Apple: cuentan como desconocidos y se excluyen.}}',
  bodyClickers: '{count, plural, one {<b># persona</b> hizo clic en un enlace de “{name}”.} other {<b># personas</b> hicieron clic en un enlace de “{name}”.}} Los clics de los escáneres de enlaces no cuentan.',
  nobody: 'Todavía no hay nadie en este grupo, así que por ahora el seguimiento no llegaría a nadie. Su audiencia se mantiene actualizada hasta que se envíe.',
  who: 'Enviarlo a',
  whoNonOpeners: 'Quienes no lo abrieron',
  whoClickers: 'Quienes hicieron clic',
  subject: 'Nuevo asunto',
  subjectNonOpeners: 'Por si te lo perdiste: {subject}',
  subjectClickers: 'Más sobre: {subject}',
  send: 'Envío',
  after: '{days, plural, one {# día después del original ({when})} other {# días después del original ({when})}}',
  later: 'Elegir en Revisar y enviar',
  note: 'Crea un borrador con el mismo contenido. Las supresiones, las bajas y el límite de marketing se aplican como siempre.',
  create: 'Crear el seguimiento',
  creating: 'Creando…',
  created: 'Seguimiento “{name}” creado',
  loading: 'Contando a quién llegaría…',
  subjectRequired: 'Ponle un asunto al seguimiento.',
})
