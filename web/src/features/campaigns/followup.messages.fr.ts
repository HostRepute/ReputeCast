import { translate } from '@/i18n'
import { mf } from './followup.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mf, {
  menu: 'Relancer',
  nonOpeners: 'Renvoyer aux non-ouvreurs…',
  clickers: 'Envoyer aux cliqueurs…',
  titleNonOpeners: 'Renvoyer aux personnes qui n’ont pas ouvert',
  titleClickers: 'Envoyer aux personnes qui ont cliqué',
  bodyNonOpeners: '{count, plural, one {<b># contact</b> a reçu «\u00A0{name}\u00A0» et ne l’a pas ouverte.} other {<b># contacts</b> ont reçu «\u00A0{name}\u00A0» et ne l’ont pas ouverte.}}',
  machineOnly: '{count, plural, one {# l’a ouverte uniquement via un proxy de confidentialité comme Apple Mail Privacy Protection\u00A0: ce contact compte comme inconnu et est écarté.} other {# l’ont ouverte uniquement via un proxy de confidentialité comme Apple Mail Privacy Protection\u00A0: ces contacts comptent comme inconnus et sont écartés.}}',
  bodyClickers: '{count, plural, one {<b># personne</b> a cliqué sur un lien dans «\u00A0{name}\u00A0».} other {<b># personnes</b> ont cliqué sur un lien dans «\u00A0{name}\u00A0».}} Les clics des scanners de liens ne comptent pas.',
  nobody: 'Personne n’est encore dans ce groupe\u00A0: la relance n’atteindrait donc personne pour l’instant. Son audience reste à jour jusqu’à l’envoi.',
  who: 'L’envoyer à',
  whoNonOpeners: 'Personnes qui ne l’ont pas ouverte',
  whoClickers: 'Personnes qui ont cliqué',
  subject: 'Nouvel objet',
  subjectNonOpeners: 'Au cas où vous l’auriez manqué\u00A0: {subject}',
  subjectClickers: 'Pour aller plus loin\u00A0: {subject}',
  send: 'Envoi',
  after: '{days, plural, one {# jour après l’original ({when})} other {# jours après l’original ({when})}}',
  later: 'Choisir dans Vérifier et envoyer',
  note: 'Crée un brouillon avec le même contenu. Les exclusions, les désabonnements et le plafond marketing s’appliquent comme d’habitude.',
  create: 'Créer la relance',
  creating: 'Création…',
  created: 'Relance «\u00A0{name}\u00A0» créée',
  loading: 'Calcul des destinataires…',
  subjectRequired: 'Donnez un objet à la relance.',
})
