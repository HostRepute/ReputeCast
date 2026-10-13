import type { ReactNode } from 'react'
import { useInstanceMeta } from '@/components/shell/AuthLayout'
import { Page } from '@/components/shell/Page'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { adminTabs } from '@/features/settings/SettingsTabs'
import { useMe, useWorkspace } from '@/lib/session'

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
 * The Instance admin frame (A5 §5.10): "Instance admin" with the instance,
 * host and version, the "admins only" badge, the way back to the workspace
 * and the six section tabs. Everyone but instance admins gets the 403 state.
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
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const { data: meta } = useInstanceMeta()
  const admin = me?.user.isInstanceAdmin ?? false
  const sub = subtitle ?? (meta ? [meta.instanceName, hostOf(meta.publicUrl), meta.version].filter(Boolean).join(' · ') : undefined)
  return (
    <Page
      title="Instance admin"
      documentTitle={`${documentTitle} · Instance admin`}
      subtitle={admin ? sub : undefined}
      badges={admin ? <Badge tone="purple">admins only</Badge> : undefined}
      actions={
        admin ? (
          back === 'audit' ? (
            <ButtonLink to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
              Workspace view
            </ButtonLink>
          ) : (
            <ButtonLink to="/w/$workspace/settings" params={{ workspace: slug }}>
              Workspace settings
            </ButtonLink>
          )
        ) : undefined
      }
      tabs={admin ? adminTabs(slug) : undefined}
      tabsLabel="Instance admin sections"
    >
      {me && !admin ? <NotAdmin /> : me ? children : null}
    </Page>
  )
}

/** The in-shell 403 for people who aren't instance admins. */
function NotAdmin() {
  const { slug } = useWorkspace()
  return (
    <FullPageState
      icon="lockLarge"
      tone="warn"
      badge="403 · instance admins only"
      title="Only instance admins can open Instance admin"
      actions={
        <>
          <ButtonLink variant="primary" to="/w/$workspace" params={{ workspace: slug }}>
            Back to overview
          </ButtonLink>
          <ButtonLink to="/w/$workspace/account" params={{ workspace: slug }}>
            Account &amp; security
          </ButtonLink>
        </>
      }
      footnote="Ask an instance admin for changes to sign-in, SSO or workspaces."
    >
      Instance admin manages users, sign-in methods, SSO, security policies and workspaces for the whole instance. Your own sign-in methods are in Account &amp;
      security.
    </FullPageState>
  )
}
