import { translate } from '@/i18n'
import { ma } from './access.messages'

/** French for Website › Access › Comments. */
export default translate(ma, {
  title: 'Commentaires',
  text: 'Les membres commentent sous les articles qu’ils peuvent lire. Désactivés pour chaque site jusqu’à ce que vous les activiez.',
  waiting: '{count, plural, one {# en attente de validation} other {# en attente de validation}}',
  who: 'Qui peut commenter',
  off: 'Désactivés',
  offText: 'Pas de section commentaires sous les articles. Les commentaires déjà écrits sont conservés pour quand vous les réactiverez.',
  members: 'Membres connectés',
  membersText: 'Tout membre qui peut lire l’article',
  paid: 'Membres payants uniquement',
  paidText: 'Les membres gratuits lisent les commentaires et voient un lien pour passer à l’offre payante',
  review: 'Validation',
  reviewAll: 'Chaque commentaire',
  reviewAllText: 'Rien ne s’affiche tant qu’une personne qui modère les commentaires ne l’a pas publié',
  reviewFirst: 'Le premier commentaire d’un membre',
  reviewFirstText: 'Après un commentaire publié, ceux du membre s’affichent aussitôt',
  reviewNone: 'Aucune',
  reviewNoneText: 'Les commentaires s’affichent aussitôt',
  links: 'Quel que soit votre choix, un commentaire avec 2 liens ou plus attend une validation.',
  moderatedBy: 'Modéré par',
  moderatedByNote: 'les personnes qui peuvent modérer les commentaires',
  nobody: 'Personne pour l’instant',
  perPost: 'Par article',
  perPostText: 'Un article peut désactiver les commentaires dans ses réglages web',
  limits: 'Limites',
  limitsText: '{chars} caractères par commentaire ; 10 par membre en 10 minutes et 50 par jour',
})
