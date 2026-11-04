import { translate } from '@/i18n'
import { m } from './policy.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  costTitle: 'Une campagne SMS ou WhatsApp a un coût estimé d’au moins',
  costDetail: 'D’après les tarifs des fournisseurs pour ses destinataires. Pour les SMS, «\u00A0l’autorisation d’envoi\u00A0» signifie pouvoir envoyer à la fois des campagnes et des campagnes SMS.',
  costAmount: 'Coût estimé minimal, en dollars',
  costInvalid: 'Saisissez un montant supérieur à 0\u00A0$.',
})
