import { Link, type LinkProps } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Note } from '@/components/ui/Note'
import { T, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { FormsTabs } from '@/features/forms/common'
import { useShopBuiltinPages } from '@/features/pages/shop'
import { m } from './messages'
import './website.css'

/** One built-in page: its name, address, group, state and editor. */
export interface BuiltinPage {
  key: string
  name: string
  addr: string
  /** Shown in mono (an address) or as words ("personal link in every email"). */
  group: string
  state: string
  /** A switch after the state, for a page that can be turned off (Shop: "On" and its switch). */
  control?: ReactNode
  to: LinkProps['to']
  search?: LinkProps['search']
}

/** The built-in pages of 1.0 and the website (design expansion phase 7). */
export function useBuiltinPages(): BuiltinPage[] {
  const t = useT()
  // Pages other areas add, each from its own hook, in the board's order
  // after Archive: Shop (X19), then Grow's Referral hub, Invite page,
  // Leaderboard and Recommendations, then Earn's Upgrade and Give a gift.
  const shop = useShopBuiltinPages()
  return [
    { key: 'subscribe', name: t(m.builtinSubscribe), addr: '/subscribe', group: t(m.groupSigningUp), state: t(m.alwaysOn), to: '/w/$workspace/forms/subscribe' },
    { key: 'confirm', name: t(m.builtinConfirm), addr: t(m.addrAfterConfirm), group: t(m.groupSigningUp), state: t(m.alwaysOn), to: '/w/$workspace/forms/subscribe' },
    { key: 'preferences', name: t(m.builtinPreferences), addr: t(m.addrPersonal), group: t(m.groupSigningUp), state: t(m.alwaysOn), to: '/w/$workspace/forms/preferences' },
    { key: 'unsubscribe', name: t(m.builtinUnsubscribe), addr: t(m.addrPersonal), group: t(m.groupSigningUp), state: t(m.alwaysOn), to: '/w/$workspace/forms/unsubscribe' },
    { key: 'signin', name: t(m.builtinSignin), addr: '/signin', group: t(m.groupWebsite), state: t(m.onWithWebsite), to: '/w/$workspace/forms/website' },
    { key: 'account', name: t(m.builtinAccount), addr: '/account', group: t(m.groupWebsite), state: t(m.onWithWebsite), to: '/w/$workspace/forms/website' },
    { key: 'archive', name: t(m.builtinArchive), addr: '/archive', group: t(m.groupWebsite), state: t(m.on), to: '/w/$workspace/forms/archive' },
    ...shop,
  ]
}

/** The Built-in pages card: two columns of pages, each opening its editor. */
export function BuiltinPagesCard({ pages }: { pages: BuiltinPage[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const half = Math.ceil(pages.length / 2)
  const cols = [pages.slice(0, half), pages.slice(half)].filter((c) => c.length > 0)
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.builtinTitle, { count: pages.length })}</h2>
          <span className="k">{t(m.builtinIntro)}</span>
        </div>
      </div>
      <div className="ws-bigrid">
        {cols.map((col, i) => (
          <div key={i}>
            {col.map((p) => (
              <div key={p.key} className="ws-bpg">
                <span className="nm">
                  <Link to={p.to} params={{ workspace: slug } as never} search={p.search as never}>
                    {p.name}
                  </Link>
                  <span className="ws-addr">{p.addr}</span>
                </span>
                <span className="ws-no-phone">
                  <Badge tone="neutral" className="ws-badge-sm">
                    {p.group}
                  </Badge>
                </span>
                <span className="st">
                  {p.state}
                  {p.control}
                </span>
                <Link className="btn sm" to={p.to} params={{ workspace: slug } as never} search={p.search as never} aria-label={t(m.editPage, { page: p.name })}>
                  {t(m.edit)}
                </Link>
              </div>
            ))}
          </div>
        ))}
      </div>
      <div className="ws-foot">
        <span className="k">{t(m.builtinNote)}</span>
      </div>
    </section>
  )
}

/** Forms & pages › Pages, until the page builder (M29) lists the workspace's own pages here. */
export function BuiltinPagesTab() {
  const t = useT()
  const { slug } = useWorkspace()
  const pages = useBuiltinPages()
  return (
    <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
      <BuiltinPagesCard pages={pages} />
      <Note tone="neutral" role="note">
        <T
          msg={m.themeNote}
          values={{
            link: (c) => (
              <Link to="/w/$workspace/forms/branding" params={{ workspace: slug }}>
                {c}
              </Link>
            ),
          }}
        />
      </Note>
    </Page>
  )
}
