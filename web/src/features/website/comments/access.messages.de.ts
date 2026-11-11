import { translate } from '@/i18n'
import { ma } from './access.messages'

/** German for Website › Access › Comments. */
export default translate(ma, {
  title: 'Kommentare',
  text: 'Mitglieder kommentieren unter den Beiträgen, die sie lesen können. Für jede Website aus, bis du sie einschaltest.',
  waiting: '{count, plural, one {# wartet auf Prüfung} other {# warten auf Prüfung}}',
  who: 'Wer kommentieren kann',
  off: 'Aus',
  offText: 'Kein Kommentarbereich unter Beiträgen. Schon geschriebene Kommentare bleiben erhalten, bis du sie wieder einschaltest.',
  members: 'Angemeldete Mitglieder',
  membersText: 'Jedes Mitglied, das den Beitrag lesen kann',
  paid: 'Nur zahlende Mitglieder',
  paidText: 'Kostenlose Mitglieder lesen die Kommentare und sehen einen Upgrade-Link',
  review: 'Prüfung',
  reviewAll: 'Jeder Kommentar',
  reviewAllText: 'Nichts erscheint, bis jemand, der Kommentare moderiert, ihn veröffentlicht',
  reviewFirst: 'Der erste Kommentar eines Mitglieds',
  reviewFirstText: 'Nach einem veröffentlichten Kommentar erscheinen die Kommentare eines Mitglieds sofort',
  reviewNone: 'Keine',
  reviewNoneText: 'Kommentare erscheinen sofort',
  links: 'Egal, was du wählst: Ein Kommentar mit 2 oder mehr Links wartet auf Prüfung.',
  moderatedBy: 'Moderiert von',
  moderatedByNote: 'Personen, die Kommentare moderieren dürfen',
  nobody: 'Noch niemand',
  perPost: 'Pro Beitrag',
  perPostText: 'Ein Beitrag kann Kommentare in seinen Web-Einstellungen ausschalten',
  limits: 'Grenzen',
  limitsText: '{chars} Zeichen pro Kommentar; 10 pro Mitglied in 10 Minuten und 50 am Tag',
})
