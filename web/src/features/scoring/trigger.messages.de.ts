import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  kind: 'Score passiert …',
  risesTo: 'Steigt auf {threshold} oder mehr',
  fallsBelow: 'Fällt unter {threshold}',
  oncePer: 'Höchstens einmal pro Kontakt in {days, plural, one {# Tag} other {# Tagen}}',
  startsWhen: 'Startet, wenn der Score',
  rises: 'Steigt auf',
  falls: 'Fällt unter',
  direction: 'Richtung',
  threshold: 'Score',
  orMore: 'oder mehr · {band} beginnt bei {from}',
  below: 'und darunter · {band} beginnt bei {from}',
  orMorePlain: 'oder mehr',
  belowPlain: 'und darunter',
  bandHot: 'Heiß',
  bandWarm: 'Warm',
  oncePerLabel: 'Höchstens einmal pro Kontakt in',
  days: 'Tagen',
  explain: 'Startet bei Änderungen durch Aktivität und Abklingen. Für Kontakte, die die Schwelle nur wegen einer geänderten Scoring-Regel passieren, wird beim Speichern der Regel nachgefragt; standardmäßig startet die Automation für sie nicht.',
  needs: 'Nur wer Automationen bearbeiten darf, kann diesen Auslöser wählen.',
  noScoring: 'Das Scoring hat noch keine Regeln, daher passiert kein Score irgendeine Schwelle.',
  setUp: 'Scoring einrichten',
  isOneOf: 'ist einer von',
  isOneOfPhrase: '{field} ist {values}',
})
