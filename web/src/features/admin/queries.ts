import { useQueryClient } from '@tanstack/react-query'

/** Refreshes what the Users tab shows after an action on an account. */
export function useInvalidateUsers() {
  const qc = useQueryClient()
  return () =>
    Promise.all([
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/users'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/policy'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/sign-ins'] }),
    ])
}

/** Refreshes the SSO provider lists and pages. */
export function useInvalidateSSO() {
  const qc = useQueryClient()
  return () =>
    Promise.all([
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/sso'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/sso/{providerId}'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/policy'] }),
      qc.invalidateQueries({ queryKey: ['meta'] }),
    ])
}
