import { translate } from '@/i18n'
import { m } from './formfields.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  phone: 'Mobilnummer',
  phoneOptional: 'Mobilnummer · optional',
  phoneRequired: 'Mobilnummer · erforderlich',
  textsTitle: 'SMS',
  smsConsent: 'Kontrollkästchen für SMS-Einwilligung · standardmäßig leer',
  needsPhone: 'Erfordert das Feld „Mobilnummer“.',
  consentText: 'SMS-Einwilligungstext',
  evidence: 'Die Nummer, die Listen, genau dieser Text samt Version, die Seite, die IP-Adresse und der Zeitpunkt werden als Nachweis der SMS-Einwilligung gespeichert.',
  newVersion: 'Beim Speichern beginnt Version {version}; frühere Anmeldungen behalten den Text, dem sie zugestimmt haben.',
  separate: 'Eine E-Mail-Einwilligung meldet niemanden für SMS an: Dieses Kästchen steht für sich.',
  confirmation: 'Wer es ankreuzt, erhält die Opt-in-Bestätigung per SMS aus Routing & Regeln.',
  termsWords: 'SMS-Bedingungen',
})
