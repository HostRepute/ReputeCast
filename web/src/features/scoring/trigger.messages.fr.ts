import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  kind: 'Le score franchit…',
  risesTo: 'Atteint {threshold} ou plus',
  fallsBelow: 'Passe sous {threshold}',
  oncePer: 'Au plus une fois par contact sur {days, plural, one {# jour} other {# jours}}',
  startsWhen: 'Démarre quand le score',
  rises: 'Atteint',
  falls: 'Passe sous',
  direction: 'Sens',
  threshold: 'Score',
  orMore: 'ou plus · {band} commence à {from}',
  below: 'et moins · {band} commence à {from}',
  orMorePlain: 'ou plus',
  belowPlain: 'et moins',
  bandHot: 'Chaud',
  bandWarm: 'Tiède',
  oncePerLabel: 'Au plus une fois par contact sur',
  days: 'jours',
  explain: 'Démarre sur les variations dues à l’activité et à la décroissance. Pour les contacts qui franchissent le seuil uniquement parce qu’une règle de scoring a changé, la question est posée à l’enregistrement de la règle\u202F; par défaut, l’automatisation ne démarre pas.',
  needs: 'Seules les personnes autorisées à modifier les automatisations peuvent choisir ce déclencheur.',
  noScoring: 'Le scoring n’a encore aucune règle\u00A0: aucun score ne franchit donc de seuil.',
  setUp: 'Configurer le scoring',
  isOneOf: 'est l’un de',
  isOneOfPhrase: '{field} est l’un de\u00A0: {values}',
})
