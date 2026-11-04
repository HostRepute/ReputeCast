import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  kind: 'Meilenstein erreicht',
  anyMilestone: 'jeder Meilenstein',
  aMilestone: 'ein Meilenstein',
  program: 'Programm',
  anyProgram: 'Alle Programme',
  milestone: 'Meilenstein',
  anyMilestoneOption: 'Alle Meilensteine',
  programMilestone: '{list} · {milestone}',
  once: 'Einmal pro Kontakt und Meilenstein',
  onceDetail: 'einmal pro Kontakt und Meilenstein',
  explain: 'Startet, wenn eine empfehlende Person den Meilenstein erreicht, ob der Meilenstein eine eigene E-Mail sendet oder nicht. Die Platzhalter für Meilenstein und Prämie funktionieren in den E-Mails dieses Ablaufs.',
  noMilestones: 'Noch kein Empfehlungsprogramm hat Meilensteine, daher startet nichts diesen Ablauf.',
  setUp: 'Weiterempfehlungen einrichten',
  needs: 'Nur Personen, die Automationen bearbeiten dürfen, können diesen Auslöser wählen.',
})
