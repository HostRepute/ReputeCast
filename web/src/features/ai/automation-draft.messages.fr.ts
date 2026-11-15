import { translate } from '@/i18n'
import { ma } from './automation-draft.messages'

export default translate(ma, {
  title: 'Décrire l’automatisation',
  lead: 'Rédigez une séquence d’e-mails lors de l’inscription à une liste. Vérifiez chaque étape et e-mail avant d’enregistrer l’ensemble.',
  brief: 'Que doit faire cette séquence ?',
  hint: 'Décrivez les e-mails, délais et branches d’engagement. N’incluez aucune coordonnée, information d’identification ou URL.',
  name: 'Nom de l’automatisation',
  list: 'Liste sélectionnée',
  chooseList: 'Choisir une liste',
  audience: 'Après publication manuelle : inscriptions futures confirmées uniquement, une fois par personne, hors imports. Les lecteurs quittent le parcours en quittant la liste. Enregistrer le brouillon n’inscrit aucun membre existant.',
  scope: 'Cette recette prend en charge les e-mails, délais, branches d’engagement, répartitions A/B et sorties. Les liens, images et Liquid sélectionnés sont conservés. Les autres déclencheurs, conditions de contact, webhooks et actions d’abonnement nécessitent l’éditeur de parcours.',
  privacy: 'La description, les textes éditoriaux sélectionnés, les étapes autorisées et le ton enregistré sont envoyés au modèle. Les identifiants, noms de listes et d’e-mails, destinations des liens et pieds légaux restent ici. Aucun contact n’est lu, aucune URL consultée.',
  waiting: 'Création des étapes et e-mails. Aucun parcours ni e-mail enregistré. L’arrêt peut quand même entraîner des frais du fournisseur.',
  saved: 'Automatisation et e-mails enregistrés en brouillons. Rien n’est publié et personne n’est inscrit.',
  rejected: 'Proposition refusée. Aucun parcours ni e-mail modifié.',
  leaveBody: 'La description non enregistrée sera fermée. Rouvrez cette boîte de dialogue pour examiner la proposition avant son expiration après 24 heures. Aucun parcours ni e-mail n’est modifié.',
  unsaved: 'Attendez l’enregistrement du parcours avant de générer ou d’accepter un remplacement.',
  replace: 'Comparez le graphe et les e-mails enregistrés à la proposition. L’acceptation crée de nouvelles copies des e-mails et remplace ce graphe brouillon ; les e-mails existants restent inchangés.',
  unsupported: 'Cette recette exige des inscriptions futures confirmées, une fois par personne, hors imports, avec sortie en quittant la liste. Conservez les autres règles d’audience dans l’éditeur de parcours.',
  graph: 'Étapes et chemins',
  emails: 'Brouillons d’e-mails',
  exact: 'Vérifier le graphe exact',
  next: 'Suite : {steps}',
  end: 'Fin du chemin',
  checked: 'Graphe et contenu des e-mails validés. Les vérifications de publication restent nécessaires.',
})
