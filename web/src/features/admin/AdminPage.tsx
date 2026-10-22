import type { ReactNode } from 'react'
import { useInstanceMeta } from '@/components/shell/AuthLayout'
import { Page } from '@/components/shell/Page'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { adminTabs } from '@/features/settings/SettingsTabs'
import { useT } from '@/i18n'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'

/** "reputecast.acme.io" from the public URL. */
export function hostOf(url?: string) {
  if (!url) return ''
  try {
    return new URL(url).host
  } catch {
    return url
  }
}

/**
 * The Instance admin frame (A5 §5.10) with its six section tabs. Everyone but
 * instance admins gets the 403 state.
 */
export function AdminPage({
  subtitle,
  documentTitle,
  back = 'settings',
  children,
}: {
  /** Replaces "Acme ReputeCast · reputecast.acme.io · v1.4.2". */
  subtitle?: ReactNode
  documentTitle: string
  /** The workspace button: settings, or the workspace audit log on Audit log. */
  back?: 'settings' | 'audit'
  children: ReactNode
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const { data: meta } = useInstanceMeta()
  const admin = me?.user.isInstanceAdmin ?? false
  const sub = subtitle ?? (meta ? [meta.instanceName, hostOf(meta.publicUrl), meta.version].filter(Boolean).join(' · ') : undefined)
  return (
    <Page
      title={t(m.title)}
      documentTitle={t(m.documentTitle, { page: documentTitle })}
      subtitle={admin ? sub : undefined}
      badges={admin ? <Badge tone="purple">{t(m.adminsOnly)}</Badge> : undefined}
      actions={
        admin ? (
          back === 'audit' ? (
            <ButtonLink to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
              {t(m.workspaceView)}
            </ButtonLink>
          ) : (
            <ButtonLink to="/w/$workspace/settings" params={{ workspace: slug }}>
              {t(m.workspaceSettings)}
            </ButtonLink>
          )
        ) : undefined
      }
      tabs={admin ? adminTabs(slug) : undefined}
      tabsLabel={t(m.tabsLabel)}
    >
      {me && !admin ? <NotAdmin /> : me ? children : null}
    </Page>
  )
}

/** The in-shell 403 for people who aren't instance admins. */
function NotAdmin() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <FullPageState
      icon="lockLarge"
      tone="warn"
      badge={t(m.notAdminBadge)}
      title={t(m.notAdminTitle)}
      actions={
        <>
          <ButtonLink variant="primary" to="/w/$workspace" params={{ workspace: slug }}>
            {t(m.backToOverview)}
          </ButtonLink>
          <ButtonLink to="/w/$workspace/account" params={{ workspace: slug }}>
            {t(m.account)}
          </ButtonLink>
        </>
      }
      footnote={t(m.notAdminFootnote)}
    >
      {t(m.notAdminBody)}
    </FullPageState>
  )
}
