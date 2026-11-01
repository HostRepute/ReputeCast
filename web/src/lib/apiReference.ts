import { useInstanceMeta } from '@/components/shell/AuthLayout'

/**
 * The API reference this instance serves, of the version it runs (review
 * 2026-11 IN-4): not the repository's main branch, which may be ahead.
 */
export function apiReferenceHref(version?: string, anchor?: string): string {
  return `/api/reference${version ? `?version=${encodeURIComponent(version)}` : ''}${anchor ? `#${anchor}` : ''}`
}

export function useApiReference(anchor?: string): string {
  return apiReferenceHref(useInstanceMeta().data?.version, anchor)
}
