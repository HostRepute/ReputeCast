import { translate } from '@/i18n'
import { m } from './policy.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  costTitle: 'Die geschätzten Kosten einer SMS- oder WhatsApp-Kampagne betragen mindestens',
  costDetail: 'Aus den Preisen der Anbieter für ihre Empfänger. Bei SMS bedeutet „die Sendeberechtigung“, sowohl Kampagnen als auch SMS-Kampagnen senden zu dürfen.',
  costAmount: 'Geschätzte Mindestkosten in Dollar',
  costInvalid: 'Geben Sie einen Betrag über 0 $ ein.',
})
