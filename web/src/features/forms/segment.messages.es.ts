import { translate } from '@/i18n'
import { sm } from './segment.messages'

export default translate(sm, {
  visitorsInSegment: 'En un segmento',
  visitorsNotInSegment: 'Fuera de un segmento',
  segment: 'Segmento',
  pickSegment: 'Elige un segmento',
  needsTrackingIn:
    'Necesita el seguimiento del sitio web en el mismo sitio. El formulario solo se muestra a los visitantes que ReputeCast conoce y que están en el segmento; a nadie más.',
  needsTrackingNotIn:
    'Necesita el seguimiento del sitio web en el mismo sitio. El formulario se oculta a los visitantes que ReputeCast conoce y que están en el segmento; los demás lo ven.',
  inSegmentVisitors: 'visitantes de un segmento',
  notInSegmentVisitors: 'visitantes fuera de un segmento',
  segmentsError: 'No se pudieron cargar los segmentos.',
})
