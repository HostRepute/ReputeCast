import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import type { NavTab } from '@/components/ui/Tabs'
import { FullPageState, RequestAccessDialog } from '@/components/shell/SystemScreens'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { formatList, useT } from '@/i18n'
import { roleName } from '@/lib/roles'
import { useCan, useWorkspace } from '@/lib/session'
import { me } from './messages'
import './earn.css'

export type EarnTab = 'overview' | 'plans' | 'members' | 'gifts' | 'sponsors' | 'products' | 'settings'

export const EARN = '/api/v1/workspaces/{workspace}/earn' as const

/** The payment provider chip's data, shared by every Earn screen. */
export function useEarnStatus() {
  const { slug } = useWorkspace()
  const can = useCan('earn:view')
  return $api.useQuery('get', `${EARN}/status`, { params: { path: { workspace: slug } } }, { enabled: can, staleTime: 30_000 })
}

/**
 * One Earn screen: the title with the payment provider chip, the section
 * tabs (Overview · Plans & offers · Paid members · Gifts · Sponsors ·
 * Products · Settings) and, while only a test key is connected, the Test
 * mode banner. Without earn:view the section is a 403.
 */
export function EarnPage({ tab, title, actions, children, banner = true }: { tab: EarnTab; title?: string; actions?: ReactNode; children: ReactNode; banner?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('earn:view')
  const status = useEarnStatus()
  const params = { workspace: slug }
  // The overview opens the upgrade page from the top bar (EarnStates).
  const open =
    tab === 'overview' && !actions && status.data?.upgradeUrl ? (
      <ButtonAnchor href={status.data.upgradeUrl} iconRight="external">
        {t(me.openUpgrade)}
      </ButtonAnchor>
    ) : null
  const tabs: NavTab[] = [
    { label: t(me.tabOverview), to: '/w/$workspace/earn', params, exact: true, current: tab === 'overview' },
    { label: t(me.tabPlans), to: '/w/$workspace/earn/plans', params, current: tab === 'plans' },
    { label: t(me.tabMembers), to: '/w/$workspace/earn/members', params, current: tab === 'members' },
    { label: t(me.tabGifts), to: '/w/$workspace/earn/gifts', params, current: tab === 'gifts' },
    { label: t(me.tabSponsors), to: '/w/$workspace/earn/sponsors', params, current: tab === 'sponsors' },
    { label: t(me.tabProducts), to: '/w/$workspace/earn/products', params, current: tab === 'products' },
    { label: t(me.tabSettings), to: '/w/$workspace/earn/settings', params, current: tab === 'settings' },
  ]
  if (!can) {
    return (
      <Page title={t(me.title)}>
        <EarnForbidden />
      </Page>
    )
  }
  return (
    <Page title={t(me.title)} documentTitle={title ?? t(me.title)} badges={<ProviderChip status={status.data} />} tabs={tabs} tabsLabel={t(me.tabsLabel)} actions={actions ?? open}>
      {banner && status.data?.testOnly ? <TestModeBanner /> : null}
      {children}
    </Page>
  )
}

/** "● Stripe · Acme Media  Live": opens Settings › Payments; "No payment provider · Connect" before one is. */
export function ProviderChip({ status }: { status?: Schemas['EarnStatus'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (!status) return <span className="ea-chip" aria-hidden="true" />
  if (!status.mode) {
    return (
      <span className="ea-chip" aria-label={t(me.chipNoneLabel)}>
        <span className="dot" style={{ color: 'var(--ink-4)' }} aria-hidden="true" />
        {t(me.chipNone)}
        {status.canManage ? (
          <Link to="/w/$workspace/earn/connect" params={{ workspace: slug }} className="ea-chip-act">
            {status.setup ? t(me.chipResume) : t(me.chipConnect)}
          </Link>
        ) : null}
      </span>
    )
  }
  const tone = status.health === 'connected' || status.health === 'webhook_silent' ? (status.health === 'connected' ? 'ok' : 'warn') : 'bad'
  // A healthy test connection's dot is purple, like its Test badge.
  const color = tone === 'ok' && status.mode === 'test' ? '#7c3aed' : { ok: 'var(--ok-bar)', warn: 'var(--warn-bar)', bad: 'var(--bad-bar)' }[tone]
  return (
    <Link
      to="/w/$workspace/earn/settings"
      params={{ workspace: slug }}
      search={{ tab: 'payments' }}
      className="ea-chip"
      aria-label={t(me.chipLabel, { provider: status.providerName ?? 'Stripe', account: status.account ?? '', mode: status.mode })}
    >
      <span className="dot" style={{ color }} aria-hidden="true" />
      <span>
        {status.providerName} · {status.account}
      </span>
      <span className={`badge sm ${status.mode === 'live' ? 'ok' : 'pur'}`}>{status.mode === 'live' ? t(me.live) : t(me.test)}</span>
    </Link>
  )
}

/** "Test mode · Only test payments so far" while only a test key is connected (EarnStates). */
function TestModeBanner() {
  const t = useT()
  const { slug } = useWorkspace()
  const canManage = useCan('payments:manage')
  return (
    <section className="ea-testbar mb-4" role="status">
      <Badge tone="purple">{t(me.testBannerBadge)}</Badge>
      <div className="tx">
        <span className="ttl">{t(me.testBannerTitle)}</span>
        <span>{t(me.testBannerText)}</span>
      </div>
      {canManage ? (
        <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} search={{ mode: 'live' }} size="sm" variant="primary">
          {t(me.testBannerAction)}
        </ButtonLink>
      ) : null}
    </section>
  )
}

/**
 * 403 for members without earn:view (EarnStates, Daniel Park, Content
 * Writer): who to ask, what they can still do in drafts, and a request for
 * access that goes to the members who manage the team.
 */
function EarnForbidden() {
  const t = useT()
  const { slug, me: who } = useWorkspace()
  const canDraft = useCan('campaigns:edit')
  const [asking, setAsking] = useState(false)
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/access-requests/suggestion', {
    params: { path: { workspace: slug }, query: { permission: 'earn:view' } },
  })
  const s = q.data
  const role = roleName({ key: who.workspace.roleKey, name: who.workspace.roleName })
  const ask = s?.managers.length ? t(me.forbiddenAsk, { names: formatList(s.managers, 'disjunction') }) : ''
  return (
    <>
      <FullPageState
        icon="lockLarge"
        tone="warn"
        badge={t(me.forbiddenBadge, { role })}
        title={t(me.forbiddenTitle)}
        actions={
          <>
            {s?.requestedAt ? null : (
              <Button variant="primary" onClick={() => setAsking(true)} disabledReason={q.isPending ? t(me.loading) : undefined}>
                {t(me.forbiddenRequest)}
              </Button>
            )}
            <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
              {t(me.forbiddenOverview)}
            </ButtonLink>
          </>
        }
        footnote={s?.requestedAt ? t(me.forbiddenAsked) : undefined}
      >
        {[ask, canDraft ? t(me.forbiddenDrafts, { role }) : ''].filter(Boolean).join(' ')}
      </FullPageState>
      {s ? <RequestAccessDialog open={asking} onOpenChange={setAsking} permission="earn:view" roleName={roleName({ name: s.roleName })} managers={s.managers} /> : null}
    </>
  )
}

/** Copies a link and says so. */
export async function copyText(text: string, t: ReturnType<typeof useT>, toast: (x: { kind: 'success' | 'error'; title: string; detail?: string }) => void) {
  try {
    await navigator.clipboard.writeText(text)
    toast({ kind: 'success', title: t(me.copied), detail: text })
  } catch {
    toast({ kind: 'error', title: t(me.copyFailed), detail: text })
  }
}
