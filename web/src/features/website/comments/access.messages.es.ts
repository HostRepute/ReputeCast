import { translate } from '@/i18n'
import { ma } from './access.messages'

/** Spanish for Website › Access › Comments. */
export default translate(ma, {
  title: 'Comentarios',
  text: 'Los miembros comentan bajo los artículos que pueden leer. Desactivados en todos los sitios hasta que los actives.',
  waiting: '{count, plural, one {# pendiente de revisión} other {# pendientes de revisión}}',
  who: 'Quién puede comentar',
  off: 'Desactivados',
  offText: 'Sin sección de comentarios bajo los artículos. Los comentarios ya escritos se guardan para cuando los vuelvas a activar.',
  members: 'Miembros con sesión iniciada',
  membersText: 'Cualquier miembro que pueda leer el artículo',
  paid: 'Solo miembros de pago',
  paidText: 'Los miembros gratuitos leen los comentarios y ven un enlace para mejorar su plan',
  review: 'Revisión',
  reviewAll: 'Cada comentario',
  reviewAllText: 'Nada se muestra hasta que alguien que modera comentarios lo publique',
  reviewFirst: 'El primer comentario de un miembro',
  reviewFirstText: 'Tras un comentario publicado, los comentarios del miembro se muestran al instante',
  reviewNone: 'Ninguna',
  reviewNoneText: 'Los comentarios se muestran al instante',
  links: 'Elijas lo que elijas, un comentario con 2 o más enlaces espera revisión.',
  moderatedBy: 'Moderado por',
  moderatedByNote: 'personas que pueden moderar comentarios',
  nobody: 'Nadie todavía',
  perPost: 'Por artículo',
  perPostText: 'Un artículo puede desactivar los comentarios en sus ajustes web',
  limits: 'Límites',
  limitsText: '{chars} caracteres por comentario; 10 por miembro en 10 minutos y 50 al día',
})
