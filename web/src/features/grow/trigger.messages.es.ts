import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  kind: 'Alcanza un hito',
  anyMilestone: 'cualquier hito',
  aMilestone: 'un hito',
  program: 'Programa',
  anyProgram: 'Cualquier programa',
  milestone: 'Hito',
  anyMilestoneOption: 'Cualquier hito',
  programMilestone: '{list} · {milestone}',
  once: 'Una vez por contacto y hito',
  onceDetail: 'una vez por contacto y hito',
  explain: 'Empieza cuando un referidor alcanza el hito, tanto si el hito envía su propio correo como si no. Las variables del hito y de la recompensa funcionan en los correos de este recorrido.',
  noMilestones: 'Ningún programa de referidos tiene hitos todavía, así que nada inicia este recorrido.',
  setUp: 'Configurar referidos',
  needs: 'Solo quienes pueden editar automatizaciones pueden elegir este disparador.',
})
