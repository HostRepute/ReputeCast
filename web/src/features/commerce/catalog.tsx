import { useState } from 'react'
import { Button, ButtonLink } from '@/components/ui/Button'
import { m } from '@/features/integrations/messages'
import type { Entry } from '@/features/integrations/shared'
import { useT } from '@/i18n'
import { ago } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ConnectStoreDialog } from './ConnectStoreDialog'
import { cm } from './messages'

type StorePlatform = 'shopify' | 'woocommerce'

/**
 * The Commerce card's action: Connect opens the store's connect dialog;
 * a store already started opens its page (where it stopped, while it's
 * connecting).
 */
export function StoreAction({ entry: e, platform, canManage }: { entry: Entry; platform: StorePlatform; canManage: boolean }) {
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
        {open ? <ConnectStoreDialog platform={platform} onOpenChange={setOpen} /> : null}
      </>
    )
  }
  return (
    <ButtonLink
      size="sm"
      to="/w/$workspace/settings/integrations/stores/$storeId"
      params={{ workspace: slug, storeId: e.connectionId }}
      search={e.status === 'waiting' && canManage ? { connect: true } : {}}
    >
      {canManage ? t(m.manage) : t(m.view)}
    </ButtonLink>
  )
}

/** The Commerce card's fact line: "Acme Merch · 412 orders in 30 days · last event 2 min ago". */
export function StoreFact({ entry: e, platform }: { entry: Entry; platform: StorePlatform }) {
  const t = useT()
  const f = e.facts.store
  if (e.status === 'not_connected' || !f) return <>{t(platform === 'shopify' ? cm.factShopify : cm.factWoo)}</>
  const name = f.name
  switch (f.state) {
    case 'connecting':
      return <>{t(cm.factConnecting, { name })}</>
    case 'importing':
      return <>{t(cm.factImporting, { name })}</>
    case 'webhook_missing':
      return <>{t(cm.factWebhookMissing, { name })}</>
    case 'rate_limited':
      return <>{t(cm.factRateLimited, { name, platform: t(platform === 'shopify' ? cm.shopify : cm.woocommerce) })}</>
    case 'no_plugin':
      return <>{t(cm.factNoPlugin, { name })}</>
    case 'revoked':
      return <>{t(cm.factRevoked, { name })}</>
  }
  if (f.stores > 1) return <>{t(cm.factStores, { count: f.stores, name })}</>
  return f.lastEventAt ? <>{t(cm.factStoreEvent, { name, orders: f.orders, ago: ago(f.lastEventAt) })}</> : <>{t(cm.factStore, { name, orders: f.orders })}</>
}
