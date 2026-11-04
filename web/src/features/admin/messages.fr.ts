import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Administration de l’instance',
  documentTitle: '{page} · Administration de l’instance',
  adminsOnly: 'admins uniquement',
  workspaceView: 'Vue de l’espace de travail',
  workspaceSettings: 'Paramètres de l’espace de travail',
  tabsLabel: 'Sections de l’administration de l’instance',
  notAdminBadge: '403 · admins de l’instance uniquement',
  notAdminTitle: 'Seuls les admins de l’instance peuvent ouvrir l’administration de l’instance',
  backToOverview: 'Retour à la vue d’ensemble',
  account: 'Compte et sécurité',
  notAdminFootnote: 'Demandez à un admin de l’instance pour modifier la connexion, le SSO ou les espaces de travail.',
  notAdminBody: 'L’administration de l’instance gère les utilisateurs, les méthodes de connexion, le SSO, les politiques de sécurité et les espaces de travail de toute l’instance. Vos propres méthodes de connexion se trouvent dans Compte et sécurité.',
})
