import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Apparence',
  caption: 'Clair ou sombre, sur chaque écran où vous vous connectez. Les e-mails, les pages publiques et leurs aperçus gardent leurs propres couleurs.',
  legend: 'Mode',
  system: 'Comme l’appareil',
  light: 'Clair',
  dark: 'Sombre',
  systemDetail: 'Change avec le réglage clair ou sombre de votre appareil. Il est en mode {theme} en ce moment.',
  lightDetail: 'Texte foncé sur fond clair.',
  darkDetail: 'Texte clair sur fond sombre. Plus reposant pour les yeux le soir.',
  nowLight: 'clair',
  nowDark: 'sombre',
  savedNote: 'Enregistrée dans votre compte, elle s’applique donc dans chaque navigateur où vous vous connectez.',
  switchedLight: 'Mode clair activé.',
  switchedDark: 'Mode sombre activé.',
  switchedSystem: 'L’apparence suit votre appareil.',
  notSaved: 'Votre apparence n’a pas été enregistrée, elle est donc revenue comme avant.',
  menuLabel: 'Apparence',
  useLight: 'Utiliser le mode clair',
  useDark: 'Utiliser le mode sombre',
  useSystem: 'Utiliser l’apparence de l’appareil',
  paletteDetail: 'Apparence',
})
