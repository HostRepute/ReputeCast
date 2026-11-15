import { translate } from '@/i18n'
import { ma } from './automation-draft.messages'

export default translate(ma, {
  title: 'Automation beschreiben',
  lead: 'Eine E-Mail-Serie für Listenbeitritte entwerfen. Prüfe Schritte und jede E-Mail, bevor du das Paket speicherst.',
  brief: 'Was soll diese Serie tun?',
  hint: 'Beschreibe E-Mails, Wartezeiten und Interaktionszweige. Keine Kontaktdaten, Zugangsdaten oder URLs eingeben.',
  name: 'Name der Automation',
  list: 'Ausgewählte Liste',
  chooseList: 'Liste auswählen',
  audience: 'Nach deiner manuellen Veröffentlichung: nur zukünftige bestätigte Beitritte, einmal je Person, ohne Importe. Wer die Liste verlässt, verlässt auch den Ablauf. Das Speichern nimmt keine bestehenden Mitglieder auf.',
  scope: 'Diese Vorlage unterstützt E-Mail, Wartezeit, Interaktionszweig, A/B-Aufteilung und Ende. Ausgewählte Links, Bilder und Liquid bleiben erhalten. Andere Auslöser, Kontaktbedingungen, Webhooks und Abo-Aktionen benötigen den Ablaufeditor.',
  privacy: 'Beschreibung, ausgewählte redaktionelle Texte, erlaubte Schritte und Markenstimme gehen an dein Modell. IDs, Listen- und E-Mail-Namen, Linkziele und Fußzeilen bleiben hier. Keine Kontakte werden gelesen und keine URLs abgerufen.',
  waiting: 'Schritte und E-Mails werden entworfen. Noch nichts gespeichert. Auch nach dem Stoppen kann der Anbieter Gebühren berechnen.',
  saved: 'Automation und E-Mails als Entwürfe gespeichert. Nichts veröffentlicht, niemand aufgenommen.',
  rejected: 'Vorschlag abgelehnt. Kein Ablauf und keine E-Mail geändert.',
  leaveBody: 'Die ungespeicherte Beschreibung wird geschlossen. Öffne diesen Dialog erneut, um den Vorschlag vor Ablauf der 24 Stunden zu prüfen. Kein Ablauf und keine E-Mail wird geändert.',
  unsaved: 'Warte, bis der Ablauf gespeichert ist, bevor du einen Ersatz erstellst oder annimmst.',
  replace: 'Vergleiche den gespeicherten Graphen und die E-Mail-Inhalte mit dem Vorschlag. Die Annahme erstellt neue E-Mail-Kopien und ersetzt diesen Entwurfsgraphen; bestehende E-Mails bleiben unverändert.',
  unsupported: 'Diese Vorlage benötigt zukünftige bestätigte Listenbeitritte, einmal pro Person, ohne Importe und mit Austritt beim Verlassen. Andere Zielgruppenregeln bleiben im Ablaufeditor.',
  graph: 'Schritte und Pfade',
  emails: 'E-Mail-Entwürfe',
  exact: 'Exakten Graphen prüfen',
  next: 'Weiter: {steps}',
  end: 'Ende des Pfads',
  checked: 'Graph und E-Mail-Inhalte geprüft. Veröffentlichungsprüfungen gelten weiterhin.',
})
