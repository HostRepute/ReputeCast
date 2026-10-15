import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { http } from '@/api/client'
import { AuthLanguageMenu } from '@/features/language/AuthLanguageMenu'
import { useDocumentTitle } from '@/lib/hooks'
import { LogoMark } from '../icons/Icon'

export function useInstanceMeta() {
  return useQuery({
    queryKey: ['meta'],
    queryFn: async () => (await http.GET('/api/v1/meta')).data!,
    staleTime: 5 * 60_000,
  })
}

/** Centered card used by sign-in, two-factor, invite, reset and first-run screens. */
export function AuthLayout({ title, children, labelledBy, wide }: { title: string; children: ReactNode; labelledBy: string; wide?: boolean }) {
  useDocumentTitle(`${title} · ReputeCast`)
  const { data } = useInstanceMeta()
  return (
    <div className="auth-page rc-app">
      <main id="main" className="w-full" style={{ maxWidth: wide ? 560 : 440 }}>
        <section className="card auth" aria-labelledby={labelledBy} style={wide ? { maxWidth: 560 } : undefined}>
          <InstanceBrand />
          {children}
        </section>
        <AuthLanguageMenu languages={data?.languages} />
      </main>
    </div>
  )
}

/** Instance name and host, as set by the instance admin. */
export function InstanceBrand() {
  const { data } = useInstanceMeta()
  const host = data?.publicUrl ? safeHost(data.publicUrl) : ''
  return (
    <div className="brand">
      {data?.logoSvg ? (
        // An <img> never runs scripts inside an SVG, so an uploaded logo can't inject code.
        <img src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(data.logoSvg)}`} width={28} height={28} alt="" />
      ) : (
        <LogoMark size={28} />
      )}
      <span>
        {data?.instanceName ?? 'ReputeCast'} {host ? <small>{host}</small> : null}
      </span>
    </div>
  )
}

function safeHost(u: string) {
  try {
    return new URL(u).host
  } catch {
    return ''
  }
}
