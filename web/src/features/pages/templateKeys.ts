import type { Schemas } from '@/api/client'

/**
 * The six templates of "New page", in the picker's order. Its own module so
 * the Pages route's search schema doesn't load the pages feature with the
 * app (review 2026-11 FE-9).
 */
export const TEMPLATE_KEYS = ['about', 'lead_magnet', 'launch', 'webinar', 'sales', 'blank'] as const satisfies readonly Schemas['PageListItem']['template'][]
