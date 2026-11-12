import { translate } from '@/i18n'
import { sm } from './segment.messages'

export default translate(sm, {
  visitorsInSegment: 'Dans un segment',
  visitorsNotInSegment: 'Hors d’un segment',
  segment: 'Segment',
  pickSegment: 'Choisir un segment',
  needsTrackingIn:
    'Nécessite le suivi du site web sur le même site. Le formulaire ne s’affiche qu’aux visiteurs que ReputeCast connaît et qui sont dans le segment ; à personne d’autre.',
  needsTrackingNotIn:
    'Nécessite le suivi du site web sur le même site. Le formulaire est masqué pour les visiteurs que ReputeCast connaît et qui sont dans le segment ; tous les autres le voient.',
  inSegmentVisitors: 'les visiteurs d’un segment',
  notInSegmentVisitors: 'les visiteurs hors d’un segment',
  segmentsError: 'Impossible de charger les segments.',
})
