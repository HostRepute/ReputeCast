import { defineMessages } from '@/i18n'

/**
 * Form targeting by segment (M62 G-32): "Where and when" › Visitors shows
 * a form only to, or hides it from, the contacts of a segment, known
 * through Website tracking on the same site.
 */
export const sm = defineMessages('forms.segment', {
  visitorsInSegment: 'In a segment',
  visitorsNotInSegment: 'Not in a segment',
  segment: 'Segment',
  pickSegment: 'Pick a segment',
  needsTrackingIn:
    "Needs Website tracking on the same site. The form shows only to visitors ReputeCast knows who are in the segment; it doesn't show to anyone else.",
  needsTrackingNotIn: 'Needs Website tracking on the same site. The form hides from visitors ReputeCast knows who are in the segment; everyone else sees it.',
  /** In the targeting summary: "Shown on 3 pages · visitors in a segment". */
  inSegmentVisitors: 'visitors in a segment',
  notInSegmentVisitors: 'visitors not in a segment',
  segmentsError: "Couldn't load the segments.",
})
