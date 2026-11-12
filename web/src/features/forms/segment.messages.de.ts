import { translate } from '@/i18n'
import { sm } from './segment.messages'

export default translate(sm, {
  visitorsInSegment: 'In einem Segment',
  visitorsNotInSegment: 'Nicht in einem Segment',
  segment: 'Segment',
  pickSegment: 'Segment auswählen',
  needsTrackingIn:
    'Erfordert Website-Tracking auf derselben Website. Das Formular erscheint nur Besuchern, die ReputeCast kennt und die im Segment sind; allen anderen nicht.',
  needsTrackingNotIn:
    'Erfordert Website-Tracking auf derselben Website. Das Formular bleibt Besuchern verborgen, die ReputeCast kennt und die im Segment sind; alle anderen sehen es.',
  inSegmentVisitors: 'Besucher in einem Segment',
  notInSegmentVisitors: 'Besucher nicht in einem Segment',
  segmentsError: 'Die Segmente konnten nicht geladen werden.',
})
