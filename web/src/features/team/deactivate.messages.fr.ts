import { translate } from '@/i18n'
import { m } from './deactivate.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  eyebrow: 'Boîte de dialogue · désactiver un membre',
  titleOne: 'Désactiver {name}\u202F?',
  titleMany: 'Désactiver {count, plural, one {# membre} other {# membres}}\u202F?',
  descriptionOne: 'Le membre {name} conserve son historique et pourra être réactivé plus tard. Le retirer supprimerait au contraire son appartenance à l’espace\u202F; le journal d’audit garderait son nom.',
  descriptionMany: 'Ces membres conservent leur historique et pourront être réactivés plus tard. Le journal d’audit garde leur nom.',
  loadingChanges: 'Chargement des conséquences…',
  confirmOne: 'Désactiver {name}',
  confirmMany: 'Désactiver {count, plural, one {# membre} other {# membres}}',
  doneOne: 'Membre désactivé\u00A0: {name}',
  doneMany: '{count, plural, one {# membre désactivé} other {# membres désactivés}}',
  doneDetail: '{count, plural, one {# session fermée} other {# sessions fermées}}',
  failedOne: 'Impossible de désactiver {name}',
  failedMany: 'Impossible de désactiver {count, plural, one {# membre} other {# membres}}',
  signedOutEverywhere: '{count, plural, one {Déconnexion partout, immédiatement (# session)} other {Déconnexion partout, immédiatement (# sessions)}}',
  cantSignIn: 'Ne peut plus se connecter à {workspace}\u202F; les autres espaces de travail ne sont pas concernés',
  scheduled: '{count, plural, one {# campagne programmée conserve sa programmation et part au nom de l’espace de travail} other {# campagnes programmées conservent leur programmation et partent au nom de l’espace de travail}}',
  approvals: '{count, plural, one {# demande d’approbation en attente reste ouverte pour les approbateurs} other {# demandes d’approbation en attente restent ouvertes pour les approbateurs}}',
  keysLegend: 'Clés API créées · {count}',
  keepKey: 'Conserver la clé {keys} active <k>({permissions})</k>',
  keepKeys: 'Conserver les clés {keys} actives',
  revoke: '{count, plural, one {La révoquer maintenant} other {Les révoquer maintenant}}',
  reason: 'Motif <k>· consigné dans le journal d’audit</k>',
  reasonPlaceholder: 'Fin de contrat',
})
