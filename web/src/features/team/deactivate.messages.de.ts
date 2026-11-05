import { translate } from '@/i18n'
import { m } from './deactivate.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  eyebrow: 'Dialog · Mitglied deaktivieren',
  titleOne: '{name} deaktivieren?',
  titleMany: '{count, plural, one {# Mitglied} other {# Mitglieder}} deaktivieren?',
  descriptionOne: '{name} behält den eigenen Verlauf und kann später reaktiviert werden. Beim Entfernen wird die Mitgliedschaft dagegen gelöscht; im Audit-Protokoll bleibt der Name erhalten.',
  descriptionMany: 'Die Mitglieder behalten ihren Verlauf und können später reaktiviert werden. Im Audit-Protokoll bleiben ihre Namen erhalten.',
  loadingChanges: 'Auswirkungen werden geladen…',
  confirmOne: '{name} deaktivieren',
  confirmMany: '{count, plural, one {# Mitglied} other {# Mitglieder}} deaktivieren',
  doneOne: '{name} deaktiviert',
  doneMany: '{count, plural, one {# Mitglied deaktiviert} other {# Mitglieder deaktiviert}}',
  doneDetail: '{count, plural, one {von # Sitzung abgemeldet} other {von # Sitzungen abgemeldet}}',
  failedOne: '{name} konnte nicht deaktiviert werden',
  failedMany: '{count, plural, one {# Mitglied konnte} other {# Mitglieder konnten}} nicht deaktiviert werden',
  signedOutEverywhere: '{count, plural, one {Sofort überall abgemeldet (# Sitzung)} other {Sofort überall abgemeldet (# Sitzungen)}}',
  staysSignedInOne: 'Bleibt anderswo auf dieser Instanz angemeldet',
  staysSignedInMany: '{count, plural, one {# davon bleibt anderswo auf dieser Instanz angemeldet} other {# davon bleiben anderswo auf dieser Instanz angemeldet}}',
  cantSignIn: 'Kann sich nicht bei {workspace} anmelden; andere Arbeitsbereiche sind nicht betroffen',
  scheduled: '{count, plural, one {# geplante Kampagne behält ihren Zeitplan und läuft unter dem Arbeitsbereich weiter} other {# geplante Kampagnen behalten ihren Zeitplan und laufen unter dem Arbeitsbereich weiter}}',
  approvals: '{count, plural, one {# ausstehende Freigabeanfrage bleibt für freigebende Personen offen} other {# ausstehende Freigabeanfragen bleiben für freigebende Personen offen}}',
  keysLegend: 'Erstellte API-Schlüssel · {count}',
  keepKey: '{keys} aktiv lassen <k>({permissions})</k>',
  keepKeys: '{keys} aktiv lassen',
  revoke: '{count, plural, one {Jetzt widerrufen} other {Jetzt widerrufen}}',
  reason: 'Grund <k>· wird im Audit-Protokoll erfasst</k>',
  reasonPlaceholder: 'Vertrag beendet',
})
