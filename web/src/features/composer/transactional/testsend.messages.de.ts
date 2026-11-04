import { translate } from '@/i18n'
import { mts } from './testsend.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(mts, {
  menuItem: 'Test-E-Mail senden…',
  title: 'Test von {key} senden',
  sendTo: 'Senden an',
  ownOnly: 'Ohne transactional:send gehen Tests nur an Ihre eigene Adresse.',
  fillWith: 'Füllen mit',
  samples: 'Beispielwerte',
  testData: 'Testdaten · {name}',
  note: 'Der Entwurf im aktuellen Stand, über die Route System, mit [Test] im Betreff. Er zählt nicht zu den Zahlen im Protokoll.',
  send: 'Test senden',
  sending: 'Wird gesendet…',
  sent: 'Test an {email} gesendet',
  sentWith: 'Test an {email} mit „{name}“ gesendet',
})
