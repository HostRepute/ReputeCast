import { translate } from '@/i18n'
import { mc } from './messages'

/** French for the composer's Discount code block (M59). */
export default translate(mc, {
  block: 'Code de réduction',
  name: 'Code de réduction',
  defaultLabel: 'Votre code',
  defaultButton: 'Acheter',
  canvasPick: 'Choisissez une réserve de codes de réduction dans les paramètres du bloc.',
  canvasUnknown: 'Aucune réserve de codes de réduction ne s’appelle {slug}. Choisissez-en une autre dans les paramètres du bloc.',
  oneCode: 'Un code par lecteur, attribué à l’envoi de l’e-mail.',
  pool: 'Réserve',
  poolOption: '{name} · {free} libres',
  poolNone: 'Choisir une réserve',
  poolStore: '{platform} · {store}',
  poolUpload: 'Codes importés',
  poolOnePer: 'un code par contact',
  poolLiquid: 'en Liquid <mono>{token}</mono>',
  noPools: 'Aucune réserve pour l’instant. <a>Créez-en une</a> dans Campagnes › Codes de réduction.',
  label: 'Libellé',
  button: 'Bouton',
  buttonOn: 'Afficher le bouton',
  buttonOpens: 'Ouvre {store} avec le code appliqué : <mono>{token}</mono>',
  buttonApply: 'Ouvre l’adresse d’application avec le code : <mono>{token}</mono>',
  buttonNoUrl: '{name} n’a pas d’adresse d’application : le bloc n’affiche donc pas de bouton. Ajoutez-en une sur la page de la réserve.',
  manage: 'Gérer les codes de réduction',
  manageNote: 'Les réserves, leurs chiffres et où chacune sert, dans Campagnes › Codes de réduction.',
  pass: 'Conforme',
  passNote: 'Vérifier et envoyer s’assure qu’il y a un code pour chaque destinataire.',
  warn: 'Attention',
  archivedNote: '{name} est archivée : les lecteurs ne reçoivent pas de code. Choisissez une autre réserve.',
  tokensGroup: 'Codes de réduction',
})
