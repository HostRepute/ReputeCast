import { translate } from '@/i18n'
import { m } from './approval.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  badge: 'Seite',
  badgeLabel: 'Seite: „{name}“ im Seiteneditor öffnen',
  line: 'Seite · {path} · {visibility}',
  whenApproved: 'Nach der Freigabe',
  scheduled: '{date}, {time}',
  published: 'Veröffentlicht',
  cardTitle: 'Seite',
  cardCaption: 'Was die Freigabe veröffentlicht: Inhalt, Adresse, Sichtbarkeit und Veröffentlichungszeit',
  page: 'Seite',
  pageValue: '{name} · {template}',
  address: 'Adresse',
  visibility: 'Sichtbarkeit',
  goesLive: 'Geht live',
  goesLiveNow: 'Sobald sie freigegeben ist',
  goesLivePublished: 'Nach der Freigabe veröffentlicht',
  version: 'Version',
  versionValue: 'v{version}',
  previewTitle: 'Vorschau von „{name}“, v{version}',
  previewFullTitle: 'Vollständige Vorschau von „{name}“',
  previewCaption: 'Die Seite, wie Besucher sie sehen, im Rahmen der Website',
  previewUnavailable: 'Die Vorschau ist nicht verfügbar: {error}',
  previewLoading: 'Die Vorschau wird geladen.',
  openPreview: 'Vollständige Vorschau öffnen',
  openBuilder: 'Im Seiteneditor öffnen',
  decisionNote: 'Die Freigabe gilt für genau diese Version: Inhalt, Adresse, Sichtbarkeit und Veröffentlichungszeit. {when, select, at {Die Seite geht zu ihrer Veröffentlichungszeit live} other {Die Seite geht live, sobald sie freigegeben ist}}. Wenn {name} sie vorher ändert, braucht die Veröffentlichung eine neue Freigabe.',
  rejectDetail: '{name} wird benachrichtigt, und die Seite wird nicht veröffentlicht. Lehnen Sie ab, wenn sie gar nicht live gehen soll; für Korrekturen fordern Sie stattdessen Änderungen an.',
  changesDetail: 'Die Anfrage geht mit Ihrem Kommentar zurück an {name}. Die Seite wird erst veröffentlicht, wenn erneut angefragt wird.',
})
