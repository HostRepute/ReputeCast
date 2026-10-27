import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Icon } from '@/components/icons/Icon'
import { useT, type MessageDescriptor } from '@/i18n'
import { dayMonthDate, orList } from '@/lib/format'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { FormsTabs, throwIfForbidden } from '@/features/forms/common'
import { bare, useWebsite, type Address, type Overview } from './api'
import { m } from './messages'
import './website.css'

export type WebsiteView = 'posts' | 'home' | 'access'

/**
 * Loads the website's overview and hands it on; its own loading, error and
 * 403 states keep the Forms & pages frame around them.
 */
export function WithWebsite({ children }: { children: (ov: Overview) => ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('website:view')
  const ov = useWebsite()
  throwIfForbidden(ov.error)
  if (!can) {
    return (
      <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
        <ForbiddenState
          permission="website:view"
          title={t(m.forbiddenTitle)}
          readOnly={
            <ButtonLink to="/w/$workspace/forms" params={{ workspace: slug }}>
              {t(m.screenTitle)}
            </ButtonLink>
          }
        />
      </Page>
    )
  }
  if (ov.isError && !ov.data) {
    return (
      <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void ov.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(ov.error)} {t(m.loadFailedText)}
        </ErrorPanel>
      </Page>
    )
  }
  if (!ov.data) {
    return (
      <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
        <Loading label={t(m.loading)} className="ws-main">
          <div className="flex justify-between gap-3">
            <Skeleton w={300} h={30} r={999} />
            <Skeleton w={260} h={30} r={8} />
          </div>
          <div className="ws-stats">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="card ws-stat">
                <Skeleton w={90} />
                <Skeleton w={110} h={22} />
                <Skeleton w={160} />
              </div>
            ))}
          </div>
          <div className="card pad">
            <Skeleton w="100%" h={360} />
          </div>
        </Loading>
      </Page>
    )
  }
  return <>{children(ov.data)}</>
}

/**
 * One view of the website while it's on (WebsitePosts, WebsiteHomeNav,
 * WebsiteAccess): Forms & pages with its tabs, the views as chips, and the
 * site's address with View site.
 */
export function WebsitePage({ view, overview, actions, children, documentTitle }: { view: WebsiteView; overview: Overview; actions?: ReactNode; children: ReactNode; documentTitle?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  const views: { key: WebsiteView; to: '/w/$workspace/forms/website' | '/w/$workspace/forms/website/home' | '/w/$workspace/forms/website/access'; label: string }[] = [
    { key: 'posts', to: '/w/$workspace/forms/website', label: t(m.viewPosts) },
    { key: 'home', to: '/w/$workspace/forms/website/home', label: t(m.viewHomeNav) },
    { key: 'access', to: '/w/$workspace/forms/website/access', label: t(m.viewAccess) },
  ]
  const current = views.find((v) => v.key === view)!
  return (
    <Page title={t(m.screenTitle)} documentTitle={documentTitle ?? `${current.label} · ${t(m.screenTitle)}`} filters={<FormsTabs />} actions={actions}>
      <div className="ws-main">
        <div className="ws-vbar">
          <nav className="ws-vp" aria-label={t(m.viewsLabel)}>
            {views.map((v) => (
              <Link key={v.key} to={v.to} params={params} className={v.key === view ? 'ws-fchip on' : 'ws-fchip'} aria-current={v.key === view ? 'page' : undefined} activeOptions={{ exact: true }}>
                {v.label}
              </Link>
            ))}
          </nav>
          <div className="ws-vr">
            <LiveBadge address={overview.address} />
            <ViewSite address={overview.address} />
          </div>
        </div>
        {children}
      </div>
    </Page>
  )
}

/** "● Live · news.acme.io · certificate valid", or what's wrong with the address. */
export function LiveBadge({ address }: { address: Address }) {
  const t = useT()
  const a = address
  if (a.domain && a.domainStatus !== 'verified') {
    return (
      <Badge tone="warn" dot>
        {t(m.liveDomainPending, { host: a.fallback, domain: a.domain })}
      </Badge>
    )
  }
  if (!a.domain) {
    return (
      <Badge tone="ok" dot>
        {t(m.liveNoDomain, { host: a.host })}
      </Badge>
    )
  }
  if (a.certificate.status === 'failing') {
    return (
      <Badge tone="bad" dot>
        {t(m.liveFailing, { host: a.host })}
      </Badge>
    )
  }
  if (a.certificate.status !== 'valid') {
    return (
      <Badge tone="warn" dot>
        {t(m.livePending, { host: a.host })}
      </Badge>
    )
  }
  return (
    <Badge tone="ok" dot>
      {t(m.liveValid, { host: a.host })}
    </Badge>
  )
}

/** View site: the site's home in a new tab. */
export function ViewSite({ address, primary, size = 'sm' }: { address: Address; primary?: boolean; size?: 'sm' | 'md' }) {
  const t = useT()
  return (
    <a className={`btn${size === 'sm' ? ' sm' : ''}${primary ? ' pri' : ''}`} href={address.url} target="_blank" rel="noreferrer">
      <Icon name="external" size={13} />
      {t(m.viewSite)}
    </a>
  )
}

/** "All changes saved · 29 Sep, Linh Nguyen", next to Save changes. */
export function SavedLine({ at, by }: { at: string; by?: string }) {
  const t = useT()
  return <span className="k max-md:hidden">{by ? t(m.allSaved, { date: dayMonthDate(at), name: by }) : t(m.allSavedNoName, { date: dayMonthDate(at) })}</span>
}

/**
 * "Read-only · Aisha Bello, Viewer. You can see every setting. Your role
 * can't … Ask Sofia Rossi, Marcus Chen or Linh Nguyen." `msg` says what
 * the role can't do in words, never the permission key (review X-1).
 */
export function ReadOnlyNote({ msg, helpers }: { msg: MessageDescriptor; helpers: string[] }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data?.user
  return (
    <section className="ws-ro" role="status">
      <div className="hd">
        <Icon name="info" size={15} />
        {t(m.readOnlyTitle, { name: user?.name ?? '', role: me.workspace.roleName })}
      </div>
      <div className="bd">
        {t(msg)}
        {helpers.length ? ` ${t(m.askHelpers, { helpers: orList(helpers) })}` : null}
      </div>
    </section>
  )
}

/** "news.acme.io" for "https://news.acme.io/". */
export { bare }
