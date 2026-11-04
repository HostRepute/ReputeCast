import { translate } from '@/i18n'
import { m } from './approval.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  ruleTitle: 'Publicar una plantilla transaccional también requiere aprobación',
  ruleDetail: 'Se aplica a todas las personas que pueden publicar plantillas transaccionales. Desactivado: la publicación solo se registra y la API envía la nueva versión al momento.',
  cardTitle: 'Plantilla transaccional',
  cardCaption: 'El borrador que la aprobación permite publicar a su autor',
  version: 'Versión',
  versionValue: 'v{version} de {key}',
  versionReplaces: 'v{version} de {key}, en sustitución de la v{live} publicada',
  category: 'Categoría',
  subject: 'Asunto',
  variables: 'Variables',
  content: 'Contenido',
  noContentChange: 'Sin cambios de contenido',
  breaking: 'Cambio incompatible',
  testRenders: 'Renderizados de prueba',
  renders: 'Se renderiza',
  fails: 'Falla',
  checksLine: '{passed, plural, one {# comprobación correcta} other {# comprobaciones correctas}}{warnings, plural, =0 {} one {, # aviso} other {, # avisos}}',
  published: 'Publicada como v{version}',
  decisionNote: 'La aprobación cubre exactamente este borrador. Si {name} lo edita antes de publicarlo, la publicación necesita una nueva aprobación.',
  rejectDetail: 'Se avisa a {name} y el borrador no se publica. La API sigue enviando la versión publicada. Rechaza cuando no deba salir en absoluto; para correcciones, solicita cambios.',
  changesDetail: 'La solicitud vuelve a {name} con tu comentario. El borrador sigue sin publicar hasta que vuelva a solicitarla.',
})
