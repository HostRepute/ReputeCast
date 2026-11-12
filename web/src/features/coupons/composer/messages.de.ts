import { translate } from '@/i18n'
import { mc } from './messages'

/** German for the composer's Discount code block (M59). */
export default translate(mc, {
  block: 'Rabattcode',
  name: 'Rabattcode',
  defaultLabel: 'Ihr Code',
  defaultButton: 'Jetzt einkaufen',
  canvasPick: 'Wählen Sie in den Einstellungen des Blocks einen Rabattcode-Pool.',
  canvasUnknown: 'Es gibt keinen Rabattcode-Pool namens {slug}. Wählen Sie in den Einstellungen des Blocks einen anderen.',
  oneCode: 'Ein Code pro Leser, vergeben beim Senden der E-Mail.',
  pool: 'Pool',
  poolOption: '{name} · {free} frei',
  poolNone: 'Pool wählen',
  poolStore: '{platform} · {store}',
  poolUpload: 'Hochgeladene Codes',
  poolOnePer: 'ein Code pro Kontakt',
  poolLiquid: 'in Liquid <mono>{token}</mono>',
  noPools: 'Noch kein Pool. <a>Erstellen Sie einen</a> unter Kampagnen › Rabattcodes.',
  label: 'Beschriftung',
  button: 'Button',
  buttonOn: 'Button anzeigen',
  buttonOpens: 'Öffnet {store} mit dem eingelösten Code: <mono>{token}</mono>',
  buttonApply: 'Öffnet die Einlöseadresse mit dem Code: <mono>{token}</mono>',
  buttonNoUrl: '{name} hat keine Einlöseadresse, daher zeigt der Block keinen Button. Fügen Sie auf der Seite des Pools eine hinzu.',
  manage: 'Rabattcodes verwalten',
  manageNote: 'Pools, ihre Zahlen und wo jeder verwendet wird, unter Kampagnen › Rabattcodes.',
  pass: 'OK',
  passNote: 'Prüfen & senden prüft, ob es für jeden Empfänger einen Code gibt.',
  warn: 'Warnung',
  archivedNote: '{name} ist archiviert: Leser bekommen keinen Code. Wählen Sie einen anderen Pool.',
  tokensGroup: 'Rabattcodes',
})
