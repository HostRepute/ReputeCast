import type { Schemas } from '@/api/client'

/**
 * The workflow tools with a page of their own under Settings › Integrations.
 * Kept apart from shared.tsx so the route that checks its parameter doesn't
 * load the catalog's copy with the app (review 2026-11 FE-9).
 */
export type PlatformKind = Schemas['IntegrationApp']['kind']

export function isPlatform(k: string): k is PlatformKind {
  return k === 'zapier' || k === 'make' || k === 'n8n'
}
