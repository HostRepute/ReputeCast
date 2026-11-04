import { translate } from '@/i18n'
import { m } from './senders.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Domaine de liens courts',
  sub: 'Chaque lien d’un SMS est raccourci ici, avec un code par destinataire. Les raccourcisseurs publics comme bit.ly sont bloqués\u00A0: les opérateurs les filtrent.',
  caption: 'Domaine de liens courts',
  domain: 'Domaine',
  usedBy: 'Utilisé par',
  smsLinks: 'Liens des SMS',
  record: 'Enregistrement',
  status: 'État',
  lastCheck: 'Dernière vérification',
  cname: 'CNAME {name} → {target}',
  verifiedOn: 'Vérifié le {date}',
  notVerified: 'Non vérifié',
  checkFailed: 'Échec de la vérification',
  never: 'Pas encore vérifié',
  checkDns: 'Vérifier le DNS',
  checked: 'DNS vérifié',
  manage: 'Routage et règles',
  manageLabel: 'Modifier le domaine de liens courts dans Routage et règles',
  none: 'Aucun domaine de liens courts pour l’instant. Les SMS contenant des liens ne peuvent pas être envoyés tant qu’un domaine n’est pas vérifié.',
  setUp: 'En configurer un',
  untilVerified: 'Tant qu’il n’est pas vérifié, les SMS contenant des liens ne peuvent pas être envoyés. Les SMS sans lien, si.',
})
