import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { DisabledReason } from '@/components/ui/DisabledReason'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { deniedReason, useWorkspace } from '@/lib/session'
import type { BuiltinPage } from '@/features/website/BuiltinPages'
import { overviewKey, usePagesOverview, type Overview, type Shop } from './format'
import { m } from './messages'
import './pages.css'

/**
 * The shop's row in Built-in pages (PagesList): Shop · /products ·
 * Products · On and its switch (products:publish) · Edit, which opens
 * Earn › Products. Empty until Pages' overview has loaded.
 */
export function useShopBuiltinPages(): BuiltinPage[] {
  const t = useT()
  const shop = usePagesOverview().data?.shop
  if (!shop) return []
  return [
    {
      key: 'shop',
      name: t(m.shopName),
      addr: shop.path,
      group: t(m.shopGroup),
      state: t(shop.on ? m.on : m.off),
      control: <ShopSwitch shop={shop} />,
      to: '/w/$workspace/earn/products',
    },
  ]
}

/** On or off at once; the switch moves back and says why when the change fails. */
function ShopSwitch({ shop }: { shop: Shop }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [busy, setBusy] = useState(false)
  const reason = shop.canToggle ? undefined : deniedReason('products:publish', [], t(m.shopToggleAction))
  const change = async (on: boolean) => {
    if (reason || busy) return
    const key = overviewKey(slug)
    const before = qc.getQueryData<Overview>(key)
    const put = (s: Shop) => qc.setQueryData<Overview>(key, (o) => (o ? { ...o, shop: s } : o))
    put({ ...shop, on })
    setBusy(true)
    try {
      const { data } = await http.PUT('/api/v1/workspaces/{workspace}/pages/shop', { params: { path: { workspace: slug } }, body: { on } })
      if (data) put(data)
    } catch (e) {
      if (before) put(before.shop)
      toast({ kind: 'error', title: t(m.shopFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <DisabledReason reason={reason}>
      <input
        type="checkbox"
        role="switch"
        className="tg pg-tg"
        checked={shop.on}
        aria-label={t(m.shopToggle)}
        aria-busy={busy || undefined}
        onChange={(e) => void change(e.target.checked)}
      />
    </DisabledReason>
  )
}
