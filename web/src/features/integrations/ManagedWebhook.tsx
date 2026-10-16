import type { Schemas } from '@/api/client'
import { ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { ManagedBy } from './ManagedKeys'
import { m } from './messages'
import { isPlatform } from './shared'

type Managed = NonNullable<Schemas['WebhookView']['managedBy']>

/**
 * Under a managed endpoint on the Webhooks page (SettingsAPI): the workflow
 * that created it, "Managed by n8n", and why it can't be edited here.
 */
export function ManagedWebhookCaption({ managedBy, id }: { managedBy: Managed; id: string }) {
  const t = useT()
  return (
    <>
      <div className="k">
        {managedBy.workflow ? t(m.hookCaption, { kind: managedBy.kind, workflow: managedBy.workflow }) : t(m.hookCaptionPlain, { name: managedBy.name })}
      </div>
      <div className="row mt-1 gap-x-2 gap-y-1">
        <ManagedBy kind={managedBy.kind} />
        <span className="k whitespace-normal" id={id}>
          {t(m.hookReadOnly, { name: managedBy.name })}
        </span>
      </div>
    </>
  )
}

/** SettingsMCP: agents never connect integrations. */
export function McpIntegrationsNote() {
  const t = useT()
  return <span className="k">{t(m.mcpNoIntegrations)}</span>
}

/** "View": a managed endpoint opens its integration's page, where its webhooks and deliveries are. */
export function ManagedWebhookView({ managedBy, describedBy }: { managedBy: Managed; describedBy: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const kind = managedBy.kind
  return isPlatform(kind) ? (
    <ButtonLink size="sm" to="/w/$workspace/settings/integrations/$kind" params={{ workspace: slug, kind }} aria-describedby={describedBy}>
      {t(m.view)}
    </ButtonLink>
  ) : (
    <ButtonLink size="sm" to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} aria-describedby={describedBy}>
      {t(m.view)}
    </ButtonLink>
  )
}
