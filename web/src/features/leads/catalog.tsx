import { useState } from 'react'
import { Button, ButtonLink } from '@/components/ui/Button'
import { m } from '@/features/integrations/messages'
import type { Entry } from '@/features/integrations/shared'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { ConnectLeadAdsDialog } from './ConnectLeadAdsDialog'
import { ml } from './messages'

/*
 * The Lead ads group's Meta Lead Ads card (M66): Connect opens the connect
 * dialog, Manage the connection's page; the fact line names the
 * connection, its Pages and its leads in 30 days, or that leads are polled.
 */

export function LeadAdsAction({ entry: e, canManage }: { entry: Entry; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [open, setOpen] = useState(false)
  if (e.status === 'not_connected' || !e.connectionId) {
    if (!canManage) return null
    return (
      <>
        <Button size="sm" onClick={() => setOpen(true)}>
          {t(m.connect)}
        </Button>
        {open ? <ConnectLeadAdsDialog onOpenChange={setOpen} /> : null}
      </>
    )
  }
  return (
    <ButtonLink size="sm" to="/w/$workspace/settings/integrations/leads/$connectionId" params={{ workspace: slug, connectionId: e.connectionId }}>
      {canManage ? t(m.manage) : t(m.view)}
    </ButtonLink>
  )
}

/** "Acme Marketing · 2 Pages · 41 leads in 30 days", or "… · polling every 15 minutes". */
export function LeadAdsFact({ entry: e }: { entry: Entry }) {
  const t = useT()
  const f = e.facts.leads
  if (e.status === 'not_connected' || !f) return null
  if (f.polling && !f.problem) return <>{t(ml.factPolling, { name: f.name })}</>
  return <>{t(ml.factConnected, { name: f.name, pages: f.pages, leads: f.leads30d })}</>
}

/** Why the card needs attention: "Meta refused the token". */
export function leadAdsWhy(t: ReturnType<typeof useT>, e: Entry): string | undefined {
  const p = e.status === 'attention' ? e.facts.leads?.problem : undefined
  switch (p) {
    case 'token_rejected':
      return t(ml.whyRefused)
    case 'token_expired':
      return t(ml.whyExpired)
    case 'app_rejected':
      return t(ml.whyApp)
    case 'permission_missing':
      return t(ml.whyPermission)
  }
  return undefined
}
