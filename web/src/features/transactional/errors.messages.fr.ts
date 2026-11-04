import { translate } from '@/i18n'
import { me } from './errors.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(me, {
  template_not_found_when: 'L’ID n’existe pas dans cet espace de travail',
  template_not_found_do: 'Vérifiez l’ID dans Modèles',
  template_not_published_when: 'Le modèle a un brouillon mais aucune version en ligne',
  template_not_published_do: 'Publiez-le d’abord',
  missing_variables_when: 'Une variable obligatoire manque dans {data}',
  missing_variables_do: 'La réponse indique lesquelles',
  invalid_recipient_when: 'L’adresse n’est pas une adresse e-mail valide',
  invalid_recipient_do: 'Validez-la dans votre application',
  recipient_suppressed_when: 'L’adresse a généré un rebond ou une plainte, ou c’est un piège à spam',
  recipient_suppressed_do: 'La réponse indique le motif et la date',
  sender_not_verified_when: '{from} n’est pas un expéditeur Transactionnel vérifié',
  sender_not_verified_do: 'Vérifiez-le dans Paramètres › Expéditeurs',
  attachments_too_large_when: 'Au-delà de la limite de la route',
  attachments_too_large_do: '{size} chez {provider}',
  rate_limited_when: 'Au-delà du débit de la clé',
  rate_limited_do: 'Respectez {header}',
  no_healthy_route_when: 'Tous les fournisseurs de la route sont en panne',
  no_healthy_route_do: 'Réessayez avec la même clé d’idempotence',
  idempotency_key_reused_when: 'La même clé est arrivée avec une requête différente en moins de 24 heures',
  idempotency_key_reused_do: 'Utilisez une nouvelle clé pour un nouveau message',
  template_not_allowed_when: 'La clé ne peut pas envoyer ce modèle ou cette catégorie',
  template_not_allowed_do: 'Utilisez une clé qui le peut, ou élargissez la portée de la clé',
})
