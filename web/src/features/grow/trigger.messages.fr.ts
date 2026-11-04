import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  kind: 'Atteint un palier',
  anyMilestone: 'n’importe quel palier',
  aMilestone: 'un palier',
  program: 'Programme',
  anyProgram: 'Tous les programmes',
  milestone: 'Palier',
  anyMilestoneOption: 'N’importe quel palier',
  programMilestone: '{list} · {milestone}',
  once: 'Une fois par contact et par palier',
  onceDetail: 'une fois par contact et par palier',
  explain: 'Démarre quand un parrain atteint le palier, que le palier envoie ou non son propre e-mail. Les variables du palier et de la récompense fonctionnent dans les e-mails de ce scénario.',
  noMilestones: 'Aucun programme de parrainage n’a encore de paliers\u00A0: rien ne démarre donc ce scénario.',
  setUp: 'Configurer le parrainage',
  needs: 'Seules les personnes autorisées à modifier les automatisations peuvent choisir ce déclencheur.',
})
