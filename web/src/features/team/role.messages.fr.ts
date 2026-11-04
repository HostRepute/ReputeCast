import { translate } from '@/i18n'
import { m } from './role.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  eyebrow: 'Boîte de dialogue · changer le rôle ou la portée',
  title: 'Changer le rôle ou la portée · {name}',
  description: 'Les modifications s’appliquent immédiatement, à la prochaine requête du membre {name}. L’ancienne et la nouvelle valeur sont consignées dans le journal d’audit.',
  nothingChanged: 'Rien n’a encore été modifié.',
  current: '{role} (actuel)',
  updated: 'Membre mis à jour\u00A0: {name}',
  scopeChanged: 'portée modifiée',
  nowRole: 'désormais {role}',
  ssoGroup: 'Le rôle du membre {name} vient d’une correspondance de groupe SSO. Une modification ici tient jusqu’à ce que la prochaine connexion réapplique la correspondance\u202F; modifiez la correspondance pour la rendre durable.',
  allScope: 'Toutes les listes et tous les domaines d’expéditeur',
  lastOwnerEyebrow: 'Boîte de dialogue · changer le rôle · bloqué',
  lastOwnerTitle: '{name} est le dernier Propriétaire',
  onlyOwnerTransfers: 'Seul le Propriétaire peut transférer la propriété.',
  transfer: 'Transférer la propriété',
  cantDemote: 'Rétrogradation impossible',
  keepsOwner: 'Un espace de travail conserve toujours au moins un Propriétaire\u00A0: ce rôle ne peut donc pas passer à Admin, et le membre ne peut être ni retiré ni désactivé.',
  transferFirst: 'Pour le modifier, transférez d’abord la propriété à un autre membre ayant activé la 2FA. {name} devient alors automatiquement Admin.',
  transferFirstSuggest: 'Pour le modifier, transférez d’abord la propriété à un autre membre ayant activé la 2FA, par exemple {suggest}. {name} devient alors automatiquement Admin.',
})
